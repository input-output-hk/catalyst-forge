package auth

import (
	"net/http"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apimodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/models/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

type GithubPolicyDeps struct{ Store store.GithubPolicyStore }

// @Summary Create GitHub policy
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.GithubPolicyCreateRequest true "Create policy"
// @Success 201 {object} auth.GithubPolicyResponse
// @Router /api/v1/auth/oidc/github/policies [post]
func RegisterGithubPolicies(r *gin.Engine, d GithubPolicyDeps) {
	r.POST("/api/v1/auth/oidc/github/policies", func(c *gin.Context) {
		var req apimodel.GithubPolicyCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = basehttp.NewBadRequestError("invalid request").Write(c.Writer)
			return
		}
		p := &store.GithubPolicy{ID: uuid.New(), Repository: req.Repository, Refs: req.Refs, Environments: req.Environments, Workflows: req.Workflows, Roles: req.Roles, Enabled: req.Enabled}
		if err := d.Store.Create(c.Request.Context(), p); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.JSON(http.StatusCreated, apimodel.GithubPolicyResponse{ID: p.ID, Repository: p.Repository, Refs: p.Refs, Environments: p.Environments, Workflows: p.Workflows, Roles: p.Roles, Enabled: p.Enabled})
	})

	// @Summary List GitHub policies
	// @Tags auth
	// @Produce json
	// @Success 200 {array} auth.GithubPolicyResponse
	// @Router /api/v1/auth/oidc/github/policies [get]
	r.GET("/api/v1/auth/oidc/github/policies", func(c *gin.Context) {
		list, err := d.Store.List(c.Request.Context())
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]apimodel.GithubPolicyResponse, 0, len(list))
		for _, p := range list {
			out = append(out, apimodel.GithubPolicyResponse{ID: p.ID, Repository: p.Repository, Refs: p.Refs, Environments: p.Environments, Workflows: p.Workflows, Roles: p.Roles, Enabled: p.Enabled})
		}
		c.JSON(http.StatusOK, out)
	})

	// @Summary Get GitHub policy
	// @Tags auth
	// @Produce json
	// @Param id path string true "Policy ID"
	// @Success 200 {object} auth.GithubPolicyResponse
	// @Failure 404 {object} map[string]interface{}
	// @Router /api/v1/auth/oidc/github/policies/{id} [get]
	r.GET("/api/v1/auth/oidc/github/policies/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid id").Write(c.Writer)
			return
		}
		p, err := d.Store.GetByID(c.Request.Context(), id)
		if err != nil {
			_ = basehttp.NewNotFoundError("").Write(c.Writer)
			return
		}
		c.JSON(http.StatusOK, apimodel.GithubPolicyResponse{ID: p.ID, Repository: p.Repository, Refs: p.Refs, Environments: p.Environments, Workflows: p.Workflows, Roles: p.Roles, Enabled: p.Enabled})
	})

	// @Summary Update GitHub policy
	// @Tags auth
	// @Accept json
	// @Param id path string true "Policy ID"
	// @Param request body auth.GithubPolicyUpdateRequest true "Update policy"
	// @Success 204
	// @Router /api/v1/auth/oidc/github/policies/{id} [put]
	r.PUT("/api/v1/auth/oidc/github/policies/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid id").Write(c.Writer)
			return
		}
		var req apimodel.GithubPolicyUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = basehttp.NewBadRequestError("invalid request").Write(c.Writer)
			return
		}
		p := &store.GithubPolicy{ID: id, Refs: req.Refs, Environments: req.Environments, Workflows: req.Workflows, Roles: req.Roles, Enabled: req.Enabled}
		if err := d.Store.Update(c.Request.Context(), p); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Delete GitHub policy
	// @Tags auth
	// @Param id path string true "Policy ID"
	// @Success 204
	// @Router /api/v1/auth/oidc/github/policies/{id} [delete]
	r.DELETE("/api/v1/auth/oidc/github/policies/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid id").Write(c.Writer)
			return
		}
		if err := d.Store.Delete(c.Request.Context(), id); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
