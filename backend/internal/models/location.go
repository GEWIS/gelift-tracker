package models

import "gorm.io/gorm"

// LocationPoint is one Owntracks location stored for map display.
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
