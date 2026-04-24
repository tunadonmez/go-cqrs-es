package queries

import (
	"strings"
	"time"

	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

// FindAllWalletsQuery returns every wallet.
type FindAllWalletsQuery struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	Currency  string
}

func (q FindAllWalletsQuery) QueryTypeName() string { return "FindAllWalletsQuery" }

// FindWalletByIDQuery returns a single wallet by ID.
type FindWalletByIDQuery struct {
	ID string
}

func (q FindWalletByIDQuery) QueryTypeName() string { return "FindWalletByIDQuery" }

// FindWalletTransactionsQuery returns transaction history for a wallet.
type FindWalletTransactionsQuery struct {
	WalletID     string
	Page         int
	PageSize     int
	SortBy       string
	SortOrder    string
	Type         dto.TransactionType
	OccurredFrom *time.Time
	OccurredTo   *time.Time
}

func (q FindWalletTransactionsQuery) QueryTypeName() string { return "FindWalletTransactionsQuery" }

// FindLedgerEntriesQuery returns ledger entries for the full ledger or one wallet.
type FindLedgerEntriesQuery struct {
	WalletID     string
	Page         int
	PageSize     int
	SortBy       string
	SortOrder    string
	EntryType    string
	EventType    string
	OccurredFrom *time.Time
	OccurredTo   *time.Time
}

func (q FindLedgerEntriesQuery) QueryTypeName() string { return "FindLedgerEntriesQuery" }

// FindDeadLettersQuery returns operational dead-letter rows.
type FindDeadLettersQuery struct {
	Page        int
	PageSize    int
	SortBy      string
	SortOrder   string
	Status      string
	EventType   string
	AggregateID string
	FailureKind string
}

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

func NormalizePage(page int) int {
	if page < 1 {
		return DefaultPage
	}
	return page
}

func NormalizePageSize(pageSize int) int {
	switch {
	case pageSize <= 0:
		return DefaultPageSize
	case pageSize > MaxPageSize:
		return MaxPageSize
	default:
		return pageSize
	}
}

func NormalizeWalletSort(sortBy, sortOrder string) (string, string) {
	switch sortBy {
	case "balance", "owner", "createdAt":
	default:
		sortBy = "createdAt"
	}
	return sortBy, normalizeSortOrder(sortOrder)
}

func NormalizeTransactionSort(sortBy, sortOrder string) (string, string) {
	switch sortBy {
	case "amount", "eventVersion", "occurredAt":
	default:
		sortBy = "occurredAt"
	}
	return sortBy, normalizeSortOrder(sortOrder)
}

func NormalizeDeadLetterSort(sortBy, sortOrder string) (string, string) {
	switch sortBy {
	case "updatedAt", "createdAt":
	default:
		sortBy = "createdAt"
	}
	return sortBy, normalizeSortOrder(sortOrder)
}

func NormalizeLedgerSort(sortBy, sortOrder string) (string, string) {
	switch sortBy {
	case "createdAt", "occurredAt":
	default:
		sortBy = "occurredAt"
	}
	return sortBy, normalizeSortOrder(sortOrder)
}

func normalizeSortOrder(sortOrder string) string {
	if strings.EqualFold(sortOrder, "desc") {
		return "desc"
	}
	return "asc"
}
