package models

import "time"

type Event struct {
	ID               string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Title            string    `gorm:"type:varchar(255);not null" json:"title"`
	Description      string    `gorm:"type:text" json:"description"`
	EventDate        time.Time `gorm:"column:event_date;index;not null" json:"event_date"`
	Location         string    `gorm:"type:varchar(255)" json:"location"`
	ImageID          *string   `gorm:"column:image_id;type:varchar(36);index" json:"-"`
	Price            int64     `gorm:"not null" json:"price"` // in Rupiah
	TotalTickets     int       `gorm:"column:total_tickets;not null" json:"total_tickets"`
	AvailableTickets int       `gorm:"column:available_tickets;not null" json:"available_tickets"`
	CategoryID       string    `gorm:"column:category_id;type:varchar(36);index;not null" json:"category_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Relationships
	Category Category `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"category"`
	Image    *File    `gorm:"foreignKey:ImageID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	Orders   []Order  `gorm:"foreignKey:EventID" json:"-"`
}

func (Event) TableName() string {
	return "events"
}
