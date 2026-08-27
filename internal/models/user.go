package models

import "time"

type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Email     string    `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	GoogleID  string    `gorm:"uniqueIndex;column:google_id;type:varchar(255)" json:"google_id"`
	AvatarURL string    `gorm:"column:avatar_url;type:varchar(512)" json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Orders []Order `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string {
	return "users"
}
