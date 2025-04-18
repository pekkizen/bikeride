package route

import (
	"testing"
)

var Fsink float64 = 5

func BenchmarkCourse(b *testing.B) {
	y := 1.0
	lon := 1.0

	for n := 0; n < b.N; n++ {
		y = course(lon, -Fsink)
		// y = math.Mod(math.Atan2(lon, -Fsink)+2*π, 2*π)
	}
	Fsink = y
}

func BenchmarkDiv(b *testing.B) {
	y := 10000000.0
	c := 1.0 / 0.99999999
	for n := 0; n < b.N; n++ {
		// y = y / c
		// y = Fsink / c
		// y = Fsink * c
		y = y * c
		// y = course2(lon, Fsink)
	}
	Fsink = y
}
