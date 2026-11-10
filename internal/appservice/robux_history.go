package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// RobuxTransactionType names a kind of Robux transaction that the history lists.
type RobuxTransactionType string

const (
	RobuxPurchases       RobuxTransactionType = "purchases"
	RobuxSales           RobuxTransactionType = "sales"
	RobuxBought          RobuxTransactionType = "robuxBought"
	RobuxPremiumStipends RobuxTransactionType = "premiumStipends"
	RobuxGroupPayouts    RobuxTransactionType = "groupPayouts"
)

// Values of the transactionType parameter of the Roblox transaction records API.
var robloxTransactionTypes = map[RobuxTransactionType]string{
	RobuxPurchases:       "Purchase",
	RobuxSales:           "Sale",
	RobuxBought:          "CurrencyPurchase",
	RobuxPremiumStipends: "PremiumStipend",
	RobuxGroupPayouts:    "GroupPayout",
}

type RobuxTransaction struct {
	ID          string `json:"id"`
	CreatedAtMs int64  `json:"createdAtMs"`
	Amount      int64  `json:"amount"`
	IsPending   bool   `json:"isPending"`
	ItemName    string `json:"itemName"`
	ItemType    string `json:"itemType"`
	AgentName   string `json:"agentName"`
	PlaceName   string `json:"placeName"`
}

type RobuxTransactionPage struct {
	Transactions []RobuxTransaction `json:"transactions"`
	NextCursor   string             `json:"nextCursor"`
}

// GetRobuxTransactions reads one page of the account's Robux history.
func (service *Service) GetRobuxTransactions(ctx context.Context, accountID int64, transactionType RobuxTransactionType, cursor string) (RobuxTransactionPage, error) {
	robloxType, ok := robloxTransactionTypes[transactionType]
	if !ok {
		return RobuxTransactionPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "robux-transactions", Message: "Choose a transaction type."}
	}
	if len(cursor) > 4096 {
		return RobuxTransactionPage{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "robux-transactions", Message: "This page has expired. Refresh the history."}
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "robux-transactions")
	if err != nil {
		return RobuxTransactionPage{}, err
	}
	defer done()
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return RobuxTransactionPage{}, mapRepositoryError(err)
	}
	listedTransactions, nextCursor, err := service.users.Transactions(ctx, accountID, version, account.RobloxUserID, robloxType, cursor)
	if err != nil {
		service.logger.WarnContext(ctx, "robux history unavailable", "operation", "robux-transactions", "error", err)
		return RobuxTransactionPage{}, err
	}
	page := RobuxTransactionPage{Transactions: make([]RobuxTransaction, 0, len(listedTransactions)), NextCursor: nextCursor}
	for _, listedTransaction := range listedTransactions {
		transaction := RobuxTransaction{
			ID: listedTransaction.IDHash, Amount: listedTransaction.Amount, IsPending: listedTransaction.IsPending,
			ItemName: listedTransaction.ItemName, ItemType: listedTransaction.ItemType,
			AgentName: listedTransaction.AgentName, PlaceName: listedTransaction.PlaceName,
		}
		if !listedTransaction.CreatedAt.IsZero() {
			transaction.CreatedAtMs = listedTransaction.CreatedAt.UnixMilli()
		}
		page.Transactions = append(page.Transactions, transaction)
	}
	return page, nil
}
