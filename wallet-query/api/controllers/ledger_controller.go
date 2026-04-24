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

type LedgerMovementListResponse struct {
	dto.BaseResponse
	Pagination      *PaginationMeta           `json:"pagination,omitempty"`
	Filters         *LedgerMovementFilterMeta `json:"filters,omitempty"`
	LedgerMovements []*domain.LedgerMovement  `json:"ledgerMovements,omitempty"`
}

type LedgerMovementDetailResponse struct {
	dto.BaseResponse
	LedgerMovement *domain.LedgerMovement `json:"ledgerMovement,omitempty"`
}

type LedgerFilterMeta struct {
	MovementID   string `json:"movementId,omitempty"`
	WalletID     string `json:"walletId,omitempty"`
	EntryType    string `json:"entryType,omitempty"`
	EventType    string `json:"eventType,omitempty"`
	OccurredFrom string `json:"occurredFrom,omitempty"`
	OccurredTo   string `json:"occurredTo,omitempty"`
}

type LedgerMovementFilterMeta struct {
	WalletID     string `json:"walletId,omitempty"`
	MovementType string `json:"movementType,omitempty"`
	Status       string `json:"status,omitempty"`
	Reference    string `json:"reference,omitempty"`
	OccurredFrom string `json:"occurredFrom,omitempty"`
	OccurredTo   string `json:"occurredTo,omitempty"`
}

type ledgerListParams struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	SortBy       string `form:"sortBy"`
	SortOrder    string `form:"sortOrder"`
	MovementID   string `form:"movementId"`
	WalletID     string `form:"walletId"`
	AggregateID  string `form:"aggregateId"`
	EntryType    string `form:"entryType"`
	EventType    string `form:"eventType"`
	OccurredFrom string `form:"occurredFrom"`
	OccurredTo   string `form:"occurredTo"`
}

type ledgerMovementListParams struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	SortBy       string `form:"sortBy"`
	SortOrder    string `form:"sortOrder"`
	WalletID     string `form:"walletId"`
	MovementType string `form:"movementType"`
	Status       string `form:"status"`
	Reference    string `form:"reference"`
	OccurredFrom string `form:"occurredFrom"`
	OccurredTo   string `form:"occurredTo"`
}

func registerLedgerRoutes(r *gin.RouterGroup, dispatcher *coreinfra.QueryDispatcher) {
	r.GET("/ledger-entries", getLedgerEntries(dispatcher))
	r.GET("/wallets/:id/ledger-entries", getWalletLedgerEntries(dispatcher))
	r.GET("/ledger-movements", getLedgerMovements(dispatcher))
	r.GET("/ledger-movements/:id", getLedgerMovementByID(dispatcher))
	r.GET("/wallets/:id/ledger-movements", getWalletLedgerMovements(dispatcher))
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
		MovementID:   strings.TrimSpace(params.MovementID),
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
			MovementID:   query.MovementID,
			WalletID:     query.WalletID,
			EntryType:    query.EntryType,
			EventType:    query.EventType,
			OccurredFrom: formatOptionalTime(query.OccurredFrom),
			OccurredTo:   formatOptionalTime(query.OccurredTo),
		},
	})
}

func getLedgerMovements(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := ledgerMovementListParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid ledger movement query parameters!"})
			return
		}
		query, ok := buildLedgerMovementQuery("", params)
		if !ok {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid ledger movement query parameters!"})
			return
		}
		renderLedgerMovements(c, dispatcher, query)
	}
}

func getWalletLedgerMovements(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := ledgerMovementListParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid wallet ledger movement query parameters!"})
			return
		}
		query, ok := buildLedgerMovementQuery(c.Param("id"), params)
		if !ok {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid wallet ledger movement query parameters!"})
			return
		}
		renderLedgerMovements(c, dispatcher, query)
	}
}

func getLedgerMovementByID(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Ledger movement id is required!"})
			return
		}
		entities, err := dispatcher.Send(queries.FindLedgerMovementByIDQuery{ID: id})
		if err != nil || len(entities) == 0 {
			c.JSON(http.StatusNotFound, LedgerMovementDetailResponse{
				BaseResponse: dto.BaseResponse{Message: "Ledger movement not found!"},
			})
			return
		}
		movement, ok := entities[0].(*domain.LedgerMovement)
		if !ok {
			c.JSON(http.StatusInternalServerError, LedgerMovementDetailResponse{
				BaseResponse: dto.BaseResponse{Message: "Failed to decode ledger movement response!"},
			})
			return
		}
		c.JSON(http.StatusOK, LedgerMovementDetailResponse{
			BaseResponse:   dto.BaseResponse{Message: "Successfully returned ledger movement!"},
			LedgerMovement: movement,
		})
	}
}

func buildLedgerMovementQuery(walletID string, params ledgerMovementListParams) (queries.FindLedgerMovementsQuery, bool) {
	occurredFrom, err := parseOptionalTime(params.OccurredFrom)
	if err != nil {
		return queries.FindLedgerMovementsQuery{}, false
	}
	occurredTo, err := parseOptionalTimeWithBounds(params.OccurredTo, true)
	if err != nil {
		return queries.FindLedgerMovementsQuery{}, false
	}
	resolvedWalletID := strings.TrimSpace(walletID)
	if resolvedWalletID == "" {
		resolvedWalletID = strings.TrimSpace(params.WalletID)
	}
	query := queries.FindLedgerMovementsQuery{
		WalletID:     resolvedWalletID,
		Page:         params.Page,
		PageSize:     params.PageSize,
		SortBy:       params.SortBy,
		SortOrder:    params.SortOrder,
		MovementType: strings.ToUpper(strings.TrimSpace(params.MovementType)),
		Status:       strings.ToUpper(strings.TrimSpace(params.Status)),
		Reference:    strings.TrimSpace(params.Reference),
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
	}
	if query.OccurredFrom != nil && query.OccurredTo != nil && query.OccurredFrom.After(*query.OccurredTo) {
		return queries.FindLedgerMovementsQuery{}, false
	}
	query.Page = queries.NormalizePage(query.Page)
	query.PageSize = queries.NormalizePageSize(query.PageSize)
	query.SortBy, query.SortOrder = queries.NormalizeLedgerMovementSort(query.SortBy, query.SortOrder)
	switch query.MovementType {
	case "", domain.LedgerMovementTypeOpeningBalance, domain.LedgerMovementTypeCredit, domain.LedgerMovementTypeDebit, domain.LedgerMovementTypeTransfer:
	default:
		return queries.FindLedgerMovementsQuery{}, false
	}
	switch query.Status {
	case "", domain.LedgerMovementStatusPending, domain.LedgerMovementStatusCompleted, domain.LedgerMovementStatusInconsistent:
	default:
		return queries.FindLedgerMovementsQuery{}, false
	}
	return query, true
}

func renderLedgerMovements(c *gin.Context, dispatcher *coreinfra.QueryDispatcher, query queries.FindLedgerMovementsQuery) {
	entities, err := dispatcher.Send(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, LedgerMovementListResponse{
			BaseResponse: dto.BaseResponse{Message: "Failed to complete ledger movement request!"},
		})
		return
	}
	if len(entities) == 0 {
		c.Status(http.StatusNoContent)
		return
	}
	items, hasMore := paginatedLedgerMovements(entities, query.PageSize)
	c.JSON(http.StatusOK, LedgerMovementListResponse{
		BaseResponse:    dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d ledger movement(s)!", len(items))},
		LedgerMovements: items,
		Pagination: &PaginationMeta{
			Page:          query.Page,
			PageSize:      query.PageSize,
			ReturnedItems: len(items),
			HasMore:       hasMore,
			SortBy:        query.SortBy,
			SortOrder:     query.SortOrder,
		},
		Filters: &LedgerMovementFilterMeta{
			WalletID:     query.WalletID,
			MovementType: query.MovementType,
			Status:       query.Status,
			Reference:    query.Reference,
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

func toLedgerMovements(entities []coredomain.BaseEntity) []*domain.LedgerMovement {
	result := make([]*domain.LedgerMovement, 0, len(entities))
	for _, entity := range entities {
		if movement, ok := entity.(*domain.LedgerMovement); ok {
			result = append(result, movement)
		}
	}
	return result
}

func paginatedLedgerMovements(entities []coredomain.BaseEntity, pageSize int) ([]*domain.LedgerMovement, bool) {
	items := toLedgerMovements(entities)
	if len(items) <= pageSize {
		return items, false
	}
	return items[:pageSize], true
}
