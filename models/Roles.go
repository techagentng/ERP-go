package models

type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:255;unique;not null"`
}
