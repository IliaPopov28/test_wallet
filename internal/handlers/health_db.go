package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db DBPinger
}

func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", h.HealthCheck)
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status db": "ok"})
}
