package certs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/certkit/certkit"
	"github.com/stretchr/testify/require"
)

func TestPKI_CA(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	cfg := certkit.DefaultConfig()
	cfg.CAArn = "arn:ca:test"
	deps := certkit.Deps{PCA: &fakePCA{}, Clock: sysClock{}}
	ck, err := certkit.New(cfg, deps)
	require.NoError(t, err)
	ck.RegisterRoutes(r.Group("/pki"))

	req := httptest.NewRequest(http.MethodGet, "/pki/ca", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
