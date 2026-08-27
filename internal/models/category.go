package models

import "time"

type Category struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name      string    `gorm:"uniqueIndex;type:varchar(100);not null" json:"name"`
	Slug      string    `gorm:"uniqueIndex;type:varchar(100);not null" json:"slug"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Events []Event `gorm:"foreignKey:CategoryID" json:"-"`
}

func (Category) TableName() string {
	return "categories"
}
