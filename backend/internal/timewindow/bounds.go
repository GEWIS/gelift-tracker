package timewindow

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

const (
	StartKey = "start_time"
	EndKey   = "end_time"
	// SettingsTimezone is the wall-clock zone for date/time strings without an explicit offset.
	SettingsTimezone = "Europe/Amsterdam"
)

var (
	settingsLocOnce sync.Once
	settingsLoc     *time.Location
	settingsLocErr  error
)

func settingsLocation() (*time.Location, error) {
	settingsLocOnce.Do(func() {
		settingsLoc, settingsLocErr = time.LoadLocation(SettingsTimezone)
	})
	if settingsLocErr != nil {
		return nil, settingsLocErr
	}
	return settingsLoc, nil
}

// LoadUnixInclusive returns inclusive Unix second bounds from settings. A zero bound means unrestricted on that side.
func LoadUnixInclusive(db *gorm.DB) (startSec, endSec int64, err error) {
	var rows []models.Setting
	if err := db.Where("key IN ?", []string{StartKey, EndKey}).Find(&rows).Error; err != nil {
		return 0, 0, err
	}
	for _, r := range rows {
		switch r.Key {
		case StartKey:
			v, e := parseDateStringToUnix(r.Value)
			if e != nil {
				return 0, 0, fmt.Errorf("setting %q: %w", StartKey, e)
			}
			startSec = v
		case EndKey:
			v, e := parseDateStringToUnix(r.Value)
			if e != nil {
				return 0, 0, fmt.Errorf("setting %q: %w", EndKey, e)
			}
			endSec = v
		}
	}
	return startSec, endSec, nil
}

// EventWindow carries resolved UTC Unix seconds for the map UI (same instants as /api/tracks).
type EventWindow struct {
	StartUnix *int64 `json:"start_unix,omitempty"`
	EndUnix   *int64 `json:"end_unix,omitempty"`
}

// LoadEventWindow returns parsed bounds for JSON APIs; omits fields when unset.
func LoadEventWindow(db *gorm.DB) (*EventWindow, error) {
	start, end, err := LoadUnixInclusive(db)
	if err != nil {
		return nil, err
	}
	out := &EventWindow{}
	if start > 0 {
		s := start
		out.StartUnix = &s
	}
	if end > 0 {
		e := end
		out.EndUnix = &e
	}
	return out, nil
}

func parseDateStringToUnix(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	loc, err := settingsLocation()
	if err != nil {
		return 0, fmt.Errorf("load timezone %q: %w", SettingsTimezone, err)
	}

	// Explicit offset or Z: honour the string (absolute instant).
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.Unix(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix(), nil
	}

	// No zone in string: interpret as local wall time in Europe/Amsterdam (DST-aware).
	naiveLayouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		time.DateOnly,
	}
	var lastErr error
	for _, layout := range naiveLayouts {
		t, err := time.ParseInLocation(layout, s, loc)
		if err == nil {
			return t.Unix(), nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return 0, fmt.Errorf(
			"parse time %q: %w (naive times use %s; or use RFC 3339 with zone, e.g. 2006-05-13T15:04:05+02:00)",
			s, lastErr, SettingsTimezone,
		)
	}
	return 0, fmt.Errorf("parse time %q", s)
}
