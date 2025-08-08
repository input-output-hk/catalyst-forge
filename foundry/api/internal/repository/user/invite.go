package user

import (
	"time"

	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

type InviteRepository interface {
	Create(inv *dbmodel.Invite) error
	GetByID(id uint) (*dbmodel.Invite, error)
	GetByTokenHash(hash string) (*dbmodel.Invite, error)
	MarkRedeemed(id uint) error
}

type inviteRepository struct {
	db *gorm.DB
}

func NewInviteRepository(db *gorm.DB) InviteRepository { return &inviteRepository{db: db} }

func (r *inviteRepository) Create(inv *dbmodel.Invite) error { return r.db.Create(inv).Error }

func (r *inviteRepository) GetByID(id uint) (*dbmodel.Invite, error) {
	var out dbmodel.Invite
	tx := r.db.First(&out, id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &out, nil
}

func (r *inviteRepository) GetByTokenHash(hash string) (*dbmodel.Invite, error) {
	var out dbmodel.Invite
	tx := r.db.First(&out, "token_hash = ?", hash)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &out, nil
}

func (r *inviteRepository) MarkRedeemed(id uint) error {
	now := time.Now()
	tx := r.db.Model(&dbmodel.Invite{}).Where("id = ?", id).Update("redeemed_at", &now)
	return tx.Error
}
