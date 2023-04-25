package route

import (
	"encoding/json"
	"io"

	// "math"
	"bikeride/logerr"
	num "numconv"
)

// ftoi rounds float64 to nearest int
func ftoi(f float64) int {
	if f > 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}

// WriteCSV uses makeRouteCSV to build the route CSV as single []byte slice.
// makeRouteCSV uses package numconv for formatting numbers directly to the slice.
// This is much faster than using strconv FormatFloat and
// FormatInt functions. And encoding/csv writer.

func (o *Route) WriteCSV(p par, writer io.WriteCloser) error {
	const segmentBytes = 125
	routeCSV := make([]byte, 0, segmentBytes*o.segments)
	o.makeRouteCSV(&routeCSV, p)
	_, err := writer.Write(routeCSV)
	if err == nil {
		err = writer.Close()
	}
	return err
}

func (r *Results) WriteTXT(p par, writer io.WriteCloser) error {
	result := make([]byte, 0, 4000)
	r.makeResultTXT(&result, p)
	_, err := writer.Write(result)
	if err == nil {
		err = writer.Close()
	}
	return err
}

func (r *Results) WriteJSON(writer io.WriteCloser) error {
	json, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return err
	}
	_, err = writer.Write(json)
	if err == nil {
		err = writer.Close()
	}
	return err
}

func (o *Route) makeRouteCSV(b *[]byte, p par) {

	header := func(b *[]byte, sep byte, useCR bool) {
		*b = append(*b, "seg"...)
		*b = append(*b, sep)
		*b = append(*b, "lat"...)
		*b = append(*b, sep)
		*b = append(*b, "lon"...)
		*b = append(*b, sep)

		*b = append(*b, "eleGPX"...)
		*b = append(*b, sep)
		*b = append(*b, "ele"...)
		*b = append(*b, sep)
		*b = append(*b, "eleShift"...)
		*b = append(*b, sep)
		*b = append(*b, "wind"...)
		*b = append(*b, sep)
		*b = append(*b, "grade"...)
		*b = append(*b, sep)
		*b = append(*b, "dist"...)

		*b = append(*b, sep)
		*b = append(*b, "radius"...)
		*b = append(*b, sep)
		*b = append(*b, "course"...)
		*b = append(*b, sep)
		*b = append(*b, "pTarget"...)
		*b = append(*b, sep)
		*b = append(*b, "pRider"...)
		*b = append(*b, sep)
		*b = append(*b, "pBrake"...)
		*b = append(*b, sep)	
		*b = append(*b, "vMax"...)
		*b = append(*b, sep)
		*b = append(*b, "vTarget"...)
		*b = append(*b, sep)
		*b = append(*b, "vEntry"...)
		*b = append(*b, sep)
		*b = append(*b, "vExit"...)
		*b = append(*b, sep)
		*b = append(*b, "segTime"...)
		*b = append(*b, sep)
		*b = append(*b, "cumTime"...)
		*b = append(*b, sep)

		*b = append(*b, "cumDist"...)
		*b = append(*b, sep)
		*b = append(*b, "calcPath"...)
		*b = append(*b, sep)
		*b = append(*b, "calcSteps"...)

		if useCR {
			*b = append(*b, '\r')
		}
		*b = append(*b, '\n')
	}
	useCR := p.UseCR
	var sep byte = ','
	if p.CSVuseTab {
		sep = '\t'
	}
	header(b, sep, useCR)

	const d3 = 3
	const d6 = 6
	var cumsec, cumdist float64

	for i := 1; i <= o.segments+1; i++ {
		s := &o.route[i]
		radius := ftoi(s.radius)
		course := ftoi(s.course * rad2deg)
		pt := s.powerTarget
		if pt > 0 {
			pt *= p.PowerOut
		}
		powerTarget := ftoi(pt)
		powerRider := ftoi(s.powerRider * p.PowerOut)
		powerBraking := ftoi(s.powerBraking)

		num.Itoa(b, s.segnum, sep)
		num.Ftoa(b, s.lat, d6, sep)
		num.Ftoa(b, s.lon, d6, sep)
		num.Ftoa2(b, s.eleGPX, sep)
		num.Ftoa2(b, s.ele, sep)
		num.Ftoa2(b, s.ele-s.eleGPX, sep)
		num.Ftoa2(b, s.wind, sep)
		num.Ftoa2(b, s.grade*100, sep)
		num.Ftoa2(b, s.dist, sep)
		num.Itoa(b, radius, sep)
		num.Itoa(b, course, sep)
		num.Itoa(b, powerTarget, sep)
		num.Itoa(b, powerRider, sep)
		num.Itoa(b, powerBraking, sep)
		
		num.Ftoa2(b, s.vMax*ms2kmh, sep)
		num.Ftoa2(b, s.vTarget*ms2kmh, sep)
		num.Ftoa2(b, s.vEntry*ms2kmh, sep)
		num.Ftoa2(b, s.vExit*ms2kmh, sep)
		num.Ftoa2(b, s.time, sep)
		num.Ftoa(b, cumsec*s2h, d3, sep)
		num.Ftoa(b, cumdist*0.001, d3, sep)
		num.Itoa(b, s.calcPath, sep)
		num.Itoa(b, s.calcSteps, 0)
		cumdist += s.dist // distance to s.ele for profile grahps
		cumsec += s.time  // time to enter segment

		if useCR {
			*b = append(*b, '\r')
		}
		*b = append(*b, '\n')
	}
}

func (r *Results) makeResultTXT(b *[]byte, p par) {

	const (
		d1 = 1
		d2 = 2
		d3 = 3
		d4 = 4
		d5 = 5
	)
	var le = "\n"
	if p.UseCR {
		le = "\r\n"
	}
	wF := func(b *[]byte, s string, f float64, dec int, lend string) {
		*b = append(*b, s...)
		*b = append(*b, '\t')
		if f >= 0 {
			*b = append(*b, ' ')
		}
		num.Ftoa(b, f, dec, 0)
		*b = append(*b, lend...)
	}
	wI := func(b *[]byte, s string, f float64, lend string) {
		*b = append(*b, s...)
		*b = append(*b, '\t')
		if f >= 0 {
			*b = append(*b, ' ')
		}
		num.Itoa(b, ftoi(f), 0)
		*b = append(*b, lend...)
	}
	wS := func(b *[]byte, s string, x string, lend string) {
		*b = append(*b, s...)
		*b = append(*b, '\t')
		*b = append(*b, ' ')
		*b = append(*b, x...)
		*b = append(*b, lend...)
	}
	header := func(b *[]byte) {
		wS(b, "Route name       \t", p.RouteName, le)
		wS(b, "Ride parameters  \t", p.RideJSON, le)
		wS(b, "Configuration    \t", p.ConfigJSON, le)
		wS(b, "Route GPX file   \t", p.GPXfile, le)
		if r.TrkpErrors > 0 {
			wI(b, "Track point errors \t", float64(r.TrkpErrors), le)
		}
	}
	environment := func(b *[]byte) {
		*b = append(*b, le+"Environment"+le...)
		wI(b, "\tWind course (deg)       ", r.WindCourse, le)
		wF(b, "\tWind speed (m/s)        ", r.WindSpeed, d1, le)
		if r.RouteCourse >= 0 {
			wI(b, "\tRoute course (deg)      ", r.RouteCourse, le)
		}
		wI(b, "\tMean elevation (m)      ", r.EleMean, le)
		if p.AirDensity < 0 {
			wF(b, "\t    temperature (C)     ", r.Temperature, d1, le)
			wF(b, "\t    air density (kg/m^3)", r.Rho, d3, le)
		} else {
			wF(b, "\tAir density (kg/m^3)", r.Rho, d3, le)
		}
		if p.AirDensity < 0 {
			wI(b, "\tBase elevation (m)      ", r.BaseElevation, le)
			wF(b, "\t    temperature (C)     ", p.Temperature, d1, le)
			wF(b, "\t    air density (kg/m^3)", r.RhoBase, d3, le)
			wF(b, "\t    air pressure        ", r.AirPressure, d2, le)
		}
	}
	filtering := func(b *[]byte) {
		*b = append(*b, le+"Filtering"+le...)
		if r.Filtered == 0 {
			wI(b, "\tFiltered  (m)     ", 0, le)
			return
		}
		wI(b, "\tGPX elevation up (m)  ", r.EleUpGPX, le)
		wI(b, "\tGPX elevation down (m)", r.EleDownGPX, le)
		wI(b, "\tFilterable (m)        ", r.Filterable, le)
		wI(b, "\tFiltered (m)          ", r.Filtered, le)
		wF(b, "\t    % filterable      ", 100*r.Filtered/r.Filterable, d1, le)
		wF(b, "\t    % up GPX          ", 100*r.Filtered/r.EleUpGPX, d1, le)
		wF(b, "\t    % down GPX        ", 100*r.Filtered/r.EleDownGPX, d1, le)
		if r.Levelations > 0 {
			wI(b, "\tLeveled (m)           ", r.EleLevelled, le)
			wI(b, "\tLevelations           ", float64(r.Levelations), le)
		}
		if r.Ipolations > 0 {
			wI(b, "\tInterpolate rounds    ", float64(r.FilterRounds), le)
			wI(b, "\tInterpolations        ", float64(r.Ipolations), le)

		}
		wF(b, "\tRelative grade SD     ", r.RelGradeSD, d2, le)
	}
	elevation := func(b *[]byte) {
		*b = append(*b, le+"Elevation (m) "+le...)
		wI(b, "\tUp               ", r.EleUp, le)
		wI(b, "\tDown             ", r.EleDown, le)
		wI(b, "\tMax              ", r.EleMax, le)
		wI(b, "\tMin              ", r.EleMin, le)
		wI(b, "\tUp by momentum   ", r.EleUpKinetic, le)
	}
	roadsegments := func(b *[]byte) {
		wI(b, le+"Road segments         \t", float64(r.Segments), le)
		wI(b, "\tTrack points dropped  ", float64(r.TrkpRejected), le)
		wS(b, "\tLength (m) ", " ", le)
		wF(b, "\t  mean                ", r.DistMean, d1, le)
		wF(b, "\t  median              ", r.DistMedian, d1, "\t(approx.)"+le)
		wF(b, "\t  max                 ", r.DistMax, d1, le)
		wF(b, "\t  min                 ", r.DistMin, d1, le)
		wS(b, "\tGrade (%) ", " ", le)
		wF(b, "\t  max                  ", r.MaxGrade, d1, le)
		wF(b, "\t  min                  ", r.MinGrade, d1, le)

	}
	distance := func(b *[]byte) {
		wF(b, le+"Distance (km)    \t", r.DistTotal, d1, le)
		wF(b, "\tDirect           ", r.DistDirect, d1, le)
		wF(b, "\tUphill   >  4%   ", r.DistUphill, d1, le)
		wF(b, "\tDownhill < -4%   ", r.DistDownhill, d1, le)
		wF(b, "\tFlat   -1% - 1%  ", r.DistFlat, d1, le)
		wF(b, "\tBraking          ", r.DistBrake, d1, le)
		wF(b, "\tFreewheeling     ", r.DistFreewheel, d1, le)
	}
	speed := func(b *[]byte) {
		*b = append(*b, le+"Speed (km/h)"+le...)
		wF(b, "\tMean                 ", r.VelAvg, d2, le)
		wF(b, "\tMax                  ", r.VelMax, d2, le)
		wF(b, "\tMin                  ", r.VelMin, d2, le)
		wF(b, "\tDownhill < -4% mean  ", r.VelDownhill, d2, le)
		wI(b, "\tDown vertical (m/h)  ", r.VelDownVert, le)
	}
	drivingtime := func(b *[]byte) {
		wF(b, le+"Driving time (h)       \t", r.Time, d2, le)
		wF(b, "\tFrom target speeds     ", r.TimeTargetSpeeds, d2, le)
		wF(b, "\tPedal powered          ", r.TimeRider, d2, le)
		wF(b, "\tBraking                ", r.TimeBraking, d2, le)
		wF(b, "\tFreewheeling           ", r.TimeFreewheel, d2, le)
		if r.TimeUHBreaks > 0 {
			wF(b, "\tWith uphill breaks     ", r.Time+r.TimeUHBreaks, d2, le)
			wF(b, "\tUphill break time      ", r.TimeUHBreaks, d2, le)
		}
		if p.UphillBreak.PowerLimit > 0 {
			*b = append(*b, "\tOver "...)
			num.Itoa(b, ftoi(p.UphillBreak.PowerLimit), '%')
			wF(b, " uphill power  ", r.TimeFullPower, d2, le)
		}
		wF(b, "\tOver flat ground power ", r.TimeOverFlatPower, d2, le)

	}

	// Joules below are converted to Wh before
	energyrider := func(b *[]byte) {
		wI(b, le+"Energy rider (Wh)          \t", r.JriderTotal, le)
		wI(b, "\tFrom target powers (Wh)  ", r.JfromTargetPower, le)
		wI(b, "\tFood (kcal)              ", r.FoodRider, le)
		wI(b, "\tBananas (pcs)            ", r.BananaRider, le)
		wI(b, "\tLard (g)                 ", r.FatRider, le)
		wI(b, "\tAverage power (W)        ", r.PowerRiderAvg, le)
		wF(b, "\tEnergy/distance (Wh/km)  ", r.JriderTotal/r.DistTotal, d2, le)
	}

	riderenergyusage := func(b *[]byte) {
		c := 100.0 / r.JriderTotal
		wS(b, le+"Rider's energy usage \t\t  Wh", "%", le)
		wI(b, "\tGravity up          ", r.JriderGravUp, "  \t")
		num.Ftoa(b, c*r.JriderGravUp, d1, 0)
		*b = append(*b, le...)
		wI(b, "\tAir resistance      ", r.JriderDrag, "  \t")
		num.Ftoa(b, c*r.JriderDrag, d1, 0)
		*b = append(*b, le...)
		wI(b, "\tRolling resistance  ", r.JriderRoll, "  \t")
		num.Ftoa(b, c*r.JriderRoll, d1, 0)
		*b = append(*b, le...)
		wI(b, "\tDrivetrain loss     ", r.JlossDT, "  \t")
		num.Ftoa(b, c*r.JlossDT, d1, 0)
		*b = append(*b, le...)
		wF(b, "\tEnergy net sum      ", r.EnergySumRider, d1, le)
		wI(b, "\tAcceleration        ", r.JriderAcce, "  \t(included above)"+le)
	}

	totalenergybalance := func(b *[]byte) {
		*b = append(*b, le+"Total energy balance (Wh)"+le...)
		wI(b, "\tRider               ", r.JriderTotal, le)
		wI(b, "\tDrivetrain loss     ", -r.JlossDT, le)
		wF(b, "\tKinetic resistance  ", r.JkineticAcce, d1, le)
		wF(b, "\tKinetic push        ", r.JkineticDece, d1, le)
		wI(b, "\tGravity up          ", r.JgravUp, le)
		wI(b, "\tGravity down        ", r.JgravDown, le)
		wI(b, "\tAir resistance      ", r.JdragResistance, le)
		wI(b, "\t    pedaling     ", r.JdragRider, le)
		wI(b, "\t    freewheeling ", r.JdragFreewheel, le)
		wI(b, "\t    braking      ", r.JdragBrake, le)
		if r.JdragPush > 0 {
			wI(b, "\tWind push           ", r.JdragPush, le)
		}
		wI(b, "\tRolling resistance  ", r.Jroll, le)
		wI(b, "\tBraking             ", r.Jbraking, le)
		wF(b, "\tEnergy net sum      ", r.EnergySumTotal, d1, le) 
	}

	rider := func(b *[]byte) {
		P := p.Powermodel
		*b = append(*b, le+"Rider/riding"+le...)
		wI(b, "\tTotal weight (kg)          ", p.Weight.Total, le)
		wF(b, "\tFlat ground speed (km/h)   ", P.FlatSpeed, d2, le)
		wI(b, "\tFlat ground power (W)      ", P.FlatPower, le)
		wF(b, "\tAir drag coefficient CdA   ", p.CdA, d2, le)
		wI(b, "\tUphill power (W)           ", P.UphillPower, le)
		wI(b, "\tVertical speed up (m/h)    ", P.VerticalUpSpeed, le)
		wF(b, "\tUphill power speed (km/h)  ", P.UphillPowerSpeed, d1, le)
		wF(b, "\tUphill power grade (%)     ", P.UphillPowerGrade, d1, le)
		if r.MaxGradeUp > 0 {
			wF(b, "\tUphill power max grade (%)", r.MaxGradeUp, d1, "\t(")
			num.Ftoa(b, p.Ride.MinSpeed, d1, 0)
			*b = append(*b, " km/h)"+le...)
		}
		wF(b, "\tMax pedaled speed (km/h)  ", P.MaxPedaledSpeed, d1, le)
		wF(b, "\tMin pedaled grade (%)     ", p.MinPedaledGrade, d2, "\t(")
		num.Ftoa(b, P.MaxPedaledSpeed, d1, 0)
		*b = append(*b, " km/h)"+le...)
		if p.ReportTech {
			wF(b, "\tDownhill power grade (%)  ", P.DownhillPowerGrade, d2, le)
			wF(b, "\tDownhill power (%)        ", P.DownhillPower, d1, le)
			wF(b, "\tDownhill power speed (km/h)", r.DownhillPowerSpeed, d1, le)
		}
		if !p.Ride.LimitDownSpeeds {
			return
		}
		if p.Ride.BrakingDist <= 0 {
			wI(b, "\tVertical speed down (m/h)", p.Ride.VerticalDownSpeed, le)
		}
		if p.Ride.BrakingDist > 0 {
			wI(b, "\tBraking distance (m)    ", p.Ride.BrakingDist, le)
		}
		wF(b, "\tDownhill (-6%) max speed", r.DownhillMaxSpeed, d1, le)
	}
	technical := func(b *[]byte) {
		const (
			newtonRaphson   = 1
			singleQuadratic = 2
			doubleQuadratic = 3
			doubleLinear  = 4
		)
		*b = append(*b, le+"Calculation"+le...)
		wF(b, "\tSegment energy mean error (J)     ", r.SegEnergyMean, d2, le)
		wF(b, "\tSegment energy mean abs(error) (J)", r.SegEnergyMeanAbs, d2, le)
		// wF(b, "\tJoule waste bin / segments (J)    ", r.Jsink/float64(r.Segments), d2, le)
		wF(b, "\tJoule waste bin (J)               ", r.Jsink, d2, le)
		wF(b, "\tMean latitude (deg)                ", r.LatMean, d2, le)
		wF(b, "\tLocal gravity                      ", r.Gravity, d3, le)
		*b = append(*b, "\n\tStepping "...)
		switch p.DiffCalc {
		case 1:
			*b = append(*b, "velocity de/increments"...)
		case 2:
			*b = append(*b, "distance increments"...)
		case 3:
			*b = append(*b, "time increments"...)
		}
		*b = append(*b, le...)
		wF(b, "\t  steps per road segment (mean)", r.CalcStepsAvg, d2, le)
		wF(b, "\t  single step segments (%)     ", r.SingleStepPros, d1, le)
		// wI(b, "\t  CalcSegs                     ", float64(r.CalcSegs), le)

		*b = append(*b, le+"\tVelocity solver: "...)
		if p.UseVelTable {
			*b = append(*b, "lookup table & "...)
		}
		switch p.VelSolver {
		case newtonRaphson:
			*b = append(*b, "Newton-Raphson"...)
		case singleQuadratic:
			*b = append(*b, "Quadratic interpolation"...)
		case doubleQuadratic:
			*b = append(*b, "Two quadratic interpolations"...)
		case doubleLinear:
			*b = append(*b, "Two linear interpolations"...)
		}
		*b = append(*b, le...)
		if p.VelSolver > 1 {
			wF(b, "\t  function evaluations (mean)", r.SolverRoundsAvg, d2, le)
		}
		if p.VelSolver == 1 {
			wF(b, "\t  iterations (mean)           ", r.SolverRoundsAvg, d2, le)
			wI(b, "\t  iterations (max)            ", float64(r.MaxIter), le)
		}
		wI(b, "\t  calls solver                ", float64(r.SolverCalls), le)
		wI(b, "\n\tCalls freewheeling speed      ", float64(r.FreewheelCalls), le)
		wI(b, "\tCalls power from speed        ", float64(r.PowerFromVelCalls), le)
		if !p.VelErrors {
			return
		}
		wS(b, le+"\tVelocity error (m/s)", " ", le)
		wF(b, "\t  mean abs(error)   ", r.VelErrorAbsMean, d5, le)
		wF(b, "\t  mean error, bias  ", r.VelErrorMean, d5, le)
		wF(b, "\t  max abs(error)    ", r.VelErrorMax, d5, le)
		wF(b, "\t  errors > 0 (%)    ", r.VelErrorPos*100, d1, le)
	}
	header(b)
	environment(b)
	filtering(b)
	elevation(b)
	roadsegments(b)
	distance(b)
	speed(b)
	drivingtime(b)
	energyrider(b)
	riderenergyusage(b)
	rider(b)
	if p.ReportTech {
		totalenergybalance(b)
		technical(b)

	}
}

func (r *Results) Display(p par, l *logerr.Logerr) {
	l.Printf("%s %s\n", "Route                  ", p.GPXfile)
	l.Printf("%s %d\n", "Road segments          ", r.Segments)
	if r.TrkpRejected > 0 {
		l.Printf("%s %d\n", "Track points dropped   ", r.TrkpRejected)
	}
	l.Printf("%s %5.3f\n", "Distance (km)          ", r.DistTotal)
	l.Printf("%s\n", "Elevation (m)")
	l.Printf("%s %4.0f\n", "    Up               ", r.EleUp)
	l.Printf("%s %4.0f\n", "    Down             ", r.EleDown)
	if r.Filtered > 0 || p.Filter.WeightedAvgWeight > 0 {
		l.Printf("%s %4.0f\n", "    Up GPX           ", r.EleUpGPX)
		l.Printf("%s %4.0f\n", "    Down GPX         ", r.EleDownGPX)
		l.Printf("%s %4.0f\n", "    Max              ", r.EleMax)
		l.Printf("%s %4.0f\n", "    Min              ", r.EleMin)
		l.Printf("%s %4.0f \n", "    Filtered         ", r.Filtered)
		l.Printf("%s  %4.1f%s\n", "    From filterable ", r.FilteredPros, " %")
		if r.EleLevelled > 0 {
			l.Printf("%s %4.0f\n", "    Leveled          ", r.EleLevelled)
		}
		if r.Ipolations > 0 {
			l.Printf("%s %6d\n", "Interpolations     ", r.Ipolations)
			l.Printf("%s %6d\n", "    rounds         ", r.FilterRounds)
		}
		if r.Levelations > 0 {
			l.Printf("%s %6d\n", "Levelations        ", r.Levelations)
		}
	}
	l.Printf("%s %5.1f\n", "Min grade %         ", r.MinGrade)
	l.Printf("%s %4.1f\n", "Max grade %          ", r.MaxGrade)
	l.Printf("%s %5.2f\n", "Relative grade SD   ", r.RelGradeSD)
	l.Printf("\n%s %6.1f\n", "Energy rider (Wh)         ", r.JriderTotal)
	l.Printf("%s %6.3f\n", "Ride time (h)             ", r.Time)
	if r.TimeUHBreaks > 0 {
		l.Printf("%s %6.3f\n", "Time with breaks (h)     ", r.Time+r.TimeUHBreaks)
	}
	l.Printf("%s %6.2f\n", "Energy/distance (Wh/km)  ", r.JriderTotal/r.DistTotal)
}
