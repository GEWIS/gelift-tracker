package httpapi

import (
	"net/http"
	"os"
	"strings"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/contestants"
	"backend/internal/models"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// Dependencies bundles services needed to register HTTP routes.
type Dependencies struct {
	DB          *gorm.DB
	Config      config.Config
	Contestants *contestants.Reader
}

// Register wires all Echo routes and middleware (caller runs e.Start).
func Register(e *echo.Echo, dep Dependencies) {
	registerHealth(e)
	registerTracks(e, dep.DB)
	registerAdminAuth(e, dep.Config.AdminPassword)
	registerContestantAuth(e, dep.Config, dep.Contestants)
	e.Use(auth.DocumentGate(dep.Config.AdminPassword, dep.Config.ContestantPassword))
	registerStatic(e, dep.Config.StaticDir)
}

func registerHealth(e *echo.Echo) {
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
}

func registerTracks(e *echo.Echo, db *gorm.DB) {
	h := func(c echo.Context) error {
		var locations []models.LocationPoint
		if err := db.Find(&locations).Error; err != nil {
			return err
		}
		return c.JSON(http.StatusOK, locations)
	}
	e.GET("/tracks", h)
	e.GET("/api/tracks", h)
}

func registerAdminAuth(e *echo.Echo, adminPass string) {
	e.POST("/api/admin/login", func(c echo.Context) error {
		return auth.PasswordLogin(c, auth.AdminCookieName, adminPass,
			"Admin login is not configured (set ADMIN_PASSWORD)")
	})
	e.GET("/api/admin/session", func(c echo.Context) error {
		return auth.PasswordSession(c, auth.AdminCookieName, adminPass)
	})
}

func registerContestantAuth(e *echo.Echo, cfg config.Config, reader *contestants.Reader) {
	cp := cfg.ContestantPassword

	e.POST("/api/contestant/login", func(c echo.Context) error {
		return auth.PasswordLogin(c, auth.ContestantCookieName, cp,
			"Contestant login is not configured (set CONTESTANT_PASSWORD)")
	})
	e.GET("/api/contestant/session", func(c echo.Context) error {
		return auth.PasswordSession(c, auth.ContestantCookieName, cp)
	})
	e.GET("/api/contestant/links", func(c echo.Context) error {
		if !auth.ContestantAPIAuthorized(c, cp) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		rows, warn, err := reader.Load()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read contestant configuration"})
		}
		payload := map[string]interface{}{
			"contestants": rows,
		}
		if warn != "" {
			payload["error"] = warn
		}
		return c.JSON(http.StatusOK, payload)
	})
}

func registerStatic(e *echo.Echo, staticDir string) {
	if staticDir == "" {
		staticDir = "dist"
	}
	if st, err := os.Stat(staticDir); err == nil && st.IsDir() {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:   staticDir,
			HTML5:  true,
			Browse: false,
			Skipper: func(c echo.Context) bool {
				p := c.Request().URL.Path
				return p == "/healthz" || p == "/tracks" || strings.HasPrefix(p, "/api/")
			},
		}))
		return
	}
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
}
