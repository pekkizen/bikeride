package route

import (
	"fmt"
	"math"
	"testing"
)

var F float64 = 5
var Y float64 = -5

func BenchmarkCourse(b *testing.B) {
	y := 1.0
	lon := 1.0

	for n := 0; n < b.N; n++ {
		// y = course(lon, -Fsink)
		y = math.Mod(math.Atan2(lon, -F)+2*π, 2*π)
	}
	F = y
}

func BenchmarkDiv(b *testing.B) {
	y := 10000000.0
	c := 1.0 / 0.99999999
	for n := 0; n < b.N; n++ {
		// y = y / c
		// y = Fsink / c
		y = F * c
		// y = y * c
		// y = course2(lon, Fsink)
	}
	F = y
}

func BenchmarkCopysign(b *testing.B) {
	y := -1.
	x := 1.0
	for n := 0; n < b.N; n++ {
		// y = copysign(y, -x)
		y = math.Copysign(y, -x)
		x = y
	}
	F = y + x
}

func BenchmarkSgn(b *testing.B) {
	y := 1.
	x := 1.0
	for n := 0; n < b.N; n++ {
		y = Sign(-x * y)
		// y = Sgn2(-x * y)
		// y = Sgn3(-x * y)
		// y = Sgn4(-x * y)
	}
	F = y + x
}

func TestSGN(t *testing.T) {
	const one = 0x3FF0000000000000
	var x1, x2 float64
	y := 1.0
	for y = 10.0; math.Abs(y) > 0.05; y *= -0.75 {
		x1 = Sign(y)
		// x1 = Sgn2(y)
		// x1 = Sgn3(y)
		// x1 = SgnCopilot(y)
		x2 = math.Float64frombits(one | math.Float64bits(y)&(1<<63))
		if x1 != x2 {
			t.Errorf("\nx1: %v, x2:  %v  y: %f2.2\n", x1, x2, y)
			// println(x1, x2, y)

		}
		fmt.Printf("\nx1: %v, x2:  %v  y: %2.2f\n", x1, x2, y)
		f := math.Float64bits(1)
		fmt.Printf("y   %f f  %0x\n", y, f)
	}
}
