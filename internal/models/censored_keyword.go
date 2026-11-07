package models

import (
	"time"
)

// CensoredKeyword defines a sensitive keyword, secret phrase, or pattern to be censored globally
type CensoredKeyword struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Keyword     string    `gorm:"size:255;uniqueIndex;not null" json:"keyword"`
	Category    string    `gorm:"size:64;default:'custom'" json:"category"` // custom, api_key, token, password, domain, private_pattern
	Description string    `gorm:"size:512;default:''" json:"description"`
	IsRegex     bool      `gorm:"default:false" json:"is_regex"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
