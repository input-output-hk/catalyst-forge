package certkit

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// Manager is the public facade for certkit.
type Manager interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

// New creates a new certkit manager.
func New(cfg Config, deps Deps) (Manager, error) {
	if deps.PCA == nil {
		return nil, errors.New("certkit: missing PCA dependency")
	}
	return &manager{cfg: cfg, deps: deps}, nil
}

type manager struct {
	cfg  Config
	deps Deps
}

func (m *manager) RegisterRoutes(rg *gin.RouterGroup) { registerRoutes(rg, m.cfg, m.deps) }
