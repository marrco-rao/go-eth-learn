package model

import "time"

type Log struct {
	ID         uint   `gorm:"primaryKey"`
	Level      string `gorm:"size:16;not null"`
	Message    string `gorm:"type:text;not null"`
	Method     string `gorm:"size:16;not null"`
	Path       string `gorm:"size:255;not null"`
	StatusCode int    `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
