package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/infrastructure"
	"gorm.io/gorm"
)

type DeadLetterListResponse struct {
	dto.BaseResponse
	DeadLetters []*DeadLetterDTO      `json:"deadLetters,omitempty"`
	Pagination  *PaginationMeta       `json:"pagination,omitempty"`
	Filters     *DeadLetterFilterMeta `json:"filters,omitempty"`
}

type DeadLetterDetailResponse struct {
	dto.BaseResponse
	DeadLetter *DeadLetterDTO `json:"deadLetter,omitempty"`
}

type DeadLetterReprocessResponse struct {
	dto.BaseResponse
	DeadLetter *DeadLetterDTO `json:"deadLetter,omitempty"`
}

type DeadLetterFilterMeta struct {
	Status      string `json:"status,omitempty"`
	EventType   string `json:"eventType,omitempty"`
	AggregateID string `json:"aggregateId,omitempty"`
	FailureKind string `json:"failureKind,omitempty"`
}

type DeadLetterKafkaMeta struct {
	Topic         string `json:"topic"`
	Partition     int    `json:"partition"`
	Offset        int64  `json:"offset"`
	ConsumerGroup string `json:"consumerGroup"`
}

type DeadLetterDTO struct {
	DeadLetterKey  string              `json:"deadLetterKey"`
	EventID        string              `json:"eventId"`
	EventType      string              `json:"eventType"`
	AggregateID    string              `json:"aggregateId"`
	Status         string              `json:"status"`
	FailureKind    string              `json:"failureKind"`
	RetryAttempts  int                 `json:"retryAttempts"`
	LastError      string              `json:"lastError"`
	Payload        string              `json:"payload"`
	Kafka          DeadLetterKafkaMeta `json:"kafka"`
	FirstFailedAt  time.Time           `json:"firstFailedAt"`
	LastFailedAt   time.Time           `json:"lastFailedAt"`
	DeadLetteredAt time.Time           `json:"deadLetteredAt"`
	ReprocessedAt  *time.Time          `json:"reprocessedAt,omitempty"`
	ResolvedAt     *time.Time          `json:"resolvedAt,omitempty"`
}

type deadLetterListParams struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	SortBy      string `form:"sortBy"`
	SortOrder   string `form:"sortOrder"`
	Status      string `form:"status"`
	EventType   string `form:"eventType"`
	AggregateID string `form:"aggregateId"`
	FailureKind string `form:"failureKind"`
}

func registerDeadLetterRoutes(r *gin.RouterGroup, repo *infrastructure.DeadLetterRepository, reprocessor *infrastructure.DeadLetterReprocessor) {
	r.GET("/dead-letters", getDeadLetters(repo))
	r.GET("/dead-letters/:id", getDeadLetterByID(repo))
	r.POST("/dead-letters/:id/reprocess", reprocessDeadLetter(repo, reprocessor))
}

func getDeadLetters(repo *infrastructure.DeadLetterRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := deadLetterListParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid dead-letter query parameters!"})
			return
		}

		query := queries.FindDeadLettersQuery{
			Page:        queries.NormalizePage(params.Page),
			PageSize:    queries.NormalizePageSize(params.PageSize),
			SortBy:      params.SortBy,
			SortOrder:   params.SortOrder,
			Status:      strings.TrimSpace(params.Status),
			EventType:   strings.TrimSpace(params.EventType),
			AggregateID: strings.TrimSpace(params.AggregateID),
			FailureKind: strings.TrimSpace(params.FailureKind),
		}
		query.SortBy, query.SortOrder = queries.NormalizeDeadLetterSort(query.SortBy, query.SortOrder)

		slog.Info("Dead-letter list requested",
			"component", "dead-letter-api",
			"page", query.Page,
			"pageSize", query.PageSize,
			"sortBy", query.SortBy,
			"sortOrder", query.SortOrder,
			"status", query.Status,
			"eventType", query.EventType,
			"aggregateId", query.AggregateID,
			"failureKind", query.FailureKind)

		records, err := repo.FindAll(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, DeadLetterListResponse{
				BaseResponse: dto.BaseResponse{Message: "Failed to complete dead-letter list request!"},
			})
			return
		}
		if len(records) == 0 {
			c.Status(http.StatusNoContent)
			return
		}

		items, hasMore := paginatedDeadLetters(records, query.PageSize)
		c.JSON(http.StatusOK, DeadLetterListResponse{
			BaseResponse: dto.BaseResponse{Message: "Successfully returned dead-letter events!"},
			DeadLetters:  toDeadLetterDTOs(items),
			Pagination: &PaginationMeta{
				Page:          query.Page,
				PageSize:      query.PageSize,
				ReturnedItems: len(items),
				HasMore:       hasMore,
				SortBy:        query.SortBy,
				SortOrder:     query.SortOrder,
			},
			Filters: &DeadLetterFilterMeta{
				Status:      query.Status,
				EventType:   query.EventType,
				AggregateID: query.AggregateID,
				FailureKind: query.FailureKind,
			},
		})
	}
}

func getDeadLetterByID(repo *infrastructure.DeadLetterRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		record, err := repo.FindByKey(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.Status(http.StatusNoContent)
				return
			}
			c.JSON(http.StatusInternalServerError, dto.BaseResponse{Message: "Failed to complete dead-letter detail request!"})
			return
		}

		slog.Info("Dead-letter detail requested",
			"component", "dead-letter-api",
			"deadLetterKey", record.DeadLetterKey,
			"eventId", record.EventID,
			"eventType", record.EventType,
			"aggregateId", record.AggregateID,
			"status", record.Status)

		c.JSON(http.StatusOK, DeadLetterDetailResponse{
			BaseResponse: dto.BaseResponse{Message: "Successfully returned dead-letter details!"},
			DeadLetter:   toDeadLetterDTO(record),
		})
	}
}

func reprocessDeadLetter(repo *infrastructure.DeadLetterRepository, reprocessor *infrastructure.DeadLetterReprocessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		record, err := repo.FindByKey(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, dto.BaseResponse{Message: "Dead-letter event not found!"})
				return
			}
			c.JSON(http.StatusInternalServerError, dto.BaseResponse{Message: "Failed to load dead-letter event for reprocessing!"})
			return
		}

		slog.Info("Dead-letter reprocess requested",
			"component", "dead-letter-api",
			"deadLetterKey", record.DeadLetterKey,
			"eventId", record.EventID,
			"eventType", record.EventType,
			"aggregateId", record.AggregateID,
			"status", record.Status)

		if err := reprocessor.Reprocess(c.Request.Context(), id); err != nil {
			slog.Error("Dead-letter reprocess request failed",
				"component", "dead-letter-api",
				"deadLetterKey", record.DeadLetterKey,
				"eventId", record.EventID,
				"eventType", record.EventType,
				"aggregateId", record.AggregateID,
				"error", err)
			refreshed, refreshErr := repo.FindByKey(id)
			if refreshErr == nil {
				c.JSON(http.StatusConflict, DeadLetterReprocessResponse{
					BaseResponse: dto.BaseResponse{Message: "Dead-letter reprocessing failed!"},
					DeadLetter:   toDeadLetterDTO(refreshed),
				})
				return
			}
			c.JSON(http.StatusConflict, dto.BaseResponse{Message: "Dead-letter reprocessing failed!"})
			return
		}

		refreshed, err := repo.FindByKey(id)
		if err != nil {
			c.JSON(http.StatusOK, DeadLetterReprocessResponse{
				BaseResponse: dto.BaseResponse{Message: "Dead-letter reprocessed successfully!"},
			})
			return
		}

		c.JSON(http.StatusOK, DeadLetterReprocessResponse{
			BaseResponse: dto.BaseResponse{Message: "Dead-letter reprocessed successfully!"},
			DeadLetter:   toDeadLetterDTO(refreshed),
		})
		slog.Info("Dead-letter reprocess request succeeded",
			"component", "dead-letter-api",
			"deadLetterKey", refreshed.DeadLetterKey,
			"eventId", refreshed.EventID,
			"eventType", refreshed.EventType,
			"aggregateId", refreshed.AggregateID,
			"status", refreshed.Status)
	}
}

func paginatedDeadLetters(records []*infrastructure.DeadLetterEvent, pageSize int) ([]*infrastructure.DeadLetterEvent, bool) {
	if len(records) <= pageSize {
		return records, false
	}
	return records[:pageSize], true
}

func toDeadLetterDTOs(records []*infrastructure.DeadLetterEvent) []*DeadLetterDTO {
	result := make([]*DeadLetterDTO, 0, len(records))
	for _, record := range records {
		result = append(result, toDeadLetterDTO(record))
	}
	return result
}

func toDeadLetterDTO(record *infrastructure.DeadLetterEvent) *DeadLetterDTO {
	if record == nil {
		return nil
	}
	return &DeadLetterDTO{
		DeadLetterKey: record.DeadLetterKey,
		EventID:       record.EventID,
		EventType:     record.EventType,
		AggregateID:   record.AggregateID,
		Status:        record.Status,
		FailureKind:   record.FailureKind,
		RetryAttempts: record.RetryAttempts,
		LastError:     record.LastError,
		Payload:       record.Payload,
		Kafka: DeadLetterKafkaMeta{
			Topic:         record.Topic,
			Partition:     record.Partition,
			Offset:        record.Offset,
			ConsumerGroup: record.ConsumerGroup,
		},
		FirstFailedAt:  record.FirstFailedAt,
		LastFailedAt:   record.LastFailedAt,
		DeadLetteredAt: record.DeadLetteredAt,
		ReprocessedAt:  record.ReprocessedAt,
		ResolvedAt:     record.ResolvedAt,
	}
}
