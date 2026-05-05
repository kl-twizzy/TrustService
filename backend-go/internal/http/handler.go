package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"seller-trust-map/backend-go/internal/domain"
	"seller-trust-map/backend-go/internal/service"
)

type Handler struct {
	trustService *service.TrustService
}

func NewHandler(trustService *service.TrustService) *Handler {
	return &Handler{trustService: trustService}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/api/v1/overview", h.overview)
	router.POST("/api/v1/trust/analyze", h.analyze)
	router.POST("/api/v1/trust/analyze-url", h.analyzeURL)
	router.GET("/api/v1/checks/recent", h.recentChecks)
}

func (h *Handler) analyze(c *gin.Context) {
	var req domain.AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.trustService.AnalyzeWithContext(c.Request.Context(), req, domain.AnalyzeContext{
		ClientID: clientIDFromRequest(c),
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) analyzeURL(c *gin.Context) {
	var req domain.AnalyzeURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.trustService.AnalyzeURLWithContext(c.Request.Context(), req.ProductURL, domain.AnalyzeContext{
		ClientID: clientIDFromRequest(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) recentChecks(c *gin.Context) {
	result, err := h.trustService.ListRecentChecksForClient(c.Request.Context(), 10, clientIDFromRequest(c))
	if err != nil {
		c.JSON(http.StatusOK, []domain.RecentCheck{})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) overview(c *gin.Context) {
	result, err := h.trustService.GetOverviewForClient(c.Request.Context(), clientIDFromRequest(c))
	if err != nil {
		c.JSON(http.StatusOK, domain.OverviewResponse{})
		return
	}

	c.JSON(http.StatusOK, result)
}

func clientIDFromRequest(c *gin.Context) string {
	value := strings.TrimSpace(c.GetHeader("X-Client-Id"))
	if value != "" {
		return value
	}
	return strings.TrimSpace(c.Query("client_id"))
}
