package db

import (
	"errors"
	"fmt"

	"github.com/techagentng/telair-erp/models"
	"gorm.io/gorm"
)

type MovieRepository interface {
	CreateTrailer(trailer *models.Trailer) error
}

type movieRepo struct {
	DB *gorm.DB
}

func NewMovieRepo(db *GormDB) MovieRepository {
	return &movieRepo{db.DB}
}

func (r *movieRepo) CreateTrailer(trailer *models.Trailer) error {
	if trailer == nil {
		return errors.New("trailer cannot be nil")
	}

	// Save the trailer to the database
	if err := r.DB.Create(trailer).Error; err != nil {
		return fmt.Errorf("failed to create trailer: %w", err)
	}

	return nil
}
