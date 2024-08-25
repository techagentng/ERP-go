package models

import (
	"time"

	"gorm.io/gorm"
)

type MovieStatus string

const (
    Pending  MovieStatus = "Pending"
    Approved MovieStatus = "Approved"
)

type MovieBase struct {
	ID          uint           `gorm:"primaryKey"`
	Title       string         `gorm:"size:255"`
	Description string         `gorm:"type:text"`
	Duration    int            `gorm:"not null"` // Duration in minutes
	UploadedAt  time.Time      `gorm:"autoCreateTime"`
	Status      MovieStatus    `gorm:"type:varchar(20);default:'Pending'"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}