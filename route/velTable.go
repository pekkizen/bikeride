package route

import "github.com/pekkizen/motion"

const (
	gradeMax  = 15
	gradeMin  = -5
	windMax   = 5
	tableGlim = gradeMax - gradeMin
	tableWlim = 2 * windMax
)

var velTable [tableGlim + 1][tableWlim + 1]float64

// velFromTable interpolates velocity from precalculated grade x wind table.
func velFromTable(c *motion.BikeCalc, grade, wind float64) (power, vel float64, ok bool) {
	ok = true
	wind += windMax
	grade *= 100
	grade -= gradeMin

	if grade < 0 {
		grade = 0
		ok = false
	}
	if wind < 0 {
		wind = 0
		ok = false
	}
	g0 := int(grade)
	w0 := int(wind)
	g1 := g0 + 1
	w1 := w0 + 1

	if g1 > tableGlim {
		g0 = tableGlim
		ok = false
	}
	if w1 > tableWlim {
		w0 = tableWlim
		ok = false
	}
	if !ok {
		return 0, velTable[g0][w0], false
		// good velguess for the next function to start with.
	}
	grade -= float64(g0)
	wind -= float64(w0)

	v00 := velTable[g0][w0]
	v01 := velTable[g0][w1]
	v10 := velTable[g1][w0]
	v11 := velTable[g1][w1]

	v0 := v00 + wind*(v01-v00)
	v1 := v10 + wind*(v11-v10)
	const bias = 0.005
	vel = v0 + grade*(v1-v0) - bias
	power = c.PowerFromVel(vel)
	return
}

func fillTargetVelTable(c *motion.BikeCalc, power ratioGenerator, basePower float64) (ok bool) {
	velguess := 10.0
	for g := gradeMin; g <= gradeMax; g++ {
		grade := float64(g) * 0.01
		c.SetGradeExact(grade)
		velguess += 3

		for w := -windMax; w <= windMax; w++ {
			wind := float64(w)
			c.SetWind(wind)

			pow := basePower * power.Ratio(grade, wind)
			vel, iter := c.NewtonRaphson(pow, 0.1, velguess)
			if iter == 0 {
				return false
			}
			velguess = vel * 0.96
			velTable[g-gradeMin][w+windMax] = vel
		}
	}
	return true
}
