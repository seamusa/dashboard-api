package routes

import (
	"net/http"
	"time"

	"github.com/chechetech/app/azure-go/repositories/database"
	"github.com/gin-gonic/gin"
)

type DatabaseHandler struct {
	repo database.DatabaseRepository
}

func NewDatabaseHandler(repo database.DatabaseRepository) *DatabaseHandler {
	return &DatabaseHandler{repo: repo}
}

// GET /database/metrics?start=...&end=...
func (h *DatabaseHandler) GetMetrics(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
		return
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
		return
	}

	minStartTime := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	if startTime.Before(minStartTime) {
		startTime = minStartTime
	}

	metrics, err := h.repo.GetMetrics(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// GET /database/querystore?start=...&end=...
func (h *DatabaseHandler) GetQueryStoreRuntime(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
		return
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
		return
	}

	minStartTime := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	if startTime.Before(minStartTime) {
		startTime = minStartTime
	}

	results, err := h.repo.GetQueryStoreRuntime(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

// GET /database/queries/:id
func (h *DatabaseHandler) GetQuerySqlText(c *gin.Context) {
	queryID := c.Param("id")
	if queryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queryid is required"})
		return
	}

	result, err := h.repo.GetQuerySqlText(c.Request.Context(), queryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func RegisterDatabaseRoutes(r *gin.Engine, repo database.DatabaseRepository) {
	handler := NewDatabaseHandler(repo)
	db := r.Group("/database")
	db.GET("/metrics", handler.GetMetrics)
	db.GET("/queries", handler.GetQueryStoreRuntime)
	db.GET("/queries/:id", handler.GetQuerySqlText) // fixed route param
}
