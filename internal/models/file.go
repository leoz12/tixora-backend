package models

import "time"

// FileStatus tracks whether an uploaded object has been linked to an
// owning entity yet, so unlinked (pending) uploads can be swept up later.
type FileStatus string

const (
	FileStatusPending  FileStatus = "pending"
	FileStatusAttached FileStatus = "attached"
)

// File is a generic registry of objects stored in R2 (or any future
// provider). It does not know which entity owns it - consumers (Event,
// User, ...) hold their own foreign key(s) into this table.
type File struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	ObjectKey    string     `gorm:"column:object_key;type:varchar(512);uniqueIndex;not null" json:"-"`
	OriginalName string     `gorm:"column:original_name;type:varchar(255)" json:"original_name"`
	Provider     string     `gorm:"type:varchar(50);not null;default:'r2'" json:"-"`
	MimeType     string     `gorm:"column:mime_type;type:varchar(100);not null" json:"mime_type"`
	Size         int64      `gorm:"not null" json:"size"`
	Status       FileStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (File) TableName() string {
	return "files"
}
