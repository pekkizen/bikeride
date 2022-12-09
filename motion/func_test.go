package motion

import (
	"math"
	"testing"

	// "math/rand"
	rand "prng"
)

var fsink float64

var Cal = Calculator()

// func (c *BikeCalc) functionTst(v float64) float64 {
// 	c.callsFunc++
// 	return v*(c.fGR + c.cDrag * math.Abs(v+c.wind)*(v+c.wind)) - c.power
// }

func signedSquare(x float64) float64 {
	return math.Float64frombits(math.Float64bits(x*x) | (math.Float64bits(x) & (1 << 63)))
}

// var rng = rand.New(0)
// var rng = rand.New(rand.NewSource(99))

func fastExp(x float64) float64 {
	const o4096 = 1.0 / 4096.0
	x = 1 + x*o4096
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	return x
}
func BenchmarkFastExp(b *testing.B) {

	y := 10.0
	for n := 0; n < b.N; n++ {

		y += fastExp(1)
	}
	fsink += y
}
func BenchmarkExp(b *testing.B) {
	y := 10.0
	for n := 0; n < b.N; n++ {

		y += math.Exp(1)
	}
	fsink += y
}

func BenchmarkRand(b *testing.B) {
	sign := 1.0
	y := 0.0
	for n := 0; n < b.N; n++ {
		x := rand.Float64() * sign
		sign *= -1
		y += x
	}
	fsink = y
}
func BenchmarkNull(b *testing.B) {
	var k int
	for n := 0; n < b.N; n++ {
		k += n
	}
	fsink += float64(k)
}

func BenchmarkSignedSquare(b *testing.B) {
	sign := 100.0
	y := 10.0
	for n := 0; n < b.N; n++ {
		x := rand.Float64() * sign
		sign *= -1
		y += signedSquare(x)
	}
	fsink += y
}

func BenchmarkFunction(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(0.05)
	Cal.SetWind(-3)
	Cal.SetPower(100)
	y := 0.0
	for n := 0; n < b.N; n++ {
		v := float64(n&7 + 1)
		y += Cal.function(v)
	}
	fsink = y
}

func BenchmarkSetGrade(b *testing.B) {
	Cal.SetWeight(100)
	for n := 0; n < b.N; n++ {
		Cal.SetGrade(0.01)
	}
}

func BenchmarkNewtonRaphson(b *testing.B) {
	// var newton =  (*BikeCalc).NewtonRaphson
	Cal.SetWeight(100)
	Cal.SetGrade(-0.02)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	v := 5.0
	for n := 0; n < b.N; n++ {
		p := float64(n % 128)
		v, _ = Cal.NewtonRaphson(p, 0.1, v)
		v += 1.2
	}
	fsink = v
}
func BenchmarkNewtonRaphsonHalley(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(0.01)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	v := 5.0
	for n := 0; n < b.N; n++ {
		// v = float64(n % 9 + 1)
		p := float64(n % 128)
		v, _ = Cal.NewtonRaphsonHalley(p, 0.05, v)
		v += 1.2

	}
	fsink = v
}

func BenchmarkSingleQuadratic(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(-0.02)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	v := 5.0
	for n := 0; n < b.N; n++ {
		p := float64(n % 128)
		v, _ = Cal.SingleQuadratic(p, 1, v)
		v += 1.2
	}
	fsink = v
}

func BenchmarkLinear(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(-0.02)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	v := 5.0
	for n := 0; n < b.N; n++ {
		p := float64(n % 128)
		v, _ = Cal.SingleLinear(p, 3, v)
		v += 1.2
	}
	fsink = v
}

func BenchmarkDoubleLinear(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(-0.02)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	v := 5.0
	for n := 0; n < b.N; n++ {
		p := float64(n % 128)
		v, _ = Cal.DoubleLinear(p, 3, v)
		v += 1.2
	}
	fsink = v
}

func BenchmarkBracket(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(-0.02)
	Cal.SetPower(100)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	v := 5.0
	for n := 0; n < b.N; n++ {
		v, _, _, _ = Cal.bracket(2.5, v)
		v += 2.6
	}
	fsink = v
}
func BenchmarkDeltaDist(b *testing.B) {
	Cal.SetWeight(100)
	Cal.SetGrade(0.0)
	Cal.SetWind(0)
	Cal.SetCrr(0.007)
	Cal.SetCdA(0.7)
	dist := 0.0
	for n := 0; n < b.N; n++ {
		dist, _, _, _ = Cal.DeltaVel(3, 6, 200)
	}
	fsink = dist
}
