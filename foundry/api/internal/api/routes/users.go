package routes

import (
	"github.com/gin-gonic/gin"
	apiuser "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	auth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

type UserDeps struct {
	Auth     *middleware.AuthMiddleware
	User     *apiuser.UserHandler
	Role     *apiuser.RoleHandler
	UserRole *apiuser.UserRoleHandler
	UserKey  *apiuser.UserKeyHandler
}

func RegisterUsers(r *gin.Engine, d UserDeps) {
	// User endpoints
	r.POST("/auth/users", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), d.User.CreateUser)
	r.GET("/auth/users", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserRead}), d.User.ListUsers)
	r.GET("/auth/users/email/:email", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserRead}), d.User.GetUserByEmail)
	r.GET("/auth/users/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserRead}), d.User.GetUser)
	r.PUT("/auth/users/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), d.User.UpdateUser)
	r.DELETE("/auth/users/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), d.User.DeleteUser)
	r.POST("/auth/users/:id/activate", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), d.User.ActivateUser)
	r.POST("/auth/users/:id/deactivate", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), d.User.DeactivateUser)

	// User key endpoints
	r.POST("/auth/keys", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), d.UserKey.CreateUserKey)
	r.POST("/auth/keys/bootstrap", d.UserKey.BootstrapKET)
	r.POST("/auth/keys/register", d.UserKey.RegisterWithKET)
	r.GET("/auth/keys", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), d.UserKey.ListUserKeys)
	r.GET("/auth/keys/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), d.UserKey.GetUserKey)
	r.GET("/auth/keys/kid/:kid", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), d.UserKey.GetUserKeyByKid)
	r.PUT("/auth/keys/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), d.UserKey.UpdateUserKey)
	r.DELETE("/auth/keys/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), d.UserKey.DeleteUserKey)
	r.POST("/auth/keys/:id/revoke", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), d.UserKey.RevokeUserKey)
	r.GET("/auth/keys/user/:user_id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), d.UserKey.GetUserKeysByUserID)
	r.GET("/auth/keys/user/:user_id/active", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), d.UserKey.GetActiveUserKeysByUserID)
	r.GET("/auth/keys/user/:user_id/inactive", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), d.UserKey.GetInactiveUserKeysByUserID)

	// Roles
	r.POST("/auth/roles", d.Auth.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), d.Role.CreateRole)
	r.GET("/auth/roles", d.Auth.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), d.Role.ListRoles)
	r.GET("/auth/roles/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), d.Role.GetRole)
	r.GET("/auth/roles/name/:name", d.Auth.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), d.Role.GetRoleByName)
	r.PUT("/auth/roles/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), d.Role.UpdateRole)
	r.DELETE("/auth/roles/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), d.Role.DeleteRole)

	// User-roles
	r.POST("/auth/user-roles", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite, auth.PermRoleWrite}), d.UserRole.AssignUserToRole)
	r.DELETE("/auth/user-roles", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite, auth.PermRoleWrite}), d.UserRole.RemoveUserFromRole)
	r.GET("/auth/user-roles", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserRead, auth.PermRoleRead}), d.UserRole.GetUserRoles)
	r.GET("/auth/role-users", d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserRead, auth.PermRoleRead}), d.UserRole.GetRoleUsers)
}
