package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"backend/internal/models"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"gorm.io/gorm"
)

// Run connects to MQTT, subscribes to Owntracks topics, and persists locations until shutdown.
// It blocks until the connection manager exits (normally after SIGINT/SIGTERM).
func Run(db *gorm.DB) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	u, err := url.Parse(os.Getenv("MQTT_URL"))
	if err != nil {
		panic(err)
	}

	mqttClientID := os.Getenv("MQTT_CLIENT_ID")
	if mqttClientID == "" {
		mqttClientID = "go-server"
	}

	cliCfg := autopaho.ClientConfig{
		ConnectUsername:               os.Getenv("MQTT_USERNAME"),
		ConnectPassword:               []byte(os.Getenv("MQTT_PASSWORD")),
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			fmt.Println("mqtt connection up")
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
		ClientConfig: paho.ClientConfig{
			ClientID: mqttClientID,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				locationHandler(db),
			},
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

	c, err := autopaho.NewConnection(ctx, cliCfg)
	if err != nil {
		panic(err)
	}
	if err = c.AwaitConnection(ctx); err != nil {
		panic(err)
	}

	fmt.Println("mqtt client running (waiting until disconnect or shutdown signal)")
	<-c.Done()
	fmt.Println("mqtt client stopped")
}

func locationHandler(db *gorm.DB) func(paho.PublishReceived) (bool, error) {
	return func(pr paho.PublishReceived) (bool, error) {
		var payload Payload
		if err := json.Unmarshal(pr.Packet.Payload, &payload); err != nil {
			return false, err
		}

		if payload.Type != "location" {
			fmt.Printf("Received packet of type %s, discarding...\n", payload.Type)
			return true, nil
		}

		parts := strings.Split(pr.Packet.Topic, "/")
		if len(parts) < 3 {
			return true, nil
		}

		amount := db.Where(
			"timestamp = ? AND user = ?",
			payload.Timestamp, parts[2],
		).First(&models.LocationPoint{}).RowsAffected

		fmt.Printf("Received packet from %s / %s, on timestamp %d. Found %d in the database.\n", parts[1], parts[2], payload.Timestamp, amount)

		if amount >= 1 {
			fmt.Printf("Duplicate packet received from %s / %s, discarding...\n", parts[1], parts[2])
			return true, nil
		}

		fmt.Printf("Topic: %s; Lat: %g; Lon: %g; Batt: %d%% \n", pr.Packet.Topic, payload.Latitude, payload.Longitude, payload.Battery)

		if err := db.Create(&models.LocationPoint{
			Team:      parts[1],
			User:      parts[2],
			Latitude:  payload.Latitude,
			Longitude: payload.Longitude,
			Battery:   payload.Battery,
			Velocity:  payload.Velocity,
			Timestamp: payload.Timestamp,
			PacketID:  pr.Packet.PacketID,
		}).Error; err != nil {
			fmt.Println(err)
		}

		return true, nil
	}
}
