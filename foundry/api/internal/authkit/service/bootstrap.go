package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

var (
	ErrBootstrapDisabled = errors.New("bootstrap disabled")
	ErrBootstrapInvalid  = errors.New("invalid bootstrap token")
	ErrBootstrapUsed     = errors.New("bootstrap token already used")
)

// BootstrapService creates the initial admin user using a one-time token.
type BootstrapService interface {
	Bootstrap(ctx context.Context, token string, email string) (*domain.User, error)
}

type bootstrapService struct {
	bootstrapToken string
	users          store.UserStore
	boot           store.BootstrapStore
}

func NewBootstrapService(bootstrapToken string, users store.UserStore, boot store.BootstrapStore) BootstrapService {
	return &bootstrapService{bootstrapToken: bootstrapToken, users: users, boot: boot}
}

func (s *bootstrapService) Bootstrap(ctx context.Context, token string, email string) (*domain.User, error) {
	// Config preconditions
	if len(s.bootstrapToken) != 32 {
		return nil, ErrBootstrapDisabled
	}
	if len(token) != 32 {
		return nil, ErrBootstrapInvalid
	}
	// Constant-time compare
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.bootstrapToken)) != 1 {
		// Return not-found semantics to avoid info leak
		return nil, ErrBootstrapInvalid
	}

	// Hash token for replay record
	sum := sha256.Sum256([]byte(token))
	hashHex := hex.EncodeToString(sum[:])

	// Attempt to mark used first (fail-closed). If unique constraint violated,
	// treat as already used.
	if err := s.boot.MarkUsed(ctx, hashHex, email); err != nil {
		// Return not-found semantics to avoid leaking reuse state
		return nil, ErrBootstrapUsed
	}

	// Create admin user
	user, err := s.users.Create(ctx, email, []string{"admin"})
	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}
	return user, nil
}
