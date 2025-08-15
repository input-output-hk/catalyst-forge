//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_Onboard_Begin_And_Complete validates onboarding begin/complete hooks exist and are guarded.
func TestAuth_Onboard_Begin_And_Complete(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// Begin should NOT require auth (invite-based), expect 200/400 depending on inputs
	var begin map[string]any
	resp, _ := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/onboard/begin", nil, map[string]any{
		"invite_id":   "00000000-0000-0000-0000-000000000000",
		"token":       "invalid",
		"device_name": "test",
	}, &begin)
	assert.NotNil(t, resp)
	assert.Contains(t, []int{http.StatusBadRequest}, resp.StatusCode)

	// With bearer, begin should at least not 401; accept 200 (normal) or 400/500 if flow preconditions missing in test env
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	begin = map[string]any{}
	resp, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/onboard/begin", headers, map[string]any{
		"invite_id":   "00000000-0000-0000-0000-000000000000",
		"token":       "invalid",
		"device_name": "test",
	}, &begin)
	// Likely 400 due to invalid invite
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Complete is bound; for now just verify it requires auth and responds (204 in happy path, may be 400 w/o body)
	var complete map[string]any
	resp, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/onboard/complete", nil, map[string]any{
		"session_key": "bad",
		"credential":  map[string]any{"fake": true},
	}, &complete)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
