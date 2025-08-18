package rbac

import (
	"github.com/gin-gonic/gin"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/rbac"
)

// This file contains documentation-only handlers to generate Swagger paths for RBAC endpoints.

// @Summary List registered RBAC conditions
// @Tags rbac
// @Produce json
// @Success 200 {object} rbac.ConditionsResponse
// @Router /api/v1/rbac/conditions [get]
func DocRBACConditionsList(c *gin.Context) {}

// @Summary List canonical resource types
// @Tags rbac
// @Produce json
// @Success 200 {object} rbac.ResourceTypesResponse
// @Router /api/v1/rbac/resource-types [get]
func DocRBACResourceTypes(c *gin.Context) {}

// @Summary Resource type catalog (endpoints for each type)
// @Tags rbac
// @Produce json
// @Success 200 {object} rbac.ResourceTypeCatalogResponse
// @Router /api/v1/rbac/resource-types/catalog [get]
func DocRBACResourceTypesCatalog(c *gin.Context) {}

// @Summary List permission catalog
// @Tags rbac
// @Produce json
// @Success 200 {object} rbac.PermissionsCatalogResponse
// @Router /api/v1/rbac/permissions/catalog [get]
func DocRBACPermissionsCatalog(c *gin.Context) {}

// @Summary List roles
// @Tags rbac
// @Produce json
// @Success 200 {object} rbac.RolesListResponse
// @Router /api/v1/rbac/roles [get]
func DocRBACRolesList(c *gin.Context) {}

// @Summary Get role
// @Tags rbac
// @Produce json
// @Param slug path string true "Role slug"
// @Success 200 {object} rbac.Role
// @Failure 404
// @Router /api/v1/rbac/roles/{slug} [get]
func DocRBACRolesGet(c *gin.Context) {}

// @Summary Create role
// @Tags rbac
// @Accept json
// @Produce json
// @Param body body rbac.RoleDef true "Role definition"
// @Success 201
// @Router /api/v1/rbac/roles [post]
func DocRBACRolesCreate(c *gin.Context) {}

// @Summary Update role
// @Tags rbac
// @Accept json
// @Param slug path string true "Role slug"
// @Param body body rbac.RoleDef true "Role definition"
// @Success 204
// @Router /api/v1/rbac/roles/{slug} [put]
func DocRBACRolesUpdate(c *gin.Context) {}

// @Summary Bump role version
// @Tags rbac
// @Param slug path string true "Role slug"
// @Success 204
// @Router /api/v1/rbac/roles/{slug}/bump-version [post]
func DocRBACRolesBumpVersion(c *gin.Context) {}

// @Summary List bindings for a subject
// @Tags rbac
// @Produce json
// @Param subject_type query string true "Subject type (user|group|service)"
// @Param subject_id query string true "Subject ID"
// @Success 200 {object} rbac.BindingsListResponse
// @Router /api/v1/rbac/bindings [get]
func DocRBACBindingsList(c *gin.Context) {}

// @Summary List bindings in a scope
// @Tags rbac
// @Produce json
// @Param scope_type query string true "Scope type (global|org|project|resource)"
// @Param scope_id query string true "Scope ID"
// @Success 200 {object} rbac.BindingsListResponse
// @Router /api/v1/rbac/bindings/by-scope [get]
func DocRBACBindingsByScope(c *gin.Context) {}

// @Summary Create binding
// @Tags rbac
// @Accept json
// @Param body body rbac.BindingCreateRequest true "Binding"
// @Success 201
// @Router /api/v1/rbac/bindings [post]
func DocRBACBindingsCreate(c *gin.Context) {}

// @Summary Delete binding
// @Tags rbac
// @Param id path string true "Binding ID (UUID)"
// @Success 204
// @Router /api/v1/rbac/bindings/{id} [delete]
func DocRBACBindingsDelete(c *gin.Context) {}

// @Summary Bump principal version
// @Tags rbac
// @Param subject_type path string true "Subject type (user|group|service)"
// @Param subject_id path string true "Subject ID"
// @Success 204
// @Router /api/v1/rbac/subjects/{subject_type}/{subject_id}/bump-version [post]
func DocRBACSubjectBumpVersion(c *gin.Context) {}

// @Summary Explain decision
// @Tags rbac
// @Accept json
// @Produce json
// @Param body body rbac.ExplainRequest true "Explain request"
// @Success 200 {object} rbac.ExplainResponse
// @Failure 428
// @Router /api/v1/rbac/explain [post]
func DocRBACExplain(c *gin.Context) {}

// Silence unused imports for docs-only types
var _ = []interface{}{
	(*rbac.Role)(nil),
	(*rbac.Binding)(nil),
}
