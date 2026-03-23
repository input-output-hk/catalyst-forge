package email

import (
	"context"
)

// Service defines an email sending interface
type Service interface {
	SendInvite(ctx context.Context, to string, inviteLink string) error
}
