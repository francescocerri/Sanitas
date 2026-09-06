package user

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray maps a Go []string to/from a Postgres text[] column. GORM's
// database/sql based Postgres driver has no native array support — this
// hand-rolled Scanner/Valuer avoids pulling in lib/pq (already discouraged
// elsewhere in this project, see the driver policy in CLAUDE.md) just for
// the one type this needs (roles.permissions).
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	quoted := make([]string, len(a))
	for i, s := range a {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		quoted[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}", nil
}

func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var text string
	switch v := src.(type) {
	case string:
		text = v
	case []byte:
		text = string(v)
	default:
		return fmt.Errorf("pgarray: unsupported scan type %T", src)
	}

	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return fmt.Errorf("pgarray: unexpected array literal %q", text)
	}
	inner := text[1 : len(text)-1]
	if inner == "" {
		*a = StringArray{}
		return nil
	}

	result := StringArray{}
	var current strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range inner {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			result = append(result, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	result = append(result, current.String())
	*a = result
	return nil
}
