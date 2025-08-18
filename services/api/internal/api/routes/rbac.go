package routes

import (
	"net/http"
	"strings"
	"time"

	"log/slog"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/rbac"
	authperms "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/permissions"
	authplanner "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/planner"
	appres "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/resolvers"
	resources "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/resources"
	apiauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
	rbacgorm "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac/gormstore"
	"gorm.io/gorm"
)

// RegisterRBACAdmin mounts RBAC administration endpoints under /api/v1/rbac.
// Endpoints are expected to be protected via policy registry rules (e.g., rbac:admin).
func RegisterRBACAdmin(r *gin.Engine, db *gorm.DB, logger *slog.Logger) {
	store := rbacgorm.New(db)
	cfg := rbac.DefaultConfig()
	cfg.Scopes = authplanner.Build()
	mgr, _ := rbac.New(cfg, rbac.Deps{
		Store:  store,
		Clock:  libauth.DefaultClock(),
		Logger: apiauth.NewLogger(logger),
		Cache:  rbac.NewMemoryCache(),
	})

	// Register canonical resolvers from internal/auth/resolvers
	for _, e := range appres.Registry() {
		mgr.RegisterResolver(e.Pattern, e.Resolver)
	}

	g := r.Group("/api/v1/rbac")

	// @Summary List registered RBAC conditions
	// @Tags rbac
	// @Produce json
	// @Success 200 {object} apimodels.ConditionsResponse
	// @Router /api/v1/rbac/conditions [get]
	g.GET("/conditions", func(c *gin.Context) {
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.ConditionsResponse{Conditions: rbac.ListConditions()})
	})

	// @Summary List canonical resource types
	// @Tags rbac
	// @Produce json
	// @Success 200 {object} apimodels.ResourceTypesResponse
	// @Router /api/v1/rbac/resource-types [get]
	g.GET("/resource-types", func(c *gin.Context) {
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.ResourceTypesResponse{Types: resources.Types()})
	})

	// @Summary Resource type catalog (endpoints for each type)
	// @Tags rbac
	// @Produce json
	// @Success 200 {object} apimodels.ResourceTypeCatalogResponse
	// @Router /api/v1/rbac/resource-types/catalog [get]
	g.GET("/resource-types/catalog", func(c *gin.Context) {
		items := resources.TypeCatalog()
		out := make([]apimodels.ResourceTypeCatalogItem, 0, len(items))
		for _, it := range items {
			out = append(out, apimodels.ResourceTypeCatalogItem{
				Type: it.Type,
				Endpoint: apimodels.ResourceEndpointInfo{
					Method:        it.Endpoint.Method,
					Path:          it.Endpoint.Path,
					IDField:       it.Endpoint.IDField,
					LabelField:    it.Endpoint.LabelField,
					QueryParams:   it.Endpoint.QueryParams,
					ParentFilters: it.Endpoint.ParentFilters,
				},
			})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.ResourceTypeCatalogResponse{Types: out})
	})

	// @Summary List permission catalog
	// @Tags rbac
	// @Produce json
	// @Success 200 {object} apimodels.PermissionsCatalogResponse
	// @Router /api/v1/rbac/permissions/catalog [get]
	g.GET("/permissions/catalog", func(c *gin.Context) {
		infos := authperms.Catalog()
		out := make([]apimodels.PermissionInfo, 0, len(infos))
		for _, i := range infos {
			out = append(out, apimodels.PermissionInfo{Key: i.Key, Name: i.Name, Description: i.Description, Domain: i.Domain})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.PermissionsCatalogResponse{Permissions: out})
	})

	// @Summary List roles
	// @Tags rbac
	// @Produce json
	// @Success 200 {object} apimodels.RolesListResponse
	// @Router /api/v1/rbac/roles [get]
	g.GET("/roles", func(c *gin.Context) {
		roles, err := store.ListRoles(c.Request.Context())
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]apimodels.Role, 0, len(roles))
		for _, r := range roles {
			out = append(out, apimodels.Role{ID: r.ID, Slug: r.Slug, Name: r.Name, Description: r.Description, Color: r.Color, Version: r.Version})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.RolesListResponse{Roles: out})
	})

	// @Summary Get role
	// @Tags rbac
	// @Produce json
	// @Param slug path string true "Role slug"
	// @Success 200 {object} apimodels.Role
	// @Failure 404
	// @Router /api/v1/rbac/roles/{slug} [get]
	g.GET("/roles/:slug", func(c *gin.Context) {
		slug := c.Param("slug")
		role, err := store.GetRole(c.Request.Context(), slug)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		if role == nil {
			c.Status(http.StatusNotFound)
			return
		}
		entries := make([]apimodels.RoleEntry, 0, len(role.Entries))
		for _, e := range role.Entries {
			conds := make([]apimodels.Condition, 0, len(e.Conditions))
			for _, cnd := range e.Conditions {
				conds = append(conds, apimodels.Condition{Name: cnd.Name, Params: cnd.Params})
			}
			entries = append(entries, apimodels.RoleEntry{Effect: string(e.Effect), Permission: string(e.Permission), ResourceType: e.ResourceType, Conditions: conds})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.Role{ID: role.ID, Slug: role.Slug, Name: role.Name, Description: role.Description, Color: role.Color, Version: role.Version, Entries: entries})
	})

	// @Summary Create role
	// @Tags rbac
	// @Accept json
	// @Produce json
	// @Param body body rbac.RoleDef true "Role definition"
	// @Success 201
	// @Router /api/v1/rbac/roles [post]
	g.POST("/roles", func(c *gin.Context) {
		var in rbac.RoleDef
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.Slug == "" {
			_ = basehttp.NewBadRequestError("slug required").Write(c.Writer)
			return
		}
		if err := store.CreateRole(c.Request.Context(), in); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusCreated)
	})

	// @Summary Update role
	// @Tags rbac
	// @Accept json
	// @Param slug path string true "Role slug"
	// @Param body body rbac.RoleDef true "Role definition"
	// @Success 204
	// @Router /api/v1/rbac/roles/{slug} [put]
	g.PUT("/roles/:slug", func(c *gin.Context) {
		slug := c.Param("slug")
		var in rbac.RoleDef
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.Slug == "" {
			in.Slug = slug
		}
		if !strings.EqualFold(in.Slug, slug) {
			_ = basehttp.NewBadRequestError("slug mismatch").Write(c.Writer)
			return
		}
		if err := store.UpdateRole(c.Request.Context(), in); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Bump role version
	// @Tags rbac
	// @Param slug path string true "Role slug"
	// @Success 204
	// @Router /api/v1/rbac/roles/{slug}/bump-version [post]
	g.POST("/roles/:slug/bump-version", func(c *gin.Context) {
		slug := c.Param("slug")
		if err := store.BumpRoleVersion(c.Request.Context(), slug); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary List bindings for a subject
	// @Tags rbac
	// @Produce json
	// @Param subject_type query string true "Subject type (user|group|service)"
	// @Param subject_id query string true "Subject ID"
	// @Success 200 {object} apimodels.BindingsListResponse
	// @Router /api/v1/rbac/bindings [get]
	g.GET("/bindings", func(c *gin.Context) {
		st := c.Query("subject_type")
		sid := c.Query("subject_id")
		if st == "" || sid == "" {
			_ = basehttp.NewBadRequestError("subject_type and subject_id required").Write(c.Writer)
			return
		}
		var subjType rbac.SubjectType
		switch strings.ToLower(st) {
		case "user":
			subjType = rbac.SubjectUser
		case "group":
			subjType = rbac.SubjectGroup
		case "service":
			subjType = rbac.SubjectService
		default:
			_ = basehttp.NewBadRequestError("invalid subject_type").Write(c.Writer)
			return
		}
		binds, err := store.ListBindings(c.Request.Context(), rbac.Subject{Type: subjType, ID: sid})
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]apimodels.Binding, 0, len(binds))
		for _, b := range binds {
			out = append(out, apimodels.Binding{ID: b.ID, SubjectType: string(b.Subject.Type), SubjectID: b.Subject.ID, RoleSlug: b.RoleSlug, ScopeType: string(b.ScopeType), ScopeID: b.ScopeID, OrgID: b.OrgID, CreatedAt: b.CreatedAt})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.BindingsListResponse{Bindings: out})
	})

	// @Summary List bindings in a scope
	// @Tags rbac
	// @Produce json
	// @Param scope_type query string true "Scope type (global|org|project|resource)"
	// @Param scope_id query string true "Scope ID"
	// @Success 200 {object} apimodels.BindingsListResponse
	// @Router /api/v1/rbac/bindings/by-scope [get]
	g.GET("/bindings/by-scope", func(c *gin.Context) {
		st := c.Query("scope_type")
		sid := c.Query("scope_id")
		if st == "" || sid == "" {
			_ = basehttp.NewBadRequestError("scope_type and scope_id required").Write(c.Writer)
			return
		}
		var scope rbac.ScopeType
		switch strings.ToLower(st) {
		case "global":
			scope = rbac.ScopeGlobal
		case "org":
			scope = rbac.ScopeOrg
		case "project":
			scope = rbac.ScopeProject
		case "resource":
			scope = rbac.ScopeRes
		default:
			_ = basehttp.NewBadRequestError("invalid scope_type").Write(c.Writer)
			return
		}
		binds, err := store.ListBindingsByScope(c.Request.Context(), scope, sid)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]apimodels.Binding, 0, len(binds))
		for _, b := range binds {
			out = append(out, apimodels.Binding{ID: b.ID, SubjectType: string(b.Subject.Type), SubjectID: b.Subject.ID, RoleSlug: b.RoleSlug, ScopeType: string(b.ScopeType), ScopeID: b.ScopeID, OrgID: b.OrgID, CreatedAt: b.CreatedAt})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.BindingsListResponse{Bindings: out})
	})

	// @Summary Create binding
	// @Tags rbac
	// @Accept json
	// @Param body body apimodels.BindingCreateRequest true "Binding"
	// @Success 201
	// @Router /api/v1/rbac/bindings [post]
	g.POST("/bindings", func(c *gin.Context) {
		var in apimodels.BindingCreateRequest
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.Subject.Type == "" || in.Subject.ID == "" || in.RoleSlug == "" || in.ScopeType == "" {
			_ = basehttp.NewBadRequestError("missing required fields").Write(c.Writer)
			return
		}
		var subjType rbac.SubjectType
		switch strings.ToLower(in.Subject.Type) {
		case "user":
			subjType = rbac.SubjectUser
		case "group":
			subjType = rbac.SubjectGroup
		case "service":
			subjType = rbac.SubjectService
		default:
			_ = basehttp.NewBadRequestError("invalid subject.type").Write(c.Writer)
			return
		}
		var scope rbac.ScopeType
		switch strings.ToLower(in.ScopeType) {
		case "global":
			scope = rbac.ScopeGlobal
		case "org":
			scope = rbac.ScopeOrg
		case "project":
			scope = rbac.ScopeProject
		case "resource":
			scope = rbac.ScopeRes
		default:
			_ = basehttp.NewBadRequestError("invalid scope_type").Write(c.Writer)
			return
		}
		var orgID *uuid.UUID
		if in.OrgID != "" {
			if oid, err := uuid.Parse(in.OrgID); err == nil {
				orgID = &oid
			}
		}
		bid := uuid.New()
		if in.ID != "" {
			if v, err := uuid.Parse(in.ID); err == nil {
				bid = v
			}
		}
		b := rbac.Binding{ID: bid, Subject: rbac.Subject{Type: subjType, ID: in.Subject.ID, OrgID: orgID}, RoleSlug: in.RoleSlug, ScopeType: scope, ScopeID: in.ScopeID, OrgID: orgID, CreatedAt: time.Now().UTC()}
		if err := store.AddBinding(c.Request.Context(), b); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusCreated)
	})

	// @Summary Delete binding
	// @Tags rbac
	// @Param id path string true "Binding ID (UUID)"
	// @Success 204
	// @Router /api/v1/rbac/bindings/{id} [delete]
	g.DELETE("/bindings/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid id").Write(c.Writer)
			return
		}
		if err := store.RemoveBinding(c.Request.Context(), id); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Bump principal version
	// @Tags rbac
	// @Param subject_type path string true "Subject type (user|group|service)"
	// @Param subject_id path string true "Subject ID"
	// @Success 204
	// @Router /api/v1/rbac/subjects/{subject_type}/{subject_id}/bump-version [post]
	g.POST("/subjects/:subject_type/:subject_id/bump-version", func(c *gin.Context) {
		st := c.Param("subject_type")
		sid := c.Param("subject_id")
		var subjType rbac.SubjectType
		switch strings.ToLower(st) {
		case "user":
			subjType = rbac.SubjectUser
		case "group":
			subjType = rbac.SubjectGroup
		case "service":
			subjType = rbac.SubjectService
		default:
			_ = basehttp.NewBadRequestError("invalid subject_type").Write(c.Writer)
			return
		}
		if err := store.BumpPrincipalVersion(c.Request.Context(), rbac.Subject{Type: subjType, ID: sid}); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Explain decision
	// @Tags rbac
	// @Accept json
	// @Produce json
	// @Param body body apimodels.ExplainRequest true "Explain request"
	// @Success 200 {object} apimodels.ExplainResponse
	// @Failure 428
	// @Router /api/v1/rbac/explain [post]
	g.POST("/explain", func(c *gin.Context) {
		var in apimodels.ExplainRequest
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.Subject.Type == "" || in.Subject.ID == "" || in.Permission == "" || in.Resource.Type == "" {
			_ = basehttp.NewBadRequestError("missing required fields").Write(c.Writer)
			return
		}
		var subjType rbac.SubjectType
		switch strings.ToLower(in.Subject.Type) {
		case "user":
			subjType = rbac.SubjectUser
		case "group":
			subjType = rbac.SubjectGroup
		case "service":
			subjType = rbac.SubjectService
		default:
			_ = basehttp.NewBadRequestError("invalid subject.type").Write(c.Writer)
			return
		}
		var subjOrg *uuid.UUID
		if in.Subject.OrgID != "" {
			if v, err := uuid.Parse(in.Subject.OrgID); err == nil {
				subjOrg = &v
			}
		}
		var resOrg *uuid.UUID
		if in.Resource.OrgID != "" {
			if v, err := uuid.Parse(in.Resource.OrgID); err == nil {
				resOrg = &v
			}
		}
		subj := rbac.Subject{Type: subjType, ID: in.Subject.ID, OrgID: subjOrg, Attrs: in.Subject.Attrs}
		res := rbac.ResourceRef{Type: in.Resource.Type, ID: in.Resource.ID, OrgID: resOrg, Attrs: in.Resource.Attrs}
		dec, trace, err := mgr.Explain(c.Request.Context(), subj, rbac.PermissionKey(in.Permission), res)
		if err != nil {
			// Map known step-up error to 428
			if err == rbac.ErrConditionStepUpRequired {
				basehttp.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Step-up authentication required")
				return
			}
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.ExplainResponse{Decision: string(dec), Trace: trace})
	})
}
