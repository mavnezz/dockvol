package docker

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	dockerService *Service
	backuper      *Backuper
	configService *ConfigService
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	dockerRoutes := router.Group("/docker")

	dockerRoutes.GET("/containers", c.GetContainers)
	dockerRoutes.POST("/backup", c.CreateBackup)
	dockerRoutes.GET("/backups", c.GetBackups)
	dockerRoutes.GET("/backups/:id/download", c.DownloadBackup)
	dockerRoutes.POST("/backups/:id/restore", c.RestoreBackup)
	dockerRoutes.DELETE("/backups/:id", c.DeleteBackup)

	dockerRoutes.GET("/configs", c.GetConfigs)
	dockerRoutes.POST("/configs", c.SaveConfig)
	dockerRoutes.DELETE("/configs/:id", c.DeleteConfig)
}

// GetContainers
// @Summary List Docker containers with their mounts
// @Description List running containers on the host and the mounts eligible for backup
// @Tags docker
// @Produce json
// @Param Authorization header string true "JWT token"
// @Success 200 {object} map[string][]Container
// @Failure 401
// @Failure 500
// @Router /docker/containers [get]
func (c *Controller) GetContainers(ctx *gin.Context) {
	containers, err := c.dockerService.GetContainers(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list docker containers"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"containers": containers})
}

// CreateBackup
// @Summary Back up a container's selected mounts
// @Description Stream the selected container mounts as a gzipped tar to the target storage
// @Tags docker
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param request body CreateBackupRequestDTO true "Container, storage and mount paths to back up"
// @Success 200 {object} VolumeBackup
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /docker/backup [post]
func (c *Controller) CreateBackup(ctx *gin.Context) {
	var request CreateBackupRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	volumeBackup, err := c.backuper.CreateBackup(ctx.Request.Context(), request)
	if err != nil {
		if errors.Is(err, ErrContainerNotFound) ||
			errors.Is(err, ErrMountPathNotFound) ||
			errors.Is(err, ErrStorageNotFound) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create volume backup"})
		return
	}

	ctx.JSON(http.StatusOK, volumeBackup)
}

// GetBackups
// @Summary List volume backups
// @Description List volume backups newest-first, optionally filtered by container
// @Tags docker
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param containerId query string false "Filter by container ID"
// @Success 200 {object} map[string][]VolumeBackup
// @Failure 401
// @Failure 500
// @Router /docker/backups [get]
func (c *Controller) GetBackups(ctx *gin.Context) {
	volumeBackups, err := c.backuper.ListBackups(ctx.Query("containerId"))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list volume backups"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"backups": volumeBackups})
}

// DownloadBackup
// @Summary Download a volume backup archive
// @Tags docker
// @Produce application/gzip
// @Param Authorization header string true "JWT token"
// @Param id path string true "Backup ID"
// @Success 200 {file} binary
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /docker/backups/{id}/download [get]
func (c *Controller) DownloadBackup(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup id"})
		return
	}

	volumeBackup, reader, err := c.backuper.OpenBackup(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open backup"})
		return
	}
	defer func() { _ = reader.Close() }()

	ctx.Header("Content-Disposition", `attachment; filename="`+volumeBackup.FileName+`"`)
	ctx.Header("Content-Type", "application/gzip")
	_, _ = io.Copy(ctx.Writer, reader)
}

// RestoreBackup
// @Summary Restore a volume backup into its container's mounts (overwrites data)
// @Tags docker
// @Param Authorization header string true "JWT token"
// @Param id path string true "Backup ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /docker/backups/{id}/restore [post]
func (c *Controller) RestoreBackup(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup id"})
		return
	}

	if err := c.backuper.RestoreBackup(ctx.Request.Context(), id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore backup"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "restored"})
}

// DeleteBackup
// @Summary Delete a volume backup (record and stored archive)
// @Tags docker
// @Param Authorization header string true "JWT token"
// @Param id path string true "Backup ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /docker/backups/{id} [delete]
func (c *Controller) DeleteBackup(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup id"})
		return
	}

	if err := c.backuper.DeleteBackup(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup"})
		return
	}

	ctx.Status(http.StatusOK)
}

// GetConfigs
// @Summary List scheduled backup configs
// @Tags docker
// @Produce json
// @Param Authorization header string true "JWT token"
// @Success 200 {object} map[string][]VolumeBackupConfig
// @Failure 401
// @Failure 500
// @Router /docker/configs [get]
func (c *Controller) GetConfigs(ctx *gin.Context) {
	configs, err := c.configService.ListConfigs()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backup configs"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"configs": configs})
}

// SaveConfig
// @Summary Create or update a container's scheduled backup config
// @Tags docker
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param request body VolumeBackupConfig true "Backup config"
// @Success 200 {object} VolumeBackupConfig
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /docker/configs [post]
func (c *Controller) SaveConfig(ctx *gin.Context) {
	var config VolumeBackupConfig
	if err := ctx.ShouldBindJSON(&config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := c.configService.SaveConfig(&config); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save backup config"})
		return
	}

	ctx.JSON(http.StatusOK, config)
}

// DeleteConfig
// @Summary Delete a scheduled backup config
// @Tags docker
// @Param Authorization header string true "JWT token"
// @Param id path string true "Config ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /docker/configs/{id} [delete]
func (c *Controller) DeleteConfig(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	if err := c.configService.DeleteConfig(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup config"})
		return
	}

	ctx.Status(http.StatusOK)
}
