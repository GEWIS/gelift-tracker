package models

import "gorm.io/gorm"

// Setting is a simple key–value row for app configuration (edited in admin).
type Setting struct {
	gorm.Model
	Key   string `gorm:"size:512;uniqueIndex;not null"`
	Value string `gorm:"type:text"`
}

// SettingEntry is the JSON shape exposed to clients (key plus value only).
type SettingEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
