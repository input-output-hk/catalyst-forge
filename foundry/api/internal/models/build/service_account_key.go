package build

import "time"

// ServiceAccountKey represents a public key bound to a service account
type ServiceAccountKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SAID      uint      `gorm:"index" json:"sa_id"`
	AKID      string    `gorm:"uniqueIndex;size:255" json:"akid"`
	Alg       string    `gorm:"size:32" json:"alg"`
	PubKeyB64 string    `gorm:"type:text" json:"pubkey_b64"`
	Status    string    `gorm:"size:32;default:active" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
