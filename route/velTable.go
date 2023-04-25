package route

import "motion"

// velFromTable interpolates velocity from precalculated grade x wind table.
const (
	gradeMax      = 13
	gradeMin      = -5
	minTableGrade =  gradeMin / 100.0
	windMax       = 5
	tableGlim     = gradeMax - gradeMin
	tableWlim     = 2 * windMax
)

var velTable [tableGlim + 1][tableWlim + 1]float64

func (s *segment) velFromTable(c *motion.BikeCalc) (float64, bool) {

	grade := s.grade*100 - gradeMin
	if grade < 0 {
		return 15, false
	}
	ok := true
	wind := s.wind + windMax
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
		return velTable[g0][w0], false
	}
	v00 := velTable[g0][w0]
	v01 := velTable[g0][w1]
	v10 := velTable[g1][w0]
	v11 := velTable[g1][w1]

	grade -= float64(g0)
	wind -= float64(w0)

	v0 := v00 + wind*(v01-v00)
	v1 := v10 + wind*(v11-v10)
	v := v0 + grade*(v1-v0)

	s.vTarget = v
	s.powerTarget = c.PowerFromVel(s.vTarget)
	return v, true
}

func setupTargetVelTable(c *motion.BikeCalc, power ratioGenerator, p par) error {

	P := p.Powermodel
	prevel := 6.0
	
	for g := gradeMin; g <= gradeMax; g++ {
		grade := float64(g) * 0.01
		minSpeed := max(0.1, p.Ride.MinSpeed)
		c.SetGrade(grade)

		for w := -windMax; w <= windMax; w++ {
			var vel float64
			var ok bool
			wind := float64(w)
			c.SetWind(wind)

			pow := P.FlatPower * power.Ratio(grade, wind)

			if pow < powerTOL {
				vel = c.VelFreewheeling()
				if vel >= P.MaxPedaledSpeed {
					velTable[g-gradeMin][w+windMax] = vel
					continue
				}
			}
			if pow >= powerTOL {
				vel, ok = c.VelFromPower(pow, prevel)
				prevel = vel
				if !ok {
					return errf(" setupTargetVelTable: Target velocity is not solvable")
				}
				if vel <= P.MaxPedaledSpeed && vel >= minSpeed {
					velTable[g-gradeMin][w+windMax] = vel
					continue
				}
			}
			if vel < minSpeed {
				velTable[g-gradeMin][w+windMax] = minSpeed
				continue
			}
			vel = P.MaxPedaledSpeed
			pow = c.PowerFromVel(vel)
			if pow < powerTOL {
				vel = c.VelFreewheeling()
			}
			velTable[g-gradeMin][w+windMax] = vel
		}
	}
	return nil
}
