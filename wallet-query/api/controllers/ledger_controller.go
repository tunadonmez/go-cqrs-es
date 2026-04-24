package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
)

type LedgerEntryListResponse struct {
	dto.BaseResponse
	Pagination    *PaginationMeta       `json:"pagination,omitempty"`
	Filters       *LedgerFilterMeta     `json:"filters,omitempty"`
	LedgerEntries []*domain.LedgerEntry `json:"ledgerEntries,omitempty"`
}

type LedgerFilterMeta struct {
	WalletID     string `json:"walletId,omitempty"`
	EntryType    string `json:"entryType,omitempty"`
	EventType    string `json:"eventType,omitempty"`
	OccurredFrom string `json:"occurredFrom,omitempty"`
	OccurredTo   string `json:"occurredTo,omitempty"`
}

type ledgerListParams struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	SortBy       string `form:"sortBy"`
	SortOrder    string `form:"sortOrder"`
	WalletID     string `form:"walletId"`
	AggregateID  string `form:"aggregateId"`
	EntryType    string `form:"entryType"`
	EventType    string `form:"eventType"`
	OccurredFrom string `form:"occurredFrom"`
	OccurredTo   string `form:"occurredTo"`
}

func registerLedgerRoutes(r *gin.RouterGroup, dispatcher *coreinfra.QueryDispatcher) {
	r.GET("/ledger-entries", getLedgerEntries(dispatcher))
	r.GET("/wallets/:id/ledger-entries", getWalletLedgerEntries(dispatcher))
}

func getLedgerEntries(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := ledgerListParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid ledger query parameters!"})
			return
		}
		query, ok := buildLedgerQuery("", params)
		if !ok {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid ledger query parameters!"})
			return
		}
		renderLedgerEntries(c, dispatcher, query)
	}
}

func getWalletLedgerEntries(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := ledgerListParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid wallet ledger query parameters!"})
			return
		}
		query, ok := buildLedgerQuery(c.Param("id"), params)
		if !ok {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid wallet ledger query parameters!"})
			return
		}
		renderLedgerEntries(c, dispatcher, query)
	}
}

func buildLedgerQuery(walletID string, params ledgerListParams) (queries.FindLedgerEntriesQuery, bool) {
	occurredFrom, err := parseOptionalTime(params.OccurredFrom)
	if err != nil {
		return queries.FindLedgerEntriesQuery{}, false
	}
	occurredTo, err := parseOptionalTimeWithBounds(params.OccurredTo, true)
	if err != nil {
		return queries.FindLedgerEntriesQuery{}, false
	}
	resolvedWalletID := strings.TrimSpace(walletID)
	if resolvedWalletID == "" {
		resolvedWalletID = strings.TrimSpace(params.WalletID)
	}
	if resolvedWalletID == "" {
		resolvedWalletID = strings.TrimSpace(params.AggregateID)
	}
	query := queries.FindLedgerEntriesQuery{
		WalletID:     resolvedWalletID,
		Page:         params.Page,
		PageSize:     params.PageSize,
		SortBy:       params.SortBy,
		SortOrder:    params.SortOrder,
		EntryType:    strings.ToUpper(strings.TrimSpace(params.EntryType)),
		EventType:    strings.TrimSpace(params.EventType),
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
	}
	if query.OccurredFrom != nil && query.OccurredTo != nil && query.OccurredFrom.After(*query.OccurredTo) {
		return queries.FindLedgerEntriesQuery{}, false
	}
	query.Page = queries.NormalizePage(query.Page)
	query.PageSize = queries.NormalizePageSize(query.PageSize)
	query.SortBy, query.SortOrder = queries.NormalizeLedgerSort(query.SortBy, query.SortOrder)
	switch query.EntryType {
	case "", domain.LedgerEntryTypeDebit, domain.LedgerEntryTypeCredit:
	default:
		return queries.FindLedgerEntriesQuery{}, false
	}
	return query, true
}

func renderLedgerEntries(c *gin.Context, dispatcher *coreinfra.QueryDispatcher, query queries.FindLedgerEntriesQuery) {
	entities, err := dispatcher.Send(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, LedgerEntryListResponse{
			BaseResponse: dto.BaseResponse{Message: "Failed to complete ledger request!"},
		})
		return
	}
	if len(entities) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	items, hasMore := paginatedLedgerEntries(entities, query.PageSize)
	c.JSON(http.StatusOK, LedgerEntryListResponse{
		BaseResponse:  dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d ledger entry(s)!", len(items))},
		LedgerEntries: items,
		Pagination: &PaginationMeta{
			Page:          query.Page,
			PageSize:      query.PageSize,
			ReturnedItems: len(items),
			HasMore:       hasMore,
			SortBy:        query.SortBy,
			SortOrder:     query.SortOrder,
		},
		Filters: &LedgerFilterMeta{
			WalletID:     query.WalletID,
			EntryType:    query.EntryType,
			EventType:    query.EventType,
			OccurredFrom: formatOptionalTime(query.OccurredFrom),
			OccurredTo:   formatOptionalTime(query.OccurredTo),
		},
	})
}

func toLedgerEntries(entities []coredomain.BaseEntity) []*domain.LedgerEntry {
	result := make([]*domain.LedgerEntry, 0, len(entities))
	for _, entity := range entities {
		if entry, ok := entity.(*domain.LedgerEntry); ok {
			result = append(result, entry)
		}
	}
	return result
}

func paginatedLedgerEntries(entities []coredomain.BaseEntity, pageSize int) ([]*domain.LedgerEntry, bool) {
	items := toLedgerEntries(entities)
	if len(items) <= pageSize {
		return items, false
	}
	return items[:pageSize], true
}
