package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

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
	User      string  `json:"user"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Battery   int     `json:"battery"`
	Velocity  int     `json:"velocity"`
	Timestamp int     `json:"timestamp"`
	PacketID  uint16  `gorm:"unique" json:"pid"`
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
					err = json.Unmarshal(pr.Packet.Payload, &payload)

					if payload.Type != "location" {
						return true, nil
					}

					var splittedTopic = strings.Split(pr.Packet.Topic, "/")

					// If this packet, or this timestamp - user combo is already logged, do not try again
					amount := db.Where(
						"packet_id = ? OR (timestamp = ? AND user = ?)",
						pr.Packet.PacketID, payload.Timestamp, splittedTopic[1],
					).First(&LocationPoint{}).RowsAffected

					if amount == 1 {
						return true, nil
					}

					fmt.Printf("Topic: %s; Lat: %g; Lon: %g; Batt: %d%% \n", pr.Packet.Topic, payload.Latitude, payload.Longitude, payload.Battery)

					err := db.Create(&LocationPoint{
						User:      splittedTopic[1],
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

	fmt.Println("signal caught - exiting")
	<-c.Done()
}

func startHTTP(db *gorm.DB) {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/tracks", func(c echo.Context) error {
		var locations []LocationPoint

		err := db.Find(&locations).Error

		if err != nil {
			panic(err)
		}

		return c.JSON(http.StatusOK, locations)
	})

	e.Logger.Fatal(e.Start(":1323"))
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	fmt.Println("Connecting to the database...")
	db, err := gorm.Open(sqlite.Open(os.Getenv("DATABASE")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

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
