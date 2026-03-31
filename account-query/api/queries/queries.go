package queries

// FindAllAccountsQuery returns every account.
type FindAllAccountsQuery struct{}

func (q FindAllAccountsQuery) QueryTypeName() string { return "FindAllAccountsQuery" }

// FindAccountByIdQuery returns a single account by ID.
type FindAccountByIdQuery struct {
	ID string
}

func (q FindAccountByIdQuery) QueryTypeName() string { return "FindAccountByIdQuery" }
