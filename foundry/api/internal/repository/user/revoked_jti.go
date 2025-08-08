package user

import (
    dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
    "gorm.io/gorm"
)

type RevokedJTIRepository interface {
    IsRevoked(jti string) (bool, error)
}

type revokedJTIRepository struct {
    db *gorm.DB
}

func NewRevokedJTIRepository(db *gorm.DB) RevokedJTIRepository {
    return &revokedJTIRepository{db: db}
}

func (r *revokedJTIRepository) IsRevoked(jti string) (bool, error) {
    if jti == "" {
        return false, nil
    }
    var rec dbmodel.RevokedJTI
    tx := r.db.First(&rec, "jti = ?", jti)
    if tx.Error != nil {
        if tx.Error == gorm.ErrRecordNotFound {
            return false, nil
        }
        return false, tx.Error
    }
    return true, nil
}

