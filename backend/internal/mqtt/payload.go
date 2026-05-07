package mqtt

// Payload mirrors Owntracks JSON payload fields used by this service.
type Payload struct {
	Type      string  `json:"_type"`
	Battery   int     `json:"batt"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Timestamp int     `json:"tst"`
	Velocity  int     `json:"vel"`
}
