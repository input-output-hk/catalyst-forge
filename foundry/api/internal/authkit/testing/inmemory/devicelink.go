package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

// deviceLinkStore implements store.DeviceLinkStore for testing.
type deviceLinkStore struct {
	mu    sync.RWMutex
	links map[uuid.UUID]*domain.DeviceLink
}

// NewDeviceLinkStore creates a new in-memory device link store.
func NewDeviceLinkStore() store.DeviceLinkStore {
	return &deviceLinkStore{
		links: make(map[uuid.UUID]*domain.DeviceLink),
	}
}

// Create stores a new device link flow.
func (s *deviceLinkStore) Create(ctx context.Context, link *domain.DeviceLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clone to avoid mutations
	stored := *link
	s.links[link.ID] = &stored
	return nil
}

// GetByDeviceCode retrieves a device link by hashed device code.
func (s *deviceLinkStore) GetByDeviceCode(ctx context.Context, deviceCodeHash []byte) (*domain.DeviceLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	now := time.Now().UTC()
	for _, link := range s.links {
		if len(link.DeviceCode) == len(deviceCodeHash) {
			match := true
			for i := range link.DeviceCode {
				if link.DeviceCode[i] != deviceCodeHash[i] {
					match = false
					break
				}
			}
			if match && link.ExpiresAt.After(now) {
				// Return a copy
				result := *link
				return &result, nil
			}
		}
	}
	return nil, nil
}

// GetByUserCode retrieves a device link by user code.
func (s *deviceLinkStore) GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	now := time.Now().UTC()
	for _, link := range s.links {
		if link.UserCode == userCode && link.ExpiresAt.After(now) {
			// Return a copy
			result := *link
			return &result, nil
		}
	}
	return nil, nil
}

// Authorize marks a device link as authorized by a user.
func (s *deviceLinkStore) Authorize(ctx context.Context, id uuid.UUID, userID uuid.UUID, authorizedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	link, exists := s.links[id]
	if !exists {
		return nil
	}
	
	if link.AuthorizedAt == nil {
		link.UserID = &userID
		link.AuthorizedAt = &authorizedAt
	}
	return nil
}

// Delete removes an expired or completed device link.
func (s *deviceLinkStore) Delete(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.links, id)
	return nil
}

// DeleteExpired removes all expired device links.
func (s *deviceLinkStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var count int64
	for id, link := range s.links {
		if link.ExpiresAt.Before(before) {
			delete(s.links, id)
			count++
		}
	}
	return count, nil
}

// deviceStore implements store.DeviceStore for testing.
type deviceStore struct {
	mu      sync.RWMutex
	devices map[uuid.UUID]*domain.Device
}

// NewDeviceStore creates a new in-memory device store.
func NewDeviceStore() store.DeviceStore {
	return &deviceStore{
		devices: make(map[uuid.UUID]*domain.Device),
	}
}

// Create stores a new device.
func (s *deviceStore) Create(ctx context.Context, device *domain.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clone to avoid mutations
	stored := *device
	s.devices[device.ID] = &stored
	return nil
}

// GetByID retrieves a device by ID.
func (s *deviceStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	device, exists := s.devices[id]
	if !exists || device.RevokedAt != nil {
		return nil, nil
	}
	
	// Return a copy
	result := *device
	return &result, nil
}

// GetByUser retrieves all devices for a user.
func (s *deviceStore) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var devices []*domain.Device
	for _, device := range s.devices {
		if device.UserID == userID && device.RevokedAt == nil {
			// Add a copy
			d := *device
			devices = append(devices, &d)
		}
	}
	return devices, nil
}

// UpdateLastUsed updates the last used timestamp.
func (s *deviceStore) UpdateLastUsed(ctx context.Context, id uuid.UUID, lastUsedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if device, exists := s.devices[id]; exists {
		device.LastUsedAt = lastUsedAt
	}
	return nil
}

// Revoke marks a device as revoked.
func (s *deviceStore) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if device, exists := s.devices[id]; exists && device.RevokedAt == nil {
		device.RevokedAt = &revokedAt
	}
	return nil
}

// Delete removes a device.
func (s *deviceStore) Delete(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.devices, id)
	return nil
}