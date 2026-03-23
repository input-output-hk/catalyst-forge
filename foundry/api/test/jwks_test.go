package test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
)

func TestJWKS(t *testing.T) {
	c := client.NewClient(getTestAPIURL())
	ctx, cancel := newTestContext()
	defer cancel()

	raw, err := c.JWKS().Get(ctx)
	require.NoError(t, err)

	var doc struct {
		Keys []any `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	assert.NotEmpty(t, doc.Keys)
}
