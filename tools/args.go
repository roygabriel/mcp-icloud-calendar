package tools

import (
	"fmt"
	"math"
)

type jsonNumber interface {
	Float64() (float64, error)
}

func optionalNonNegativeInt(args map[string]interface{}, name string) (int, bool, error) {
	raw, exists := args[name]
	if !exists || raw == nil {
		return 0, false, nil
	}

	var value float64
	switch v := raw.(type) {
	case float64:
		value = v
	case float32:
		value = float64(v)
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	case jsonNumber:
		f, err := v.Float64()
		if err != nil {
			return 0, true, err
		}
		value = f
	default:
		return 0, true, fmt.Errorf("%s must be a number", name)
	}

	if value < 0 {
		return 0, true, fmt.Errorf("%s must be non-negative", name)
	}
	if math.Trunc(value) != value {
		return 0, true, fmt.Errorf("%s must be an integer", name)
	}

	return int(value), true, nil
}
