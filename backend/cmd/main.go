package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"dockvol-backend/internal/config"
	"dockvol-backend/internal/features/audit_logs"
	"dockvol-backend/internal/features/disk"
	"dockvol-backend/internal/features/docker"
	"dockvol-backend/internal/features/encryption/secrets"
	"dockvol-backend/internal/features/notifiers"
	"dockvol-backend/internal/features/storages"
	system_version "dockvol-backend/internal/features/system/version"
	users_controllers "dockvol-backend/internal/features/users/controllers"
	users_middleware "dockvol-backend/internal/features/users/middleware"
	users_services "dockvol-backend/internal/features/users/services"
	workspaces_controllers "dockvol-backend/internal/features/workspaces/controllers"
	"dockvol-backend/internal/middleware"
	"dockvol-backend/internal/schema"
	"dockvol-backend/internal/storage"
	env_utils "dockvol-backend/internal/util/env"
	files_utils "dockvol-backend/internal/util/files"
	"dockvol-backend/internal/util/logger"
)

const serverAddr = ":4005"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheckCommand() // exits the process
		return
	}

	log := logger.GetLogger()

	runAutoMigrate(log)

	err := files_utils.EnsureDirectories([]string{
		config.GetEnv().TempFolder,
		config.GetEnv().DataFolder,
	})
	if err != nil {
		log.Error("Failed to ensure directories", "error", err)
		os.Exit(1)
	}

	err = secrets.GetSecretKeyService().MigrateKeyFromDbToFileIfExist()
	if err != nil {
		log.Error("Failed to migrate secret key from database to file", "error", err)
		os.Exit(1)
	}

	err = users_services.GetUserService().CreateInitialAdmin()
	if err != nil {
		log.Error("Failed to create initial admin", "error", err)
		os.Exit(1)
	}

	handlePasswordReset(log)

	gin.SetMode(gin.ReleaseMode)
	ginApp := gin.New()
	ginApp.Use(gin.Logger())
	ginApp.Use(ginRecoveryWithLogger(log))
	ginApp.Use(middleware.NoStoreCacheControl())

	ginApp.Use(gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions(
			[]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"},
		),
	))

	enableCors(ginApp)
	setUpRoutes(ginApp)
	setUpDependencies()

	runBackgroundTasks(log)

	mountFrontend(ginApp)

	startServerWithGracefulShutdown(log, ginApp)
}

func handlePasswordReset(log *slog.Logger) {
	audit_logs.SetupDependencies()

	newPassword := flag.String("new-password", "", "Set a new password for the user")
	email := flag.String("email", "", "Email of the user to reset password")

	flag.Parse()

	if *newPassword == "" {
		return
	}

	log.Info("Found reset password command - reseting password...")

	if *email == "" {
		log.Info("No email provided, please provide an email via --email=\"some@email.com\" flag")
		os.Exit(1)
	}

	resetPassword(*email, *newPassword, log)
}

func resetPassword(email, newPassword string, log *slog.Logger) {
	log.Info("Resetting password...")

	userService := users_services.GetUserService()
	err := userService.ChangeUserPasswordByEmail(email, newPassword)
	if err != nil {
		log.Error("Failed to reset password", "error", err)
		os.Exit(1)
	}

	log.Info("Password reset successfully")
	os.Exit(0)
}

func startServerWithGracefulShutdown(log *slog.Logger, app *gin.Engine) {
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: app,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen:", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown signal received")

	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown:", "error", err)
	}

	log.Info("Server gracefully stopped")
}

func setUpRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// Public routes (only user auth routes and healthcheck should be public)
	userController := users_controllers.GetUserController()
	userController.RegisterRoutes(v1)
	system_version.GetVersionController().RegisterRoutes(v1)

	userService := users_services.GetUserService()
	authMiddleware := users_middleware.AuthMiddleware(userService)

	protected := v1.Group("")
	protected.Use(authMiddleware)

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
}

func setUpDependencies() {
	audit_logs.SetupDependencies()
	notifiers.SetupDependencies()
	storages.SetupDependencies()
}

func runBackgroundTasks(log *slog.Logger) {
	log.Info("Preparing to run background tasks...")

	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		log.Info("Shutdown signal received, cancelling all background tasks")
		cancel()
	}()

	err := files_utils.CleanFolder(config.GetEnv().TempFolder)
	if err != nil {
		log.Error("Failed to clean temp folder", "error", err)
	}

	log.Info("Starting background tasks...")

	go runWithPanicLogging(log, "audit log cleanup background service", func() {
		audit_logs.GetAuditLogBackgroundService().Run(ctx)
	})

	go runWithPanicLogging(log, "volume backup scheduler background service", func() {
		docker.GetBackupScheduler().Run(ctx)
	})
}

func runWithPanicLogging(log *slog.Logger, serviceName string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in "+serviceName, "error", r, "stacktrace", string(debug.Stack()))
		}
	}()
	fn()
}

func runAutoMigrate(log *slog.Logger) {
	log.Info("Running database migrations...")

	if err := schema.AutoMigrate(storage.GetDb()); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.Info("Database migrations completed successfully")
}

func enableCors(ginApp *gin.Engine) {
	if config.GetEnv().EnvMode == env_utils.EnvModeDevelopment {
		ginApp.Use(cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders: []string{
				"Origin",
				"Content-Length",
				"Content-Type",
				"Authorization",
				"Accept",
				"Accept-Language",
				"Accept-Encoding",
				"Access-Control-Request-Method",
				"Access-Control-Request-Headers",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Allow-Origin",
			},
			AllowCredentials: true,
		}))
	}
}

func ginRecoveryWithLogger(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Panic recovered in HTTP handler",
					"error", r,
					"stacktrace", string(debug.Stack()),
					"method", ctx.Request.Method,
					"path", ctx.Request.URL.Path,
				)

				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		ctx.Next()
	}
}

func mountFrontend(ginApp *gin.Engine) {
	staticDir := "./ui/build"
	ginApp.NoRoute(func(c *gin.Context) {
		path := filepath.Join(staticDir, c.Request.URL.Path)

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			c.File(path)
			return
		}

		c.File(filepath.Join(staticDir, "index.html"))
	})
}
