package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
)

type DeviceDeps struct {
	Auth    *middleware.AuthMiddleware
	CORS    gin.HandlerFunc
	Rate    gin.HandlerFunc
	AuthCfg *config.Config
	Reg     *handlers.DeviceRegistrationHandler
	Refresh *handlers.DeviceRefreshHandler
	Logout  *handlers.DeviceLogoutHandler
	Login   *handlers.DeviceLoginHandler
	Mgmt    *handlers.DeviceManagementHandler
}

func RegisterDevices(r *gin.Engine, d DeviceDeps) {
	// Device registration endpoints
	r.POST("/auth/devices/init", d.CORS, d.Rate, d.Reg.InitDeviceRegistration)
	r.POST("/auth/devices/register", d.CORS, d.Rate, d.Reg.RegisterDevice)

	// Returning device login endpoints
	r.POST("/auth/devices/login/init", d.CORS, d.Rate, d.Login.InitLogin)
	r.POST("/auth/devices/login", d.CORS, d.Rate, d.Login.Login)

	// Device-bound refresh/logout
	r.POST("/auth/refresh", d.CORS, d.Rate, d.Refresh.RefreshToken)
	r.POST("/auth/logout", d.CORS, d.Logout.Logout)

	// Device management (require JWT)
	r.GET("/auth/devices", d.CORS, d.Auth.ValidatePermissions(nil), d.Mgmt.ListDevices)
	r.DELETE("/auth/devices/:id", d.CORS, d.Auth.ValidatePermissions(nil), d.Mgmt.DeleteDevice)
}
