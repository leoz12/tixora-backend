package models

import "time"

// SubjectType distinguishes which principal a refresh token belongs to,
// since users and admins are separate tables with separate JWT claims.
type SubjectType string

const (
	SubjectTypeUser  SubjectType = "user"
	SubjectTypeAdmin SubjectType = "admin"
)

// RefreshToken is a rotated, revocable credential backing the 7-day session.
// Only the SHA-256 hash of the raw token is stored - the raw value only ever
// exists in the httpOnly cookie and the response to the client that issued it.
type RefreshToken struct {
	ID          string      `gorm:"primaryKey;type:varchar(36)" json:"id"`
	SubjectID   string      `gorm:"column:subject_id;type:varchar(36);not null;index:idx_refresh_tokens_subject" json:"subject_id"`
	SubjectType SubjectType `gorm:"column:subject_type;type:varchar(10);not null;index:idx_refresh_tokens_subject" json:"subject_type"`
	TokenHash   string      `gorm:"column:token_hash;type:varchar(64);uniqueIndex;not null" json:"-"`
	ExpiresAt   time.Time   `gorm:"column:expires_at;not null" json:"expires_at"`
	RevokedAt   *time.Time  `gorm:"column:revoked_at" json:"revoked_at"`
	ReplacedBy  *string     `gorm:"column:replaced_by;type:varchar(36)" json:"replaced_by"`
	UserAgent   string      `gorm:"column:user_agent;type:varchar(512)" json:"user_agent"`
	IPAddress   string      `gorm:"column:ip_address;type:varchar(64)" json:"ip_address"`
	CreatedAt   time.Time   `json:"created_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
