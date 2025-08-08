package user

import "time"

// RevokedJTI tracks denylisted access token IDs for immediate invalidation
type RevokedJTI struct {
	JTI       string    `gorm:"primaryKey" json:"jti"`
	Reason    string    `json:"reason"`
	RevokedAt time.Time `gorm:"not null;autoCreateTime" json:"revoked_at"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}

func (RevokedJTI) TableName() string { return "revoked_jtis" }
