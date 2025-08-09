package test

import (
	"testing"

	buildsessions "github.com/input-output-hk/catalyst-forge/lib/foundry/client/buildsessions"
	"github.com/stretchr/testify/require"
)

func TestBuildSessions_Create(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	req := struct {
		OwnerType string                 `json:"owner_type"`
		OwnerID   string                 `json:"owner_id"`
		TTL       string                 `json:"ttl"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}{
		OwnerType: "repo",
		OwnerID:   "owner/repo",
		TTL:       "10m",
		Metadata:  map[string]interface{}{"workflow": "ci"},
	}

	out, err := c.BuildSessions().Create(ctx, &buildsessions.CreateRequest{
		OwnerType: req.OwnerType,
		OwnerID:   req.OwnerID,
		TTL:       req.TTL,
		Metadata:  req.Metadata,
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.ID)
}
