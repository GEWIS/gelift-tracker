package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const adminAuthCookie = "admin_auth"

func adminPassword() string {
	return strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
}

func isAdminDocumentPath(path string) bool {
	if path == "/admin-login" || strings.HasPrefix(path, "/admin-login/") {
		return false
	}
	return path == "/admin" || path == "/admin/" || strings.HasPrefix(path, "/admin/")
}

func cookieMatchesAdmin(c echo.Context, adminPass string) bool {
	cookie, err := c.Cookie(adminAuthCookie)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(adminPass)) == 1
}

const contestantAuthCookie = "contestant_auth"

func contestantPassword() string {
	return strings.TrimSpace(os.Getenv("CONTESTANT_PASSWORD"))
}

func isContestantDocumentPath(path string) bool {
	if path == "/contestant-login" || strings.HasPrefix(path, "/contestant-login/") {
		return false
	}
	return path == "/contestant" || path == "/contestant/" || strings.HasPrefix(path, "/contestant/")
}

func cookieMatchesContestant(c echo.Context, contestantPass string) bool {
	cookie, err := c.Cookie(contestantAuthCookie)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(contestantPass)) == 1
}

func contestantAuthorized(c echo.Context) bool {
	pass := contestantPassword()
	if pass == "" {
		return true
	}
	return cookieMatchesContestant(c, pass)
}

func protectedDocumentGate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				return next(c)
			}
			path := c.Request().URL.Path
			if isAdminDocumentPath(path) {
				if ap := adminPassword(); ap != "" && !cookieMatchesAdmin(c, ap) {
					return c.Redirect(http.StatusFound, "/admin-login")
				}
				return next(c)
			}
			if isContestantDocumentPath(path) {
				if cp := contestantPassword(); cp != "" && !cookieMatchesContestant(c, cp) {
					return c.Redirect(http.StatusFound, "/contestant-login")
				}
				return next(c)
			}
			return next(c)
		}
	}
}

type contestantLinkDTO struct {
	Label  string `json:"label"`
	Inline string `json:"inline"`
}

type contestantFileRow struct {
	Label  string `json:"label"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Inline string `json:"inline"`
	Base64 string `json:"base64"`
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func contestantsJSONPath() string {
	p := strings.TrimSpace(os.Getenv("CONTESTANTS_JSON_PATH"))
	if p == "" {
		return "/data/contestants.json"
	}
	return p
}

func parseContestantJSON(raw []byte) ([]contestantLinkDTO, error) {
	raw = bytes.TrimPrefix(bytes.TrimSpace(raw), []byte("\xef\xbb\xbf"))
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty contestant config")
	}
	switch raw[0] {
	case '[':
		var rows []contestantFileRow
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		out := make([]contestantLinkDTO, 0, len(rows))
		for _, r := range rows {
			label := firstNonEmpty(r.Label, r.Name, r.ID)
			inline := firstNonEmpty(r.Inline, r.Base64)
			if label == "" || inline == "" {
				continue
			}
			out = append(out, contestantLinkDTO{Label: label, Inline: inline})
		}
		return out, nil
	case '{':
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]contestantLinkDTO, 0, len(keys))
		for _, k := range keys {
			out = append(out, contestantLinkDTO{Label: k, Inline: m[k]})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("contestant config JSON must be an object or array")
	}
}

func readContestantLinksForAPI() ([]contestantLinkDTO, string, error) {
	path := contestantsJSONPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []contestantLinkDTO{}, fmt.Sprintf("Contestant config file not found (%s)", path), nil
		}
		return nil, "", err
	}
	rows, err := parseContestantJSON(raw)
	if err != nil {
		return nil, "", err
	}
	return rows, "", nil
}

type MqttPayload struct {
	Type      string  `json:"_type"`
	Battery   int     `json:"batt"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Timestamp int     `json:"tst"`
	Velocity  int     `json:"vel"`
}

type LocationPoint struct {
	gorm.Model
	Team      string  `json:"team"`
	User      string  `json:"user"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Battery   int     `json:"battery"`
	Velocity  int     `json:"velocity"`
	Timestamp int     `json:"timestamp"`
	PacketID  uint16  `json:"pid"`
}

func connectMqtt(db *gorm.DB) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	u, err := url.Parse(os.Getenv("MQTT_URL"))
	if err != nil {
		panic(err)
	}

	cliCfg := autopaho.ClientConfig{
		ConnectUsername:               os.Getenv("MQTT_USERNAME"),
		ConnectPassword:               []byte(os.Getenv("MQTT_PASSWORD")),
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		// SessionExpiryInterval - Seconds that a session will survive after disconnection.
		// It is important to set this because otherwise, any queued messages will be lost if the connection drops and
		// the server will not queue messages while it is down. The specific setting will depend upon your needs
		// (60 = 1 minute, 3600 = 1 hour, 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			fmt.Println("mqtt connection up")
			// Subscribing in the OnConnectionUp callback is recommended (ensures the subscription is reestablished if
			// the connection drops)
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: "owntracks/+/+", QoS: 1},
				},
			}); err != nil {
				fmt.Printf("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
			}
			fmt.Println("mqtt subscription made")
		},
		OnConnectError: func(err error) { fmt.Printf("error whilst attempting connection: %s\n", err) },
		// eclipse/paho.golang/paho provides base mqtt functionality, the below config will be passed in for each connection
		ClientConfig: paho.ClientConfig{
			// If you are using QOS 1/2, then it's important to specify a client id (which must be unique)
			ClientID: "go-server",
			// OnPublishReceived is a slice of functions that will be called when a message is received.
			// You can write the function(s) yourself or use the supplied Router
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					var payload MqttPayload
					if err := json.Unmarshal(pr.Packet.Payload, &payload); err != nil {
						return false, err
					}

					if payload.Type != "location" {
						fmt.Printf("Received packet of type %s, discarding...\n", payload.Type)
						return true, nil
					}

					var splittedTopic = strings.Split(pr.Packet.Topic, "/")

					// If this packet, or this timestamp - user combo is already logged, do not try again
					amount := db.Where(
						"timestamp = ? AND user = ?",
						payload.Timestamp, splittedTopic[1],
					).First(&LocationPoint{}).RowsAffected

					if amount == 1 {
						fmt.Printf("Duplicate packet received from %s / %s, discarding...\n", splittedTopic[1], splittedTopic[2])
						return true, nil
					}

					fmt.Printf("Topic: %s; Lat: %g; Lon: %g; Batt: %d%% \n", pr.Packet.Topic, payload.Latitude, payload.Longitude, payload.Battery)

					err := db.Create(&LocationPoint{
						Team:      splittedTopic[1],
						User:      splittedTopic[2],
						Latitude:  payload.Latitude,
						Longitude: payload.Longitude,
						Battery:   payload.Battery,
						Velocity:  payload.Velocity,
						Timestamp: payload.Timestamp,
						PacketID:  pr.Packet.PacketID,
					}).Error

					if err != nil {
						fmt.Println(err)
					}

					return true, nil
				}},
			OnClientError: func(err error) { fmt.Printf("client error: %s\n", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		},
	}

	c, err := autopaho.NewConnection(ctx, cliCfg) // starts process; will reconnect until context cancelled
	if err != nil {
		panic(err)
	}
	// Wait for the connection to come up
	if err = c.AwaitConnection(ctx); err != nil {
		panic(err)
	}

	fmt.Println("mqtt client running (waiting until disconnect or shutdown signal)")
	<-c.Done()
	fmt.Println("mqtt client stopped")
}

func listenAddr() string {
	if v := os.Getenv("HTTP_LISTEN"); v != "" {
		return v
	}
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return ":1323"
}

func startHTTP(db *gorm.DB) {
	e := echo.New()

	tracksHandler := func(c echo.Context) error {
		var locations []LocationPoint
		if err := db.Find(&locations).Error; err != nil {
			return err
		}
		return c.JSON(http.StatusOK, locations)
	}

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/tracks", tracksHandler)
	e.GET("/api/tracks", tracksHandler)

	adminPass := adminPassword()
	e.POST("/api/admin/login", func(c echo.Context) error {
		if adminPass == "" {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": "Admin login is not configured (set ADMIN_PASSWORD)",
			})
		}
		var body struct {
			Password string `json:"password"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(body.Password)), []byte(adminPass)) != 1 {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "wrong password"})
		}
		c.SetCookie(&http.Cookie{
			Name:     adminAuthCookie,
			Value:    adminPass,
			Path:     "/",
			MaxAge:   60 * 60 * 24 * 7,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   c.Scheme() == "https",
		})
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	})
	e.GET("/api/admin/session", func(c echo.Context) error {
		if adminPass == "" {
			return c.JSON(http.StatusOK, map[string]bool{
				"authenticated": true,
				"configured":    false,
			})
		}
		return c.JSON(http.StatusOK, map[string]bool{
			"authenticated": cookieMatchesAdmin(c, adminPass),
			"configured":    true,
		})
	})

	contestantPass := contestantPassword()
	e.POST("/api/contestant/login", func(c echo.Context) error {
		if contestantPass == "" {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": "Contestant login is not configured (set CONTESTANT_PASSWORD)",
			})
		}
		var body struct {
			Password string `json:"password"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(body.Password)), []byte(contestantPass)) != 1 {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "wrong password"})
		}
		c.SetCookie(&http.Cookie{
			Name:     contestantAuthCookie,
			Value:    contestantPass,
			Path:     "/",
			MaxAge:   60 * 60 * 24 * 7,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   c.Scheme() == "https",
		})
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	})
	e.GET("/api/contestant/session", func(c echo.Context) error {
		if contestantPass == "" {
			return c.JSON(http.StatusOK, map[string]bool{
				"authenticated": true,
				"configured":    false,
			})
		}
		return c.JSON(http.StatusOK, map[string]bool{
			"authenticated": cookieMatchesContestant(c, contestantPass),
			"configured":    true,
		})
	})
	e.GET("/api/contestant/links", func(c echo.Context) error {
		if !contestantAuthorized(c) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		rows, warn, err := readContestantLinksForAPI()
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

	e.Use(protectedDocumentGate())

	staticDir := os.Getenv("STATIC_DIR")
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
	} else {
		e.GET("/", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!")
		})
	}

	e.Logger.Fatal(e.Start(listenAddr()))
}

func openGormDB() (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	driver := strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_DRIVER")))
	pgURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	sqlitePath := strings.TrimSpace(os.Getenv("DATABASE"))

	switch driver {
	case "", "auto":
		if pgURL != "" {
			driver = "postgres"
		} else {
			driver = "sqlite"
		}
	case "postgresql":
		driver = "postgres"
	}

	switch driver {
	case "postgres":
		dsn := pgURL
		if dsn == "" {
			dsn = sqlitePath
		}
		if dsn == "" {
			return nil, fmt.Errorf("postgres selected but DATABASE_URL and DATABASE are empty")
		}
		return gorm.Open(postgres.Open(dsn), cfg)
	case "sqlite":
		path := sqlitePath
		if path == "" {
			path = "gelift.db"
		}
		return gorm.Open(sqlite.Open(path), cfg)
	default:
		return nil, fmt.Errorf("unknown DATABASE_DRIVER %q (use sqlite, postgres, or auto)", driver)
	}
}

func main() {
	_ = godotenv.Load()

	fmt.Println("Connecting to the database...")
	db, err := openGormDB()
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&LocationPoint{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to the database")

	go connectMqtt(db)
	startHTTP(db)
}
