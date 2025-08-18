import type { components } from "forge-client";
import { forge } from "@/lib/client";

// Types from generated OpenAPI
type RolesListResponse = components["schemas"]["rbac.RolesListResponse"];
type Role = components["schemas"]["rbac.Role"];
type RoleDef = components["schemas"]["rbac.RoleDef"];
type BindingsListResponse = components["schemas"]["rbac.BindingsListResponse"];
type BindingCreateRequest = components["schemas"]["rbac.BindingCreateRequest"];
type ConditionsResponse = components["schemas"]["rbac.ConditionsResponse"];
type PermissionsResponse = components["schemas"]["rbac.PermissionsResponse"];
type ExplainRequest = components["schemas"]["rbac.ExplainRequest"];
type ExplainResponse = components["schemas"]["rbac.ExplainResponse"];

type Result<T> = Promise<{ data?: T; response: Response }>;

export const rbacApi = {
  listConditions: async (): Result<ConditionsResponse> =>
    forge.raw.GET("/api/v1/rbac/conditions"),

  listPermissions: async (): Result<PermissionsResponse> =>
    forge.raw.GET("/api/v1/rbac/permissions"),

  listRoles: async (): Result<RolesListResponse> =>
    forge.raw.GET("/api/v1/rbac/roles"),

  getRole: async (slug: string): Result<Role> =>
    forge.raw.GET("/api/v1/rbac/roles/{slug}", { params: { path: { slug } } }),

  createRole: async (role: RoleDef): Result<Record<string, never>> =>
    forge.raw.POST("/api/v1/rbac/roles", { body: role }),

  updateRole: async (slug: string, role: RoleDef): Result<never> =>
    forge.raw.PUT("/api/v1/rbac/roles/{slug}", { params: { path: { slug } }, body: role }),

  bumpRoleVersion: async (slug: string): Result<never> =>
    forge.raw.POST("/api/v1/rbac/roles/{slug}/bump-version", { params: { path: { slug } } }),

  listBindingsBySubject: async (
    subject_type: string,
    subject_id: string
  ): Result<BindingsListResponse> =>
    forge.raw.GET("/api/v1/rbac/bindings", { params: { query: { subject_type, subject_id } } }),

  listBindingsByScope: async (
    scope_type: string,
    scope_id: string
  ): Result<BindingsListResponse> =>
    forge.raw.GET("/api/v1/rbac/bindings/by-scope", { params: { query: { scope_type, scope_id } } }),

  createBinding: async (req: BindingCreateRequest): Result<never> =>
    forge.raw.POST("/api/v1/rbac/bindings", { body: req }),

  deleteBinding: async (id: string): Result<never> =>
    forge.raw.DELETE("/api/v1/rbac/bindings/{id}", { params: { path: { id } } }),

  bumpPrincipalVersion: async (subject_type: string, subject_id: string): Result<never> =>
    forge.raw.POST("/api/v1/rbac/subjects/{subject_type}/{subject_id}/bump-version", {
      params: { path: { subject_type, subject_id } },
    }),

  explain: async (body: ExplainRequest): Result<ExplainResponse> =>
    forge.raw.POST("/api/v1/rbac/explain", { body }),
};

export type { Role, RoleDef, RolesListResponse, BindingsListResponse, ConditionsResponse, ExplainRequest, ExplainResponse };


