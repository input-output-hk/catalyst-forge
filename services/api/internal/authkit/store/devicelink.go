package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
)

// DeviceLinkStore handles persistence of device link flows.
type DeviceLinkStore interface {
	// Create stores a new device link flow.
	Create(ctx context.Context, link *domain.DeviceLink) error

	// GetByDeviceCode retrieves a device link by hashed device code.
	GetByDeviceCode(ctx context.Context, deviceCodeHash []byte) (*domain.DeviceLink, error)

	// GetByUserCode retrieves a device link by user code.
	GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceLink, error)

	// Authorize marks a device link as authorized by a user.
	Authorize(ctx context.Context, id uuid.UUID, userID uuid.UUID, authorizedAt time.Time) error

	// Delete removes an expired or completed device link.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteExpired removes all expired device links.
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// DeviceStore handles persistence of linked devices.
type DeviceStore interface {
	// Create stores a new device.
	Create(ctx context.Context, device *domain.Device) error

	// GetByID retrieves a device by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error)

	// GetByUser retrieves all devices for a user.
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Device, error)

	// UpdateLastUsed updates the last used timestamp.
	UpdateLastUsed(ctx context.Context, id uuid.UUID, lastUsedAt time.Time) error

	// Revoke marks a device as revoked.
	Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error

	// Delete removes a device and cascades to refresh tokens.
	Delete(ctx context.Context, id uuid.UUID) error
}