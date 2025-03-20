package repository

import (
	"context"
	"errors"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"gorm.io/gorm"
)

// IDCounterRepository defines the interface for ID counter operations
type IDCounterRepository interface {
	GetNextID(ctx context.Context, project string, branch string) (string, error)
}

// GormIDCounterRepository implements IDCounterRepository using GORM
type GormIDCounterRepository struct {
	db *gorm.DB
}

// NewIDCounterRepository creates a new IDCounterRepository
func NewIDCounterRepository(db *gorm.DB) IDCounterRepository {
	return &GormIDCounterRepository{db: db}
}

// GetNextID retrieves and increments the counter for a project-branch combination
func (r *GormIDCounterRepository) GetNextID(ctx context.Context, project string, branch string) (string, error) {
	// Use a transaction to ensure atomicity when getting and updating the counter
	var nextID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var counter models.IDCounter

		result := tx.Where("project = ? AND branch = ?", project, branch).First(&counter)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				counter = models.IDCounter{
					Project: project,
					Branch:  branch,
					Counter: 0,
				}
				if err := tx.Create(&counter).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		}

		nextID = counter.GetNextID()
		return tx.Model(&counter).Update("counter", counter.Counter).Error
	})

	if err != nil {
		return "", err
	}

	return nextID, nil
}
