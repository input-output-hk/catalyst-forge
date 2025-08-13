package handlers

import (
	"net/http"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
)

// JWKSHandler writes the JSON Web Key Set response.
func JWKSHandler(w http.ResponseWriter, jwks any) {
	_ = basehttp.WriteJSON(w, http.StatusOK, jwks)
}
