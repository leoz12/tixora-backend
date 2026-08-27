package models

import "time"

type Admin struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Email        string    `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(50);not null;default:admin" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Admin) TableName() string {
	return "admins"
}
