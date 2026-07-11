package testutil

import (
	"sync"

	"github.com/gin-gonic/gin"

	"dockvol-backend/internal/features/audit_logs"
	"dockvol-backend/internal/features/disk"
	"dockvol-backend/internal/features/docker"
	"dockvol-backend/internal/features/notifiers"
	"dockvol-backend/internal/features/storages"
	system_version "dockvol-backend/internal/features/system/version"
	users_controllers "dockvol-backend/internal/features/users/controllers"
	users_middleware "dockvol-backend/internal/features/users/middleware"
	users_services "dockvol-backend/internal/features/users/services"
	workspaces_controllers "dockvol-backend/internal/features/workspaces/controllers"
)

var setupDependencies = sync.OnceFunc(func() {
	audit_logs.SetupDependencies()
	notifiers.SetupDependencies()
	storages.SetupDependencies()
})

// NewRouter builds a gin engine wired like the production app: public auth and
// version routes, plus the protected feature routes behind the auth middleware.
func NewRouter() *gin.Engine {
	setupDependencies()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")

	userController := users_controllers.GetUserController()
	userController.RegisterRoutes(v1)
	system_version.GetVersionController().RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	userController.RegisterProtectedRoutes(protected)
	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected)
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected)
	disk.GetDiskController().RegisterRoutes(protected)
	docker.GetDockerController().RegisterRoutes(protected)
	notifiers.GetNotifierController().RegisterRoutes(protected)
	storages.GetStorageController().RegisterRoutes(protected)
	audit_logs.GetAuditLogController().RegisterRoutes(protected)
	users_controllers.GetManagementController().RegisterRoutes(protected)
	users_controllers.GetSettingsController().RegisterRoutes(protected)

	return router
}
