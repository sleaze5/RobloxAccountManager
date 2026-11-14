package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// robloxCurrencyRobux is the expectedCurrency of a purchase in Robux.
const robloxCurrencyRobux = 1

// The purchase flow follows the Buy button of www.roblox.com, as Bonsai
// recorded it on 2026-10-14: the purchase sends the price that the user
// confirmed, and Roblox refuses it when the price changed. The purchases were
// not sent live, because they spend Robux.

// BuyInventoryItem buys a new copy of the item from its creator for the price
// that the user confirmed, and reports whether Roblox still delivers it.
func (service *Users) BuyInventoryItem(ctx context.Context, accountID, version, userID int64, category InventoryCategory, itemID, expectedPriceInRobux int64) (bool, error) {
	itemType, ok := inventoryItemTypeOf(category)
	if !ok || itemType == inventoryItemBadge {
		return false, purchaseRefusal("Only catalog items and game passes can be bought.")
	}
	item := strconv.FormatInt(itemID, 10)
	if itemType == inventoryItemGamePass {
		return service.buyGamePass(ctx, accountID, version, item, expectedPriceInRobux)
	}
	var details InventoryItemDetails
	var err error
	if itemType == inventoryItemBundle {
		details, err = service.bundleItemDetails(ctx, accountID, version, item)
	} else {
		details, err = service.assetItemDetails(ctx, accountID, version, item)
	}
	if err != nil {
		return false, err
	}
	if details.CollectibleItemID == "" {
		return false, purchaseRefusal("Buy this item on the Roblox website.")
	}
	return service.buyCollectibleItem(ctx, accountID, version, userID, details.CollectibleItemID, expectedPriceInRobux)
}

func (service *Users) buyGamePass(ctx context.Context, accountID, version int64, gamePassID string, expectedPriceInRobux int64) (bool, error) {
	var productPayload struct {
		ProductID    int64  `json:"ProductId"`
		IsForSale    bool   `json:"IsForSale"`
		PriceInRobux *int64 `json:"PriceInRobux"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "inventory-game-pass-details", "https://apis.roblox.com/game-passes/v1/game-passes/"+gamePassID+"/product-info", &productPayload); err != nil {
		return false, err
	}
	var currentPrice *int64
	if productPayload.IsForSale {
		currentPrice = productPayload.PriceInRobux
	}
	if err := checkPurchasePrice(currentPrice, expectedPriceInRobux); err != nil {
		return false, err
	}
	if productPayload.ProductID <= 0 {
		return false, purchaseRefusal("Buy this game pass on the Roblox website.")
	}
	// The path takes the product ID of the pass, not the pass ID.
	address := "https://apis.roblox.com/game-passes/v1/game-passes/" + strconv.FormatInt(productPayload.ProductID, 10) + "/purchase"
	response, err := service.writeAccount(ctx, accountID, version, "inventory-game-pass-purchase", http.MethodPost, address, map[string]any{
		"expectedPrice": expectedPriceInRobux, "idempotencyKey": randomUUID(),
	})
	if err != nil {
		return false, err
	}
	return readPurchaseOutcome(response, "inventory-game-pass-purchase")
}

type collectibleSale struct {
	CollectibleItemID    string  `json:"collectibleItemId"`
	CollectibleProductID string  `json:"collectibleProductId"`
	CreatorID            int64   `json:"creatorId"`
	CreatorType          string  `json:"creatorType"`
	Price                *int64  `json:"price"`
	UnitsAvailable       *int64  `json:"unitsAvailableForConsumption"`
	AssetStock           *int64  `json:"assetStock"`
	SaleLocationType     string  `json:"saleLocationType"`
	LowestResaleInstance *string `json:"lowestAvailableResaleItemInstanceId"`
}

func (service *Users) buyCollectibleItem(ctx context.Context, accountID, version, userID int64, collectibleItemID string, expectedPriceInRobux int64) (bool, error) {
	sale, err := service.readCollectibleSale(ctx, accountID, version, collectibleItemID)
	if err != nil {
		return false, err
	}
	if sale.UnitsAvailable != nil && *sale.UnitsAvailable == 0 && sale.AssetStock != nil && *sale.AssetStock > 0 {
		return false, purchaseRefusal("This item is sold out.")
	}
	if saleLocationOf(sale.SaleLocationType) == SaleInExperiences {
		return false, purchaseRefusal("This item is only sold inside experiences.")
	}
	if err := checkPurchasePrice(sale.Price, expectedPriceInRobux); err != nil {
		return false, err
	}
	if sale.CollectibleProductID == "" || !validGameCreator(sale.CreatorID, sale.CreatorType) {
		return false, purchaseRefusal("Buy this item on the Roblox website.")
	}
	address := "https://apis.roblox.com/marketplace-sales/v1/item/" + url.PathEscape(collectibleItemID) + "/purchase-item"
	response, err := service.writeAccount(ctx, accountID, version, "inventory-item-purchase", http.MethodPost, address, map[string]any{
		"collectibleItemId": collectibleItemID, "collectibleProductId": sale.CollectibleProductID,
		"expectedCurrency": robloxCurrencyRobux, "expectedPrice": expectedPriceInRobux,
		"expectedPurchaserId": userID, "expectedPurchaserType": "User",
		"expectedSellerId": sale.CreatorID, "expectedSellerType": sale.CreatorType,
		"idempotencyKey": randomUUID(),
	})
	if err != nil {
		return false, err
	}
	return readPurchaseOutcome(response, "inventory-item-purchase")
}

// readCollectibleSale reads how the creator sells the item now. Roblox
// requires a CSRF token although the request reads only. Verified on
// 2026-11-14.
func (service *Users) readCollectibleSale(ctx context.Context, accountID, version int64, collectibleItemID string) (collectibleSale, error) {
	body, err := json.Marshal(map[string][]string{"itemIds": {collectibleItemID}})
	if err != nil {
		return collectibleSale{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-item-sale", Message: "The sale request could not be created.", Cause: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "inventory-item-sale", AccountID: accountID, ExpectedSecretVersion: version,
		Authenticated: true, Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: "apis.roblox.com", Path: "/marketplace-items/v1/items/details"},
		Body: body, ContentType: "application/json", RequiresCSRF: true, MaxResponseSize: 64 << 10,
		Retry: roblox.RetryPolicy{MaxAttempts: 2, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return collectibleSale{}, err
	}
	var payload []collectibleSale
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return collectibleSale{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "inventory-item-sale", Status: response.Status, Message: "Roblox returned invalid sale data.", Cause: err}
	}
	for _, sale := range payload {
		if sale.CollectibleItemID == collectibleItemID {
			return sale, nil
		}
	}
	return collectibleSale{}, purchaseRefusal("This item can't be bought.")
}

func checkPurchasePrice(currentPriceInRobux *int64, expectedPriceInRobux int64) error {
	switch {
	case currentPriceInRobux == nil:
		return purchaseRefusal("This item is no longer for sale.")
	case *currentPriceInRobux != expectedPriceInRobux:
		price := "free"
		if *currentPriceInRobux > 0 {
			price = "R$ " + strconv.FormatInt(*currentPriceInRobux, 10)
		}
		return purchaseRefusal("The price changed to " + price + ". Check it before you buy.")
	}
	return nil
}

// readPurchaseOutcome reads the answer of either purchase endpoint. Roblox
// answers 200 for a refused purchase and names the reason in the body, in
// the order of checks that Bonsai took from the website.
func readPurchaseOutcome(response *roblox.Response, endpoint string) (bool, error) {
	var payload struct {
		Purchased               bool   `json:"purchased"`
		Pending                 bool   `json:"pending"`
		Reason                  string `json:"reason"`
		ErrorMessage            string `json:"errorMessage"`
		PurchaseResult          string `json:"purchaseResult"`
		FailureReason           string `json:"failureReason"`
		ExpirationTimeInMinutes int64  `json:"expirationTimeInMinutes"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return false, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Status: response.Status, Message: "Roblox returned an invalid purchase result.", Cause: err}
	}
	if payload.FailureReason != "" {
		if payload.ExpirationTimeInMinutes > 0 {
			return false, purchaseRefusal("Roblox paused purchases on this account for " + strconv.FormatInt(payload.ExpirationTimeInMinutes, 10) + " minutes.")
		}
		return false, purchaseRefusal("Roblox paused purchases on this account for now.")
	}
	if payload.Purchased {
		return payload.Pending, nil
	}
	reasons := []string{payload.Reason, payload.ErrorMessage, payload.PurchaseResult}
	switch {
	case slices.Contains(reasons, "TwoStepVerificationRequired"):
		return false, purchaseRefusal("Roblox wants to verify this purchase. Buy it on the Roblox website.")
	case slices.Contains(reasons, "InsufficientBalance") || slices.Contains(reasons, "InsufficientFunds"):
		return false, purchaseRefusal("This account doesn't have enough Robux.")
	case slices.Contains(reasons, "Flooded"):
		return false, purchaseRefusal("Too many purchases. Wait a minute and try again.")
	case slices.Contains(reasons, "AlreadyOwned"):
		return false, purchaseRefusal("This account already owns this item.")
	case slices.Contains(reasons, "QuantityLimitExceeded"):
		return false, purchaseRefusal("This account owns as many copies as Roblox allows.")
	}
	return false, purchaseRefusal("Roblox didn't complete the purchase.")
}

func purchaseRefusal(message string) error {
	return &roblox.Error{Kind: roblox.KindForbidden, Endpoint: "inventory-item-purchase", Message: message}
}
