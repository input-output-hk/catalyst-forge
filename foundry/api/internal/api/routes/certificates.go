package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	akmw "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/middleware"
)

type CertificatesDeps struct {
	H *handlers.CertificateHandler
}

func RegisterCertificates(r *gin.Engine, d CertificatesDeps) {
	// For now, require authentication for signing endpoints. Policy enforcement will be
	// handled by AuthKit policies if needed at a higher layer.
	r.POST("/certificates/sign", akmw.RequireAuth(), d.H.SignCertificate)
	r.POST("/ca/buildkit/server-certificates", akmw.RequireAuth(), d.H.SignServerCertificate)
	r.GET("/certificates/root", d.H.GetRootCertificate)
}
