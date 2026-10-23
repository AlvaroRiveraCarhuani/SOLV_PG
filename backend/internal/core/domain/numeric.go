package domain

import "math"

// Uint64ToInt convierte un valor de uint64 a int saturando en math.MaxInt
// cuando el valor no es representable.
func Uint64ToInt(v uint64) int {
	if v > uint64(math.MaxInt) {
		return math.MaxInt
	}
	return int(v)
}

// Uint64ToInt64 convierte un valor de uint64 a int64 saturando en math.MaxInt64
// cuando el valor no es representable.
func Uint64ToInt64(v uint64) int64 {
	if v > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(v)
}
