package route

import (
	"io"
	"os"

	"bikeride/logerr"
	"bikeride/param"
	num "numconv"
)

// ftoi rounds float64 f to nearest int
func ftoi(f float64) int {
	if f > 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}

// WriteCSV uses makeRouteCSV to build the route CSV as single []byte slice.
// makeRouteCSV uses package numconv for formatting numbers directly to the slice.
// Slice is written out by single io.Writer.Write.
// This is over 10 x faster than using strconv FormatFloat and
// FormatInt functions and encoding/csv writer.

// WriteCSV (*Route) writes route data in CSV format to io.Writer
func (o *Route) WriteCSV(p par, writer io.WriteCloser) error {
	if !p.RouteCSV {
		return nil
	}
	const roadSegmentBytes = 120
	routeCSV := make([]byte, 0, roadSegmentBytes*o.segments)
	o.makeRouteCSV(&routeCSV, p)

	_, err := writer.Write(routeCSV)
	if err == nil {
		err = writer.Close()
	}
	return err
}

// WriteTXT (*Results) writes results as tab separated text to io.Writer
func (r *Results) WriteTXT(p par, writer io.WriteCloser) error {
	if !p.ResultTXT {
		return nil
	}
	resultTXT := make([]byte, 0, 4000)
	r.makeResultTXT(&resultTXT, p)

	_, err := writer.Write(resultTXT)
	if err == nil {
		err = writer.Close()
	}
	return err
}

func (o *Route) Writer(p *param.Parameters) (io.WriteCloser, error) {
	f, err := os.OpenFile(p.ResultDir+p.RouteName+".csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
func (r *Results) Writer(p *param.Parameters) (io.WriteCloser, error) {
	f, err := os.OpenFile(p.ResultDir+p.RouteName+"_results"+".txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (o *Route) makeRouteCSV(b *[]byte, p par) {

	const d3 = 3
	const d6 = 6
	var cumsec, cumdist float64

	header := func(b *[]byte, sep byte, useCR bool) {
		*b = append(*b, "lat"...)
		*b = append(*b, sep)
		*b = append(*b, "lon"...)
		*b = append(*b, sep)
		*b = append(*b, "ele"...)
		*b = append(*b, sep)
		*b = append(*b, "eleGPX"...)
		*b = append(*b, sep)
		*b = append(*b, "wind"...)
		*b = append(*b, sep)
		*b = append(*b, "grade"...)
		*b = append(*b, sep)
		*b = append(*b, "dist"...)
		*b = append(*b, sep)
		*b = append(*b, "cumDist"...)
		*b = append(*b, sep)
		*b = append(*b, "radius"...)
		*b = append(*b, sep)
		*b = append(*b, "course"...)
		*b = append(*b, sep)
		*b = append(*b, "powerTarget"...)
		*b = append(*b, sep)
		*b = append(*b, "powerRider"...)
		*b = append(*b, sep)
		*b = append(*b, "powerBrake"...)
		*b = append(*b, sep)
		*b = append(*b, "velTarget"...)
		*b = append(*b, sep)
		*b = append(*b, "velMax"...)
		*b = append(*b, sep)
		*b = append(*b, "velEntry"...)
		*b = append(*b, sep)
		*b = append(*b, "velExit"...)
		*b = append(*b, sep)
		*b = append(*b, "segTime"...)
		*b = append(*b, sep)
		*b = append(*b, "cumTime"...)
		*b = append(*b, sep)
		*b = append(*b, "calcPath"...)
		*b = append(*b, sep)
		*b = append(*b, "segment"...)
		if useCR {
			*b = append(*b, '\r')
		}
		*b = append(*b, '\n')
	}
	useCR := p.UseCRLF
	var sep byte = '\t'
	if p.CSVuseComma {
		sep = ','
	}
	header(b, sep, useCR)

	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		cumsec += s.time
		cumdist += s.dist
		radius := ftoi(s.radius)
		course := ftoi(s.course * rad2deg)
		powerTarget := ftoi(s.powerTarget * p.PowerOut)
		powerRider := ftoi(s.powerRider * p.PowerOut)
		powerBraking := ftoi(s.powerBraking)

		num.Ftoa(b, s.lat, d6, sep)
		num.Ftoa(b, s.lon, d6, sep)
		num.Ftoa2(b, s.ele, sep)
		num.Ftoa2(b, s.eleGPX, sep)
		num.Ftoa2(b, s.wind, sep)
		num.Ftoa2(b, s.grade*100, sep)
		num.Ftoa2(b, s.dist, sep)
		num.Ftoa(b, cumdist*0.001, d3, sep)
		num.Itoa(b, radius, sep)
		num.Itoa(b, course, sep)
		num.Itoa(b, powerTarget, sep)
		num.Itoa(b, powerRider, sep)
		num.Itoa(b, powerBraking, sep)
		num.Ftoa2(b, s.vTarget*ms2kmh, sep)
		num.Ftoa2(b, s.vMax*ms2kmh, sep)
		num.Ftoa2(b, s.vEntry*ms2kmh, sep)
		num.Ftoa2(b, s.vExit*ms2kmh, sep)
		num.Ftoa2(b, s.time, sep)
		num.Ftoa(b, cumsec*s2h, d3, sep)
		num.Itoa(b, s.calcPath, sep)
		num.Itoa(b, s.segnum, 0)
		if useCR {
			*b = append(*b, '\r')
		}
		*b = append(*b, '\n')
	}
}

func (r *Results) makeResultTXT(b *[]byte, p par) {

	var (
		d1 = 1
		d2 = 2
		d3 = 3
		// d4 = 4
		d5 = 5
	)
	var le = "\n"
	if p.UseCRLF {
		le = "\r\n"
	}
	wF := func(b *[]byte, s string, f float64, dec int, lend string) {
		*b = append(*b, s...)
		*b = append(*b, '\t')
		if f > 0 {
			*b = append(*b, ' ')
		}
		num.Ftoa(b, f, dec, 0)
		*b = append(*b, lend...)
	}
	wI := func(b *[]byte, s string, f float64, lend string) {
		*b = append(*b, s...)
		*b = append(*b, '\t')
		if f > 0 {
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
		wS(b, "Parameter file   \t", p.RideJSON, le)
		wS(b, "Route GPX file   \t", p.GPXfile, le)
		if r.TrkpErrors > 0 {
			wI(b, "Track point errors\t", float64(r.TrkpErrors), le)
		}
	}
	environment := func(b *[]byte) {
		*b = append(*b, le+"Environment"+le...)
		wI(b, "\tWind course (deg)   ", r.WindCourse, le)
		wF(b, "\tWind speed (m/s)    ", r.WindSpeed, d1, le)
		if r.RouteCourse >= 0 {
			wI(b, "\tRoute course (deg)  ", r.RouteCourse, le)
		}
		wI(b, "\tMean elevation (m)  ", r.EleMean, le)
		wF(b, "\tTemperature (C)     ", r.Temperature, d1, "\t\tat mean elevation"+le)
		wI(b, "\tBase elevation (m)  ", r.BaseElevation, le)
		wF(b, "\tTemperature (C)     ", p.Temperature, d1, "\t\tat base elevation"+le)
		wF(b, "\tAir pressure        ", r.AirPressure, d2, "\tat base elevation"+le)
		wF(b, "\tAir density (kg/m^3)", r.Rho, d3, "\t\tat mean elevation"+le)

	}
	filtering := func(b *[]byte) {
		*b = append(*b, le+"Filtering"+le...)
		if r.Filtered == 0 {
			wI(b, "\tFiltered  (m)  ", 0, le)
			return
		}
		wI(b, "\tGPX elevation (m)  ", r.EleUpGPX, le)
		eleIpo := 0.0
		if r.Ipolations > 0 {
			eleIpo = r.Filtered - r.EleLevelled
		}
		wI(b, "\tInterpolated (m)   ", eleIpo, le)
		if r.EleLevelled > 0 {
			wI(b, "\tLevelled (m)       ", r.EleLevelled, le)
		}
		wF(b, "\tFiltered %         ", r.FilteredPros, d1, le)
		wI(b, "\tFiltering rounds   ", float64(r.FilterRounds), le)
		wI(b, "\tInterpolations     ", float64(r.Ipolations), le)
		wI(b, "\tLevelations        ", float64(r.Levelations), le)
	}
	elevation := func(b *[]byte) {
		*b = append(*b, le+"Elevation (m) "+le...)
		wI(b, "\tUp               ", r.EleUp, le)
		wI(b, "\tDown             ", r.EleDown, le)
		wI(b, "\tUp by momentum   ", r.EleUpKinetic, le)
	}
	roadsegments := func(b *[]byte) {
		wI(b, le+"Road segments         \t", float64(r.Segments), le)
		wI(b, "\tTrack points dropped  ", float64(r.TrkpRejected), le)
		wS(b, "\tLength (m) ", " ", le)
		wF(b, "\t  Mean                ", r.DistMean, d1, le)
		wF(b, "\t  Median              ", r.DistMedian, d1, "\t(approx.)"+le)
		wF(b, "\t  Max                 ", r.DistMax, d1, le)
		wF(b, "\t  Min                 ", r.DistMin, d1, le)
		wS(b, "\tGrade (%) ", " ", le)
		wF(b, "\t  Max                  ", r.MaxGrade, d1, le)
		wF(b, "\t  Min                  ", r.MinGrade, d1, le)
		wF(b, "\t  Grade sign change %  ", r.GradeSignChange, d1, le)
	}
	distance := func(b *[]byte) {
		wF(b, le+"Distance (km)    \t", r.DistTotal, d1, le)
		wF(b, "\tDirect           ", r.DistDirect, d1, le)
		wF(b, "\tUphill   > 4%    ", r.DistUphill, d1, le)
		wF(b, "\tDownhill <-4%    ", r.DistDownhill, d1, le)
		wF(b, "\tFlat   -1%-1%    ", r.DistFlat, d1, le)
		wF(b, "\tBraking light    ", r.DistBrake, d1, le)
		wF(b, "\tBraking heavy    ", r.DistHeavyBrake, d1, le)
		wF(b, "\tFreewheeling     ", r.DistFreewheel, d1, le)
	}
	speed := func(b *[]byte) {
		*b = append(*b, le+"Speed (km/h)"+le...)
		wF(b, "\tMean                 ", r.VelAvg, d2, le)
		wF(b, "\tMax                  ", r.VelMax, d2, le)
		wF(b, "\tMin                  ", r.VelMin, d2, le)
		wF(b, "\tDownhill <-4% mean   ", r.VelDownhill, d2, le)
		wI(b, "\tDown vertical (m/h)  ", r.VelDownVert, le)
	}
	drivingtime := func(b *[]byte) {
		wF(b, le+"Driving time (h)       \t", r.Time, d2, le)
		wF(b, "\tFrom target speeds     ", r.TimeTargetSpeeds, d2, le)
		if r.TimeUHBreaks > 0 {
			wF(b, "\tWith uphill breaks     ", r.Time+r.TimeUHBreaks, d2, le)
			wF(b, "\tUphill break time      ", r.TimeUHBreaks, d2, le)
		}
		*b = append(*b, "\tOver "...)
		num.Itoa(b, ftoi(p.UphillBreak.PowerLimit), '%')
		wF(b, " uphill power  ", r.TimeFullPower, d2, le)
		wF(b, "\tOver flat power        ", r.TimeOverFlatPower, d2, le)
		wF(b, "\tPedal powered          ", r.TimeRider, d2, le)
		wF(b, "\tBraking                ", r.TimeBraking, d2, le)
	}
	energyrider := func(b *[]byte) {
		wI(b, le+"Energy rider (Wh)    \t", r.JriderTotal, le)
		wI(b, "\tFrom target powers (Wh)", r.JfromTargetPower, le)
		wI(b, "\tFood (kcal)          ", r.FoodRider, le)
		wI(b, "\tBananas (pcs)        ", r.BananaRider, le)
		wI(b, "\tFat (g)              ", r.FatRider, le)
		wI(b, "\tAverage power (W)    ", r.PowerRiderAvg, le)
	}
	riderenergyusage := func(b *[]byte) {
		c := 100.0 / r.JriderTotal
		wS(b, le+"Rider's energy usage \t\t  Wh", "%", le)
		wI(b, "\tGravity up          ", r.JriderGravUp, " \t")
		num.Ftoa(b, c*r.JriderGravUp, d1, 0)
		*b = append(*b, le...)
		wI(b, "\tAir resistance      ", r.JriderDrag, " \t")
		num.Ftoa(b, c*r.JriderDrag, d1, 0)
		*b = append(*b, le...)
		wI(b, "\tRolling resistance  ", r.JriderRoll, " \t")
		num.Ftoa(b, c*r.JriderRoll, d1, 0)
		*b = append(*b, le...)
		wI(b, "\tDrivetrain loss     ", r.JlossDT, " \t")
		num.Ftoa(b, c*r.JlossDT, d1, 0)
		*b = append(*b, le...)
		wF(b, "\tEnergy net sum      ", r.EnergySumRider, d1, le)
		wI(b, "\tAcceleration        ", r.JriderAcce, "\t(included above)"+le)
	}
	totalenergybalance := func(b *[]byte) {
		*b = append(*b, le+"Total energy balance (Wh)"+le...)
		wI(b, "\tRider               ", r.JriderTotal, le)
		wI(b, "\tDrivetrain loss     ", -r.JlossDT, le)
		wF(b, "\tKinetic resistance  ", r.JkineticAcce, d1, le)
		wF(b, "\tKinetic push        ", r.JkineticDece, d1, le)
		wI(b, "\tGravity up          ", r.JgravUp, le)
		wI(b, "\tGravity down        ", r.JgravDown, le)
		wI(b, "\tAir resistance      ", r.JdragRes, le)
		wI(b, "\tAir braking         ", r.JdragBrake, le)
		wI(b, "\tWind push           ", r.JdragPush, le)
		wI(b, "\tRolling resistance  ", r.Jroll, le)
		wI(b, "\tBraking             ", r.Jbraking, le)
		wF(b, "\tEnergy net sum      ", r.EnergySumTotal, d1, le)
	}
	rider := func(b *[]byte) {
		P := p.Powermodel
		*b = append(*b, le+"Rider/riding"+le...)
		wI(b, "\tTotal weight (kg)          ", p.Weight.Total, le)
		wF(b, "\tFlat speed (km/h)          ", P.FlatSpeed, d2, le)
		wI(b, "\tFlat power (W)             ", P.FlatPower, le)
		wF(b, "\tAir drag coefficient CdA   ", p.CdA, d2, le)
		wI(b, "\tUphill power (W)           ", P.UphillPower, le)
		wI(b, "\tVertical speed up (m/h)    ", P.VerticalUpSpeed, le)
		wF(b, "\tUphill power speed (km/h)  ", P.UphillPowerSpeed, d1, le)
		wF(b, "\tUphill power grade (%)     ", P.UphillPowerGrade, d1, le)
		wF(b, "\tUphill power max grade (%)", r.MaxGradeUp, d1, "\t(")
		num.Ftoa(b, p.MinSpeed, d1, 0)
		*b = append(*b, "km/h)"+le...)
		wF(b, "\tMax pedalled speed (km/h) ", P.MaxPedalledSpeed, d1, le)
		wF(b, "\tMin pedalled grade (%)    ", p.MinPedalledGrade, d2, le)
		wF(b, "\tDownhill power grade (%)  ", P.DownhillPowerGrade, d2, le)
		wF(b, "\tDownhill power (%)        ", P.DownhillPower, d1, le)
		wF(b, "\tDownhill power speed (km/h)", r.DownhillPowerSpeed, d1, le)
		if !p.LimitDownSpeeds {
			return
		}
		if p.BrakingDist == 0 {
			wI(b, "\tVertical speed down (m/h)", p.VerticalDownSpeed, le)
		}
		if p.BrakingDist > 0 {
			wI(b, "\tBraking distance (m)    ", p.BrakingDist, le)

		}
		wF(b, "\tDownhill (-6%) max speed", r.DownhillMaxSpeed, d1, le)
	}
	technical := func(b *[]byte) {
		const (
			newtonRaphson   = 1
			singleQuadratic = 2
			singleLinear    = 3
			doubleLinear    = 4
			doubleQuadratic = 5
			bisect          = 6
		)

		*b = append(*b, le+"Calculation"+le...)
		wF(b, "\tSegment energy mean error (J)     ", r.SegEnergyMean, d2, le)
		wF(b, "\tSegment energy mean abs(error) (J)", r.SegEnergyMeanAbs, d2, le)
		wF(b, "\tMean latitude (deg)                ", r.LatMean, d2, le)
		wF(b, "\tLocal gravity                      ", r.Gravity, d3, le)
		*b = append(*b, "\tStepping "...)
		switch p.DiffCalc {
		case 1:
			*b = append(*b, "velocity de/increments"...)
		case 2:
			*b = append(*b, "distance increments"...)
		case 3:
			*b = append(*b, "time increments"...)
		}
		*b = append(*b, le...)
		wF(b, "\tAcce/decelaration steps (mean)    ", r.CalcStepsAvg, d2, le)
		wF(b, "\tSingle step segments (%)          ", r.SingleStepPros, d1, le)

		*b = append(*b, le+"\tVelocity solver: "...)
		if p.UseVelTable {
			*b = append(*b, "lookup table & "...)
		}
		switch p.VelSolver {
		case newtonRaphson:
			*b = append(*b, "Newton-Raphson"...)
		case singleQuadratic:
			*b = append(*b, "Bracket/quadratic interpolation"...)
		case singleLinear:
			*b = append(*b, "Bracket/single linear interpolation"...)
		case doubleLinear:
			*b = append(*b, "Bracket/two linear interpolations"...)
		case doubleQuadratic:
			*b = append(*b, "Bracket/two quadratic interpolation"...)
		case bisect:
			*b = append(*b, "Bisect"...)
		}
		*b = append(*b, le...)
		if p.VelSolver > 1 {
			wF(b, "\t  function evaluations (mean) ", r.SolverRoundsAvg, d2, le)
		}
		if p.VelSolver == 1 {
			wF(b, "\t  Iterations (mean)           ", r.SolverRoundsAvg, d2, le)
			wI(b, "\t  Iterations (max)            ", float64(r.MaxIter), le)
		}
		wI(b, "\t  calls solver                ", float64(r.SolverCalls), le)
		wI(b, "\tCalls freewheeling speed      ", float64(r.FreewheelCalls), le)
		wI(b, "\tCalls power from speed        ", float64(r.PowerFromVelCalls), le)
		if !p.CalcVelError {
			return
		}
		wS(b, le+"\tVelocity error", " ", le)
		wF(b, "\t  mean abs(error) (m/s)  ", r.VelErrorAbsMean, d5, le)
		wF(b, "\t  mean error, bias (m/s) ", r.VelErrorMean, d5, le)
		wF(b, "\t  max abs(error) (m/s)   ", r.VelErrorMax, d5, le)
		wF(b, "\t  errors > 0 (%)         ", r.VelErrorPos*100, d1, le)
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

// Display ------
func (r *Results) Display(p par, l *logerr.Logerr) {

	l.Printf("%s %s\n", "Route                  ", p.GPXfile)
	l.Printf("%s %d\n", "Road segments          ", r.Segments)
	l.Printf("%s %d\n", "Track points dropped   ", r.TrkpRejected)
	l.Printf("%s %5.1f\n", "Distance (km)          ", r.DistTotal)
	l.Printf("%s\n", "Elevation (m)")
	l.Printf("%s %4.0f\n", "    Up               ", r.EleUp)
	l.Printf("%s %4.0f\n", "    Down             ", r.EleDown)

	if r.Filtered > 0 {
		l.Printf("%s %4.0f\n", "    Up GPX           ", r.EleUpGPX)
		l.Printf("%s %4.0f\n", "    Down GPX         ", r.EleDownGPX)
		// l.Printf("%s %4.0f  %s\n", "    Filterable      ", r.Filterable, "(approx.)")
		l.Printf("%s %4.0f  %4.1f%s\n", "    Filtered         ",
			r.Filtered, r.FilteredPros, "%")

		l.Printf("%s %4.0f\n", "    Interpolated     ", r.Filtered-r.EleLevelled)
		l.Printf("%s %4.0f\n", "    Levelled         ", r.EleLevelled)

		l.Printf("%s %6d\n", "Filter rounds      ", r.FilterRounds)
		l.Printf("%s %6d\n", "Interpolations     ", r.Ipolations)
		l.Printf("%s %6d\n", "Levelations        ", r.Levelations)
	}
	l.Printf("%s %5.1f\n", "Min grade %         ", r.MinGrade)
	l.Printf("%s %4.1f\n", "Max grade %          ", r.MaxGrade)
	l.Printf("%s %5.2f\n", "Grade sign change %  ", r.GradeSignChange)
	l.Printf("%s %6.1f\n", "Energy rider (Wh)   ", r.JriderTotal)
	l.Printf("%s %8.6f\n", "Ride time (h)      ", r.Time)
	if r.TimeUHBreaks > 0 {
		l.Printf("%s %7.3f\n\n", "Time with breaks (h)", r.Time+r.TimeUHBreaks)
	} else {
		l.Printf("%s", "\n")
	}
}
