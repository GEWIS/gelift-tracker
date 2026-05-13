package httpapi

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/contestants"
	"backend/internal/models"
	"backend/internal/timewindow"

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
	registerSettings(e, dep.DB, dep.Config.AdminPassword)
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
	e.GET("/api/tracks", func(c echo.Context) error {
		startSec, endSec, err := timewindow.LoadUnixInclusive(db)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		q := db.Model(&models.LocationPoint{})
		if startSec > 0 {
			q = q.Where("timestamp >= ?", int(startSec))
		}
		if endSec > 0 {
			q = q.Where("timestamp <= ?", int(endSec))
		}
		var locations []models.LocationPoint
		if err := q.Find(&locations).Error; err != nil {
			return err
		}
		return c.JSON(http.StatusOK, locations)
	})

	e.GET("/api/event-window", func(c echo.Context) error {
		w, err := timewindow.LoadEventWindow(db)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, w)
	})
}

// adminAuthorized matches DocumentGate: open when ADMIN_PASSWORD is unset; otherwise session cookie must match.
func adminAuthorized(c echo.Context, adminPass string) bool {
	if adminPass == "" {
		return true
	}
	return auth.ConstantCookieMatches(c, auth.AdminCookieName, adminPass)
}

func registerSettings(e *echo.Echo, db *gorm.DB, adminPass string) {
	e.GET("/api/settings", func(c echo.Context) error {
		var rows []models.Setting
		if err := db.Order("key asc").Find(&rows).Error; err != nil {
			return err
		}
		out := make([]models.SettingEntry, 0, len(rows))
		for _, r := range rows {
			out = append(out, models.SettingEntry{Key: r.Key, Value: r.Value})
		}
		return c.JSON(http.StatusOK, out)
	})

	e.PUT("/api/admin/settings", func(c echo.Context) error {
		if !adminAuthorized(c, adminPass) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		var body struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		key := strings.TrimSpace(body.Key)
		if key == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
		}
		var row models.Setting
		err := db.Where("key = ?", key).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = models.Setting{Key: key, Value: body.Value}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			row.Value = body.Value
			if err := db.Save(&row).Error; err != nil {
				return err
			}
		}
		return c.JSON(http.StatusOK, models.SettingEntry{Key: row.Key, Value: row.Value})
	})

	e.DELETE("/api/admin/settings/:key", func(c echo.Context) error {
		if !adminAuthorized(c, adminPass) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		key := strings.TrimSpace(c.Param("key"))
		if key == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
		}
		if res := db.Where("key = ?", key).Delete(&models.Setting{}); res.Error != nil {
			return res.Error
		}
		return c.NoContent(http.StatusNoContent)
	})
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
