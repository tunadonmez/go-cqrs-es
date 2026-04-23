package queries

// FindAllWalletsQuery returns every wallet.
type FindAllWalletsQuery struct{}

func (q FindAllWalletsQuery) QueryTypeName() string { return "FindAllWalletsQuery" }

// FindWalletByIDQuery returns a single wallet by ID.
type FindWalletByIDQuery struct {
	ID string
}

func (q FindWalletByIDQuery) QueryTypeName() string { return "FindWalletByIDQuery" }

// FindWalletTransactionsQuery returns transaction history for a wallet.
type FindWalletTransactionsQuery struct {
	WalletID string
}

func (q FindWalletTransactionsQuery) QueryTypeName() string { return "FindWalletTransactionsQuery" }
