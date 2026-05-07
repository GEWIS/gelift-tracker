package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	AdminCookieName      = "admin_auth"
	ContestantCookieName = "contestant_auth"
)

// ConstantCookieMatches reports whether the named cookie equals secret using constant-time comparison.
func ConstantCookieMatches(c echo.Context, cookieName, secret string) bool {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(secret)) == 1
}

// SetSessionCookie sets the password-session cookie (HttpOnly, SameSite=Lax, Secure on HTTPS).
func SetSessionCookie(c echo.Context, cookieName, value string) {
	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
	})
}

// PasswordLogin binds JSON {"password"}, verifies against secret, and sets the session cookie.
func PasswordLogin(c echo.Context, cookieName, secret, notConfiguredMsg string) error {
	if secret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": notConfiguredMsg})
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(body.Password)), []byte(secret)) != 1 {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "wrong password"})
	}
	SetSessionCookie(c, cookieName, secret)
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// PasswordSession returns configured/authenticated flags for SPA gates.
func PasswordSession(c echo.Context, cookieName, secret string) error {
	if secret == "" {
		return c.JSON(http.StatusOK, map[string]bool{
			"authenticated": true,
			"configured":    false,
		})
	}
	ok := ConstantCookieMatches(c, cookieName, secret)
	return c.JSON(http.StatusOK, map[string]bool{
		"authenticated": ok,
		"configured":    true,
	})
}

// IsAdminDocumentPath reports paths that require admin cookie for GET document navigations.
func IsAdminDocumentPath(path string) bool {
	if path == "/admin-login" || strings.HasPrefix(path, "/admin-login/") {
		return false
	}
	return path == "/admin" || path == "/admin/" || strings.HasPrefix(path, "/admin/")
}

// IsContestantDocumentPath reports paths that require contestant cookie for GET document navigations.
func IsContestantDocumentPath(path string) bool {
	if path == "/contestant-login" || strings.HasPrefix(path, "/contestant-login/") {
		return false
	}
	return path == "/contestant" || path == "/contestant/" || strings.HasPrefix(path, "/contestant/")
}

// ContestantAPIAuthorized is true when contestant auth is disabled or the session cookie is valid.
func ContestantAPIAuthorized(c echo.Context, contestantSecret string) bool {
	if contestantSecret == "" {
		return true
	}
	return ConstantCookieMatches(c, ContestantCookieName, contestantSecret)
}

// DocumentGate redirects unauthenticated GET requests away from protected SPA routes.
func DocumentGate(adminSecret, contestantSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}
			path := c.Request().URL.Path
			if IsAdminDocumentPath(path) {
				if adminSecret != "" && !ConstantCookieMatches(c, AdminCookieName, adminSecret) {
					return c.Redirect(http.StatusFound, "/admin-login")
				}
				return next(c)
			}
			if IsContestantDocumentPath(path) {
				if contestantSecret != "" && !ConstantCookieMatches(c, ContestantCookieName, contestantSecret) {
					return c.Redirect(http.StatusFound, "/contestant-login")
				}
				return next(c)
			}
			return next(c)
		}
	}
}
