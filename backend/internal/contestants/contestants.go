package contestants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Link is one Owntracks inline configuration entry exposed to the contestant UI.
type Link struct {
	Label  string `json:"label"`
	Inline string `json:"inline"`
}

type fileRow struct {
	Label  string `json:"label"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Inline string `json:"inline"`
	Base64 string `json:"base64"`
}

// Reader loads contestant link definitions from a JSON file path.
type Reader struct {
	Path string
}

// Load reads and parses the JSON file. The warning string is non-fatal (e.g. missing file).
func (r *Reader) Load() (links []Link, warning string, err error) {
	raw, err := os.ReadFile(r.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Link{}, fmt.Sprintf("Contestant config file not found (%s)", r.Path), nil
		}
		return nil, "", err
	}
	links, err = parseJSON(raw)
	if err != nil {
		return nil, "", err
	}
	return links, "", nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func parseJSON(raw []byte) ([]Link, error) {
	raw = bytes.TrimPrefix(bytes.TrimSpace(raw), []byte("\xef\xbb\xbf"))
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty contestant config")
	}
	switch raw[0] {
	case '[':
		var rows []fileRow
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		out := make([]Link, 0, len(rows))
		for _, row := range rows {
			label := firstNonEmpty(row.Label, row.Name, row.ID)
			inline := firstNonEmpty(row.Inline, row.Base64)
			if label == "" || inline == "" {
				continue
			}
			out = append(out, Link{Label: label, Inline: inline})
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
		out := make([]Link, 0, len(keys))
		for _, k := range keys {
			out = append(out, Link{Label: k, Inline: m[k]})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("contestant config JSON must be an object or array")
	}
}
