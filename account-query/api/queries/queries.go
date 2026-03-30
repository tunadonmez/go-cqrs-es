package queries

// FindAllAccountsQuery returns every account.
type FindAllAccountsQuery struct{}

func (q FindAllAccountsQuery) QueryTypeName() string { return "FindAllAccountsQuery" }

// FindAccountByIdQuery returns a single account by ID.
type FindAccountByIdQuery struct {
	ID string
}

func (q FindAccountByIdQuery) QueryTypeName() string { return "FindAccountByIdQuery" }

// FindAccountByHolderQuery returns the account for a given holder name.
type FindAccountByHolderQuery struct {
	AccountHolder string
}

func (q FindAccountByHolderQuery) QueryTypeName() string { return "FindAccountByHolderQuery" }

// EqualityType for balance comparisons.
type EqualityType string

const (
	GreaterThan EqualityType = "GREATER_THAN"
	LessThan    EqualityType = "LESS_THAN"
)

// FindAccountWithBalanceQuery returns accounts filtered by balance.
type FindAccountWithBalanceQuery struct {
	EqualityType EqualityType
	Balance      float64
}

func (q FindAccountWithBalanceQuery) QueryTypeName() string { return "FindAccountWithBalanceQuery" }
