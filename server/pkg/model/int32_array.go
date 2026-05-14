package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

type Int32Array []int32

func (a *Int32Array) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	var s string
	switch v := value.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("Int32Array.Scan: unsupported type %T", value)
	}

	s = strings.TrimSpace(s)
	if s == "{}" || s == "" {
		*a = []int32{}
		return nil
	}

	s = strings.Trim(s, "{}")
	if s == "" {
		*a = []int32{}
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]int32, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.ParseInt(strings.TrimSpace(p), 10, 32)
		if err != nil {
			return fmt.Errorf("Int32Array.Scan: %w", err)
		}
		result = append(result, int32(n))
	}
	*a = result
	return nil
}

func (a Int32Array) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	if len(a) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, n := range a {
		parts[i] = strconv.FormatInt(int64(n), 10)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}
