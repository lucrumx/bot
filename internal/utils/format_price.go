package utils

import (
	"math"
	"strconv"
)

// FormatPrice formats a price based on its magnitude, ensuring appropriate decimal places for readability.
func FormatPrice(price float64) string {
	switch {
	case price == 0:
		return "0"
	case math.Abs(price) >= 1000:
		return strconv.FormatFloat(price, 'f', 2, 64)
	case math.Abs(price) >= 1:
		return strconv.FormatFloat(price, 'f', 4, 64)
	case math.Abs(price) >= 0.01:
		return strconv.FormatFloat(price, 'f', 6, 64)
	default:
		return strconv.FormatFloat(price, 'f', 8, 64)
	}
}
