package app

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed web/*
var dashboardFS embed.FS

func registerDashboard(router *gin.Engine) error {
	subFS, err := fs.Sub(dashboardFS, "web")
	if err != nil {
		return err
	}

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
	})
	router.StaticFS("/dashboard", http.FS(subFS))
	return nil
}
