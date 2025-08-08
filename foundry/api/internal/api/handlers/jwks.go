package handlers

import (
	"crypto"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	jose "gopkg.in/square/go-jose.v2"
)

// JWKSHandler serves the public JWKS for token verification
type JWKSHandler struct {
	jwtManager jwt.JWTManager
}

func NewJWKSHandler(jwtManager jwt.JWTManager) *JWKSHandler {
	return &JWKSHandler{jwtManager: jwtManager}
}

// GetJWKS returns the JSON Web Key Set with cache headers
// @Summary Get JWKS
// @Description Returns the public JSON Web Key Set used to verify access tokens
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "JWKS"
// @Router /.well-known/jwks.json [get]
func (h *JWKSHandler) GetJWKS(c *gin.Context) {
	pub := h.jwtManager.PublicKey()
	if pub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jwks unavailable"})
		return
	}

	jwk := jose.JSONWebKey{Key: pub, Algorithm: "ES256", Use: "sig"}
	// Compute RFC7638 thumbprint for stable kid
	thumb, err := jwk.Thumbprint(crypto.SHA256)
	if err == nil {
		jwk.KeyID = hex.EncodeToString(thumb)
	}

	ks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	// Compute weak ETag from hash of JSON
	// Compute ETag from fields (stable enough for single-key JWKS)
	// Build a minimal string representative for ETag
	etagSrc := jwk.KeyID + jwk.Algorithm + jwk.Use
	sum := sha256.Sum256([]byte(etagSrc))
	etag := "\"" + hex.EncodeToString(sum[:]) + "\""

	if match := c.Request.Header.Get("If-None-Match"); match != "" && match == etag {
		c.Header("ETag", etag)
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("Cache-Control", "public, max-age=300") // 5 minutes
	c.Header("ETag", etag)
	c.JSON(http.StatusOK, ks)
}
