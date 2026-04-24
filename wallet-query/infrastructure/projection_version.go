package infrastructure

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectionDefinition struct {
	Name    string
	Version int
}

var DefinedProjectionVersions = []ProjectionDefinition{
	{
		Name:    "wallet_projection",
		Version: 1,
	},
}

type ProjectionVersion struct {
	ProjectionName string    `gorm:"primaryKey;column:projection_name;size:128"`
	Version        int       `gorm:"column:version"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (ProjectionVersion) TableName() string { return "projection_versions" }

type ProjectionVersionRepository struct {
	db *gorm.DB
}

func NewProjectionVersionRepository(db *gorm.DB) *ProjectionVersionRepository {
	return &ProjectionVersionRepository{db: db}
}

func (r *ProjectionVersionRepository) FindByName(name string) (*ProjectionVersion, error) {
	var record ProjectionVersion
	if err := r.db.First(&record, "projection_name = ?", name).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *ProjectionVersionRepository) Upsert(name string, version int, updatedAt time.Time) error {
	record := &ProjectionVersion{
		ProjectionName: name,
		Version:        version,
		UpdatedAt:      updatedAt,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "projection_name"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"version":    version,
			"updated_at": updatedAt,
		}),
	}).Create(record).Error
}

type ProjectionVersionManager struct {
	repository  *ProjectionVersionRepository
	definitions []ProjectionDefinition
}

func NewProjectionVersionManager(repository *ProjectionVersionRepository, definitions []ProjectionDefinition) *ProjectionVersionManager {
	return &ProjectionVersionManager{
		repository:  repository,
		definitions: definitions,
	}
}

func (m *ProjectionVersionManager) CheckStartup() error {
	for _, definition := range m.definitions {
		record, err := m.repository.FindByName(definition.Name)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				now := time.Now().UTC()
				if err := m.repository.Upsert(definition.Name, definition.Version, now); err != nil {
					return err
				}
				slog.Info("Projection version initialized",
					"component", "projection-version",
					"projectionName", definition.Name,
					"version", definition.Version,
					"updatedAt", now)
				continue
			}
			return err
		}

		slog.Info("Projection version detected",
			"component", "projection-version",
			"projectionName", definition.Name,
			"storedVersion", record.Version,
			"codeVersion", definition.Version,
			"updatedAt", record.UpdatedAt)

		if record.Version != definition.Version {
			slog.Warn("Projection version mismatch detected; rebuild required",
				"component", "projection-version",
				"projectionName", definition.Name,
				"storedVersion", record.Version,
				"codeVersion", definition.Version,
				"rebuildRequired", true)
		}
	}
	return nil
}

func (m *ProjectionVersionManager) MarkReplayComplete() error {
	now := time.Now().UTC()
	for _, definition := range m.definitions {
		if err := m.repository.Upsert(definition.Name, definition.Version, now); err != nil {
			return err
		}
		slog.Info("Projection version updated after replay",
			"component", "projection-version",
			"projectionName", definition.Name,
			"version", definition.Version,
			"updatedAt", now)
	}
	return nil
}
