package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type LessonQuestion struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Min      *int     `json:"min,omitempty"`
	Max      *int     `json:"max,omitempty"`
}

type LessonFormSchema struct {
	Title     string           `json:"title"`
	Questions []LessonQuestion `json:"questions"`
}

type ChecklistItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	IsDone  bool   `json:"is_done"`
}

type LessonAnswers json.RawMessage
type ChecklistItems []ChecklistItem
type JSONStringArray []string
type Uint64Array []uint64

func (s LessonFormSchema) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (a LessonAnswers) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	return string(a), nil
}

func (a JSONStringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *LessonAnswers) Scan(value any) error {
	if value == nil {
		*a = LessonAnswers{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*a = make(LessonAnswers, len(v))
		copy(*a, v)
	case string:
		*a = LessonAnswers(v)
	default:
		return errors.New("type assertion to []byte or string failed")
	}
	return nil
}

func (s *LessonFormSchema) Scan(value any) error {
	if value == nil {
		*s = LessonFormSchema{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, s)
}

func (a *Uint64Array) Scan(value any) error {
	if value == nil {
		*a = Uint64Array{}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("Uint64Array.Scan: expected string, got %T", value)
	}

	// Quitar llaves: "{1,2,3}" → "1,2,3"
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")

	if str == "" {
		*a = Uint64Array{}
		return nil
	}

	parts := strings.Split(str, ",")
	result := make(Uint64Array, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return fmt.Errorf("Uint64Array.Scan: failed to parse %q: %w", p, err)
		}
		result = append(result, n)
	}

	*a = result
	return nil
}

func (a *JSONStringArray) Scan(value any) error {
	if value == nil {
		*a = JSONStringArray{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, a)
}

func (c ChecklistItems) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	return json.Marshal(c)
}

func (c *ChecklistItems) Scan(value any) error {
	if value == nil {
		*c = ChecklistItems{}
		return nil
	}

	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("ChecklistItems.Scan: unsupported type %T", value)
	}

	return json.Unmarshal(b, c)
}

func (a Uint64Array) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = strconv.FormatUint(v, 10)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}
