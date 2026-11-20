package services

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox/modelfile"
)

const (
	studioVersionURL       = "https://setup.rbxcdn.com/versionQTStudio"
	maxStudioAPIDumpBytes  = 16 << 20
	maxSuperclassDepth     = 64
	classIndexRetryDelay   = time.Minute
	ungroupedPropertyGroup = "Properties"
	// hiddenPropertyGroup holds the properties that files store but Roblox
	// Studio does not show, such as ScriptGuid.
	hiddenPropertyGroup = "Hidden"
)

// studioVersionPattern accepts only a version name, so that it can only name a
// file at the root of the setup host.
var studioVersionPattern = regexp.MustCompile(`^version-[0-9a-f]{16}$`)

// studioPropertyGroupOrder is the order of the property groups that Roblox
// Studio shows first. Other groups follow in alphabetical order, then the
// attributes, then the hidden properties.
var studioPropertyGroupOrder = []string{"Appearance", "Data", "Text", "Image", "Transform", "Pivot", "Behavior", "Collision", "Part"}

// classProperty is what Roblox Studio knows about one property: its group,
// and the enum of its value, if it has one.
type classProperty struct {
	group    string
	enumName string
}

// classIndex is the part of the Roblox API dump that the explorer reads.
type classIndex struct {
	superclassByClass   map[string]string
	propertiesByClass   map[string]map[string]classProperty
	enumItemNamesByEnum map[string]map[uint32]string
}

// property finds a property of the class or of one of its superclasses.
func (index *classIndex) property(className, propertyName string) (classProperty, bool) {
	for range maxSuperclassDepth {
		if property, isKnown := index.propertiesByClass[className][propertyName]; isKnown {
			return property, true
		}
		superclass, hasSuperclass := index.superclassByClass[className]
		if !hasSuperclass || superclass == "" {
			return classProperty{}, false
		}
		className = superclass
	}
	return classProperty{}, false
}

// ClassReflection reads the classes and enums of Roblox from the API dump of
// the current Roblox Studio release, which names the groups of properties and
// the items of enums as Studio shows them. The dump changes only with a
// Studio release, so it is read once per run.
type ClassReflection struct {
	client *roblox.Client
	mutex  sync.Mutex
	index  *classIndex
	// retryAt delays the next read after a failure, so that every explorer
	// that opens while Roblox is unreachable does not download the dump again.
	retryAt time.Time
	// lastError is the failure that retryAt delays the next read after.
	lastError error
}

func NewClassReflection(client *roblox.Client) *ClassReflection {
	return &ClassReflection{client: client}
}

// GroupProperties sorts the properties of each instance of the models into
// the groups that Roblox Studio shows, names the items of enums, and orders
// the properties by group and name. When the API dump cannot be read, it puts
// the properties in one group and returns why.
func (reflection *ClassReflection) GroupProperties(ctx context.Context, models []ExploredModel) error {
	index, err := reflection.loadIndex(ctx)
	for modelIndex := range models {
		groupInstanceProperties(index, models[modelIndex].Model.Instances)
	}
	return err
}

func groupInstanceProperties(index *classIndex, instances []modelfile.Instance) {
	for instanceIndex := range instances {
		instance := &instances[instanceIndex]
		for propertyIndex := range instance.Properties {
			property := &instance.Properties[propertyIndex]
			if property.Group == modelfile.AttributesGroup {
				continue
			}
			if index == nil {
				property.Group = ungroupedPropertyGroup
				continue
			}
			studioProperty, isKnown := index.property(instance.ClassName, property.Name)
			if !isKnown {
				property.Group = hiddenPropertyGroup
				continue
			}
			property.Group = cmp.Or(studioProperty.group, "Data")
			if property.Kind == modelfile.PropertyEnum {
				if itemName, isNamed := index.enumItemNamesByEnum[studioProperty.enumName][property.EnumValue]; isNamed {
					property.Text = itemName
				}
			}
		}
		slices.SortStableFunc(instance.Properties, func(first, second modelfile.Property) int {
			return cmp.Or(comparePropertyGroups(first.Group, second.Group), cmp.Compare(first.Name, second.Name))
		})
		groupInstanceProperties(index, instance.Children)
	}
}

func comparePropertyGroups(first, second string) int {
	rank := func(group string) int {
		switch group {
		case modelfile.AttributesGroup:
			return len(studioPropertyGroupOrder) + 1
		case hiddenPropertyGroup:
			return len(studioPropertyGroupOrder) + 2
		}
		if position := slices.Index(studioPropertyGroupOrder, group); position >= 0 {
			return position
		}
		return len(studioPropertyGroupOrder)
	}
	return cmp.Or(cmp.Compare(rank(first), rank(second)), cmp.Compare(first, second))
}

func (reflection *ClassReflection) loadIndex(ctx context.Context) (*classIndex, error) {
	reflection.mutex.Lock()
	defer reflection.mutex.Unlock()
	if reflection.index != nil {
		return reflection.index, nil
	}
	if time.Now().Before(reflection.retryAt) {
		return nil, reflection.lastError
	}
	index, err := reflection.readIndex(ctx)
	if err != nil {
		if ctx.Err() == nil {
			reflection.retryAt = time.Now().Add(classIndexRetryDelay)
			reflection.lastError = err
		}
		return nil, err
	}
	reflection.index = index
	return index, nil
}

func (reflection *ClassReflection) readIndex(ctx context.Context) (*classIndex, error) {
	versionBody, err := reflection.download(ctx, "studio-version", studioVersionURL, 1<<10)
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(string(versionBody))
	if !studioVersionPattern.MatchString(version) {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "studio-version", Message: "Roblox returned an invalid Studio version."}
	}
	dump, err := reflection.download(ctx, "studio-api-dump", "https://setup.rbxcdn.com/"+version+"-Full-API-Dump.json", maxStudioAPIDumpBytes)
	if err != nil {
		return nil, err
	}
	index, err := decodeAPIDump(dump)
	if err != nil {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "studio-api-dump", Message: "Roblox returned an invalid list of classes.", Cause: err}
	}
	return index, nil
}

func (reflection *ClassReflection) download(ctx context.Context, endpoint, address string, maxResponseSizeBytes int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	addressURL, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse %s address: %w", endpoint, err)
	}
	response, err := reflection.client.Do(ctx, roblox.Request{
		Endpoint: endpoint, Method: http.MethodGet, URL: addressURL, MaxResponseSize: maxResponseSizeBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 2, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func decodeAPIDump(dump []byte) (*classIndex, error) {
	var payload struct {
		Classes []struct {
			Name       string `json:"Name"`
			Superclass string `json:"Superclass"`
			Members    []struct {
				MemberType string `json:"MemberType"`
				Name       string `json:"Name"`
				Category   string `json:"Category"`
				ValueType  struct {
					Category string `json:"Category"`
					Name     string `json:"Name"`
				} `json:"ValueType"`
			} `json:"Members"`
		} `json:"Classes"`
		Enums []struct {
			Name  string `json:"Name"`
			Items []struct {
				Name  string `json:"Name"`
				Value uint32 `json:"Value"`
			} `json:"Items"`
		} `json:"Enums"`
	}
	if err := json.Unmarshal(dump, &payload); err != nil {
		return nil, fmt.Errorf("decode API dump: %w", err)
	}
	index := &classIndex{
		superclassByClass:   make(map[string]string, len(payload.Classes)),
		propertiesByClass:   make(map[string]map[string]classProperty, len(payload.Classes)),
		enumItemNamesByEnum: make(map[string]map[uint32]string, len(payload.Enums)),
	}
	for _, listedClass := range payload.Classes {
		index.superclassByClass[listedClass.Name] = listedClass.Superclass
		properties := map[string]classProperty{}
		for _, member := range listedClass.Members {
			if member.MemberType != "Property" {
				continue
			}
			property := classProperty{group: member.Category}
			if member.ValueType.Category == "Enum" {
				property.enumName = member.ValueType.Name
			}
			properties[member.Name] = property
		}
		index.propertiesByClass[listedClass.Name] = properties
	}
	for _, listedEnum := range payload.Enums {
		itemNames := make(map[uint32]string, len(listedEnum.Items))
		for _, listedItem := range listedEnum.Items {
			if _, isNamed := itemNames[listedItem.Value]; !isNamed {
				itemNames[listedItem.Value] = listedItem.Name
			}
		}
		index.enumItemNamesByEnum[listedEnum.Name] = itemNames
	}
	if len(index.propertiesByClass) == 0 {
		return nil, errors.New("API dump lists no classes")
	}
	return index, nil
}
