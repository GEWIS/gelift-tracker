package timewindow

import (
	"fmt"
	"strings"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

const (
	StartKey = "start_time"
	EndKey   = "end_time"
)

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

func parseDateStringToUnix(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	// Try formats from most specific to least; all interpreted in UTC (or with explicit offset in string).
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		time.DateOnly,
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, time.UTC)
		if err == nil {
			return t.Unix(), nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return 0, fmt.Errorf("parse time %q: %w (use RFC 3339 like 2006-05-13T15:04:05Z or date 2006-05-13)", s, lastErr)
	}
	return 0, fmt.Errorf("parse time %q", s)
}
