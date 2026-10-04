package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
)

const logFormatVersion = 1

type diagnosticContextKey struct{}

func Diagnostic(logger *slog.Logger, message string, attributes ...any) {
	ctx := context.WithValue(context.Background(), diagnosticContextKey{}, true)
	logger.InfoContext(ctx, message, append([]any{"record_type", "component"}, attributes...)...)
}

func launchHeader(id string) ([]byte, error) {
	stamp, process, ok := strings.Cut(id, "-")
	started, err := strconv.ParseInt(stamp, 10, 64)
	pid, pidErr := strconv.Atoi(process)
	if !ok || err != nil || started <= 0 || strconv.FormatInt(started, 10) != stamp || pidErr != nil || pid <= 0 || strconv.Itoa(pid) != process {
		return nil, fmt.Errorf("invalid launch identifier")
	}
	header := struct {
		Time          int64         `json:"time"`
		Level         string        `json:"level"`
		Module        string        `json:"module"`
		Operation     string        `json:"operation"`
		Message       string        `json:"msg"`
		RecordType    string        `json:"record_type"`
		FormatVersion int           `json:"log_format_version"`
		LaunchID      string        `json:"launch_id"`
		PID           int           `json:"pid"`
		Build         appmeta.Build `json:"app"`
	}{
		Time: started, Level: "INFO", Module: "startup", Operation: "application-launch",
		Message: "launch log created", RecordType: "launch", FormatVersion: logFormatVersion,
		LaunchID: id, PID: pid, Build: appmeta.CurrentBuild(),
	}
	data, err := json.Marshal(header)
	return append(data, '\n'), err
}

func replaceLogAttribute(_ []string, attribute slog.Attr) slog.Attr {
	if attribute.Value.Kind() == slog.KindTime {
		attribute.Value = slog.Int64Value(attribute.Value.Time().UnixMilli())
	}
	if level, ok := attribute.Value.Any().(slog.Level); attribute.Key == slog.LevelKey && ok && level == traceSlogLevel {
		attribute.Value = slog.StringValue("TRACE")
	}
	return attribute
}
