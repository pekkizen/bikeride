package route

import (
	"encoding/json"
	"io"

	"github.com/pekkizen/bikeride/logerr"
	"github.com/pekkizen/numconv"
)

/*
WriteCSV uses makeRouteCSV to build the route CSV as single []byte slice.
makeRouteCSV uses package numconv for formatting numbers directly to the slice.
This is much faster than using strconv FormatFloat and
FormatInt functions. And encoding/csv writer.
*/
func (o *Route) WriteCSV(p par, writer io.WriteCloser) error {
	const segmentBytes = 140
	routeCSV := make([]byte, 0, segmentBytes*o.segments)
	routeCSV = o.makeRouteCSV(routeCSV, p)
	_, err := writer.Write(routeCSV)
	if err == nil {
		err = writer.Close()
	}
	routeCSV = nil
	return err
}

func (r *Results) WriteTXT(p par, writer io.WriteCloser) error {
	result := make([]byte, 0, 4000)
	result = r.makeResultTXT(result, p)
	_, err := writer.Write(result)
	if err == nil {
		err = writer.Close()
	}
	result = nil
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
	json = nil
	return err
}

func header(b []byte, sep byte, useCR bool) []byte {
	b = append(b, "seg"...)
	b = append(b, sep)
	b = append(b, "lat"...)
	b = append(b, sep)
	b = append(b, "lon"...)
	b = append(b, sep)

	b = append(b, "eleGPX"...)
	b = append(b, sep)
	b = append(b, "ele"...)
	b = append(b, sep)
	b = append(b, "eleShift"...)
	b = append(b, sep)
	b = append(b, "wind"...)
	b = append(b, sep)
	b = append(b, "grade"...)
	b = append(b, sep)
	b = append(b, "dist"...)

	b = append(b, sep)
	b = append(b, "radius"...)
	b = append(b, sep)
	b = append(b, "course"...)
	b = append(b, sep)

	b = append(b, "pTarget"...)
	b = append(b, sep)
	b = append(b, "pRider"...)
	b = append(b, sep)
	b = append(b, "pBrake"...)
	b = append(b, sep)

	b = append(b, "vFreew"...)
	b = append(b, sep)
	b = append(b, "vTarget"...)
	b = append(b, sep)
	b = append(b, "vMax"...)
	b = append(b, sep)
	b = append(b, "vEntry"...)
	b = append(b, sep)
	b = append(b, "vExit"...)
	b = append(b, sep)

	b = append(b, "time"...)
	b = append(b, sep)
	b = append(b, "cumTime"...)
	b = append(b, sep)
	b = append(b, "cumDist"...)
	b = append(b, sep)
	b = append(b, "calcP"...)
	b = append(b, sep)
	b = append(b, "calcS"...)
	b = append(b, sep)
	b = append(b, "jSum"...)

	if useCR {
		b = append(b, '\r')
	}
	b = append(b, '\n')
	return b
}

func (o *Route) makeRouteCSV(b []byte, p par) []byte {

	var sep byte = ','
	if p.CSVuseTab {
		sep = '\t'
	}
	var cumsec, cumdist float64

	b = header(b, sep, p.UseCR)

	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		powerTarget := s.powerTarget
		if powerTarget > 0 { //rider power
			powerTarget *= p.PowerOut
		}
		b = numconv.Utoa8(b, uint64(s.segnum), sep)
		b = numconv.Ftoa86(b, s.lat, sep)
		b = numconv.Ftoa86(b, s.lon, sep)
		b = numconv.Ftoa82(b, s.eleGPX, sep)
		b = numconv.Ftoa82(b, s.ele, sep)
		b = numconv.Ftoa82(b, s.ele-s.eleGPX, sep)
		b = numconv.Ftoa82(b, s.wind, sep)
		b = numconv.Ftoa82(b, s.grade*100, sep)
		b = numconv.Ftoa82(b, s.dist, sep)
		b = numconv.Ftoa80(b, s.radius, sep)
		b = numconv.Ftoa80(b, s.course*(180/π), sep)
		b = numconv.Ftoa80(b, powerTarget, sep)
		b = numconv.Ftoa80(b, s.powerRider*p.PowerOut, sep)
		b = numconv.Ftoa80(b, s.powerBraking, sep)
		b = numconv.Ftoa82(b, s.vFreewheel*ms2kmh, sep)
		b = numconv.Ftoa82(b, s.vTarget*ms2kmh, sep)
		b = numconv.Ftoa82(b, s.vMax*ms2kmh, sep)
		b = numconv.Ftoa82(b, s.vEntry*ms2kmh, sep)
		b = numconv.Ftoa82(b, s.vExit*ms2kmh, sep)
		b = numconv.Ftoa82(b, s.time, sep)
		b = numconv.Ftoa83(b, cumsec*s2h, sep)
		b = numconv.Ftoa83(b, cumdist*m2km, sep)
		b = numconv.Utoa8(b, uint64(s.calcPath), sep)
		b = numconv.Utoa8(b, uint64(s.calcSteps), sep)
		if p.UseCR {
			b = numconv.Ftoa81(b, s.jouleNetSum, '\r')
			b = append(b, '\n')
		} else {
			b = numconv.Ftoa81(b, s.jouleNetSum, '\n')
		}
		cumdist += s.dist
		cumsec += s.time
	}
	return b
}

func (r *Results) makeResultTXT(b []byte, p par) []byte {

	const (
		d1  = 1
		d2  = 2
		d3  = 3
		d4  = 4
		d5  = 5
		d7  = 7
		d8  = 8
		d9  = 9
		d10 = 10
	)
	var le = "\n"
	if p.UseCR {
		le = "\r\n"
	}
	wF := func(b []byte, s string, f float64, dec int, lend string) []byte {
		b = append(b, s...)
		b = append(b, '\t')
		if f >= 0 {
			b = append(b, ' ')
		}
		b = numconv.Ftoa(b, f, dec, '0')
		b = numconv.TrimTrailingZeros(b)
		return append(b, lend...)
	}
	wI := func(b []byte, s string, f float64, lend string) []byte {
		b = append(b, s...)
		b = append(b, '\t')
		if f >= 0 {
			b = append(b, ' ')
		}
		b = numconv.Ftoa(b, f, 0, ' ')
		return append(b, lend...)
	}
	wS := func(b []byte, s string, x string, lend string) []byte {
		b = append(b, s...)
		b = append(b, '\t')
		b = append(b, ' ')
		b = append(b, x...)
		return append(b, lend...)
	}
	header := func(b []byte) []byte {
		b = wS(b, "Route name       \t", p.RouteName, le)
		b = wS(b, "Ride parameters  \t", p.RideJSON, le)
		b = wS(b, "Configuration    \t", p.ConfigJSON, le)
		b = wS(b, "Route GPX file   \t", p.GPXfile, le)
		if r.TrkpErrors > 0 {
			b = wI(b, "Track point errors \t", float64(r.TrkpErrors), le)
		}
		return b
	}
	environment := func(b []byte) []byte {
		b = append(b, le+"Environment"+le...)
		b = wI(b, "\tWind course (deg)       ", r.WindCourse, le)
		b = wF(b, "\tWind speed (m/s)        ", r.WindSpeed, d1, le)
		if r.RouteCourse >= 0 {
			b = wI(b, "\tRoute course (deg)      ", r.RouteCourse, le)
		}
		b = wI(b, "\tMean elevation (m)      ", r.EleMean, le)
		if p.Environment.AirDensity < 0 {
			b = wF(b, "\t    temperature (C)     ", r.Temperature, d1, le)
			b = wF(b, "\t    air density (kg/m^3)", r.Rho, d3, le)
		} else {
			b = wF(b, "\tAir density (kg/m^3)    ", r.Rho, d3, le)
		}
		if p.Environment.AirDensity < 0 {
			b = wI(b, "\tBase elevation (m)      ", r.BaseElevation, le)
			b = wF(b, "\t    temperature (C)     ", p.Environment.Temperature, d1, le)
			b = wF(b, "\t    air density (kg/m^3)", r.RhoBase, d3, le)
			b = wF(b, "\t    air pressure (hPa)  ", r.AirPressure, d2, le)
		}
		return b
	}
	filtering := func(b []byte) []byte {
		b = append(b, le+"Filtering"+le...)
		if r.Filtered == 0 {
			b = wI(b, "\tFiltered  (m)     ", 0, le)
			return b
		}
		b = wI(b, "\tGPX elevation up (m)  ", r.EleUpGPX, le)
		b = wI(b, "\tGPX elevation down (m)", r.EleDownGPX, le)
		b = wI(b, "\tFilterable (m)        ", r.Filterable, le)
		b = wI(b, "\tFiltered (m)          ", r.Filtered, le)
		// b = wF(b, "\tFiltered  distance (%)", r.FilteredDistPros, d2, le)
		if r.Filterable > 0 {
			b = wF(b, "\t    % filterable      ", 100*r.Filtered/r.Filterable, d1, le)
		}
		if r.EleUpGPX > 0 {
			b = wF(b, "\t    % up GPX          ", 100*r.Filtered/r.EleUpGPX, d1, le)
		}
		if r.EleDownGPX > 0 {
			b = wF(b, "\t    % down GPX        ", 100*r.Filtered/r.EleDownGPX, d1, le)
		}
		if r.Levelations > 0 {
			b = wI(b, "\tLeveled (m)           ", r.EleLevelled, le)
			b = wI(b, "\tLevelations           ", float64(r.Levelations), le)
		}
		if r.FilterRounds > 0 {
			b = wI(b, "\tInterpolate rounds    ", float64(r.FilterRounds), le)
		}
		if r.Ipolations > 0 {
			b = wI(b, "\tInterpolations        ", float64(r.Ipolations), le)
		}
		b = wF(b, "\tRoad smoothess index  ", r.RelGradeChange, d2, le)
		return b
	}
	elevation := func(b []byte) []byte {
		b = append(b, le+"Elevation (m) "+le...)
		b = wI(b, "\tUp               ", r.EleUp, le)
		b = wI(b, "\tDown             ", r.EleDown, le)
		b = wI(b, "\tMax              ", r.EleMax, le)
		b = wI(b, "\tMin              ", r.EleMin, le)
		b = wI(b, "\tUp by momentum   ", r.EleUpKinetic, le)
		return b
	}
	roadsegments := func(b []byte) []byte {
		b = wI(b, le+"Road segments         \t", float64(r.Segments), le)
		b = wI(b, "\tTrack points dropped  ", float64(r.TrkpRejected), le)
		b = wS(b, "\tLength (m) ", " ", le)
		b = wF(b, "\t  mean                ", r.DistMean, d1, le)
		b = wF(b, "\t  median              ", r.DistMedian, d1, "\t(approx.)"+le)
		b = wF(b, "\t  max                 ", r.DistMax, d1, le)
		b = wF(b, "\t  min                 ", r.DistMin, d1, le)
		b = wS(b, "\tGrade (%) ", " ", le)
		b = wF(b, "\t  max                  ", r.MaxGrade, d1, le)
		b = wF(b, "\t  min                  ", r.MinGrade, d1, le)
		return b
	}
	distance := func(b []byte) []byte {
		b = wF(b, le+"Distance (km)    \t", r.DistTotal, d1, le)
		b = wF(b, "\tDirect           ", r.DistDirect, d1, le)
		b = wF(b, "\tUphill   >  4%   ", r.DistUphill, d1, le)
		b = wF(b, "\tDownhill < -4%   ", r.DistDownhill, d1, le)
		b = wF(b, "\tFlat   -1% - 1%  ", r.DistFlat, d1, le)
		b = wF(b, "\tBraking          ", r.DistBrake, d1, le)
		b = wF(b, "\tFreewheeling     ", r.DistFreewheel, d1, le)
		b = wF(b, "\tRider powered    ", r.DistRider, d1, le)
		return b
	}
	speed := func(b []byte) []byte {
		b = append(b, le+"Speed (km/h)"+le...)
		b = wF(b, "\tMean                 ", r.VelAvg, d2, le)
		b = wF(b, "\tMax                  ", r.VelMax, d2, le)
		b = wF(b, "\tMin                  ", r.VelMin, d2, le)
		b = wF(b, "\tDownhill < -4% mean  ", r.VelDownhill, d2, le)
		b = wI(b, "\tDown vertical (m/h)  ", r.VelDownVert, le)
		return b
	}
	drivingtime := func(b []byte) []byte {
		b = wF(b, le+"Driving time (h)       \t", r.Time, d2, le)
		b = wF(b, "\tFrom target speeds     ", r.TimeTargetSpeeds, d2, le)
		b = wF(b, "\tPedal powered          ", r.TimeRider, d2, le)
		b = wF(b, "\tBraking                ", r.TimeBraking, d2, le)
		b = wF(b, "\tFreewheeling           ", r.TimeFreewheel, d2, le)
		if r.TimeUHBreaks > 0 {
			b = wF(b, "\tWith uphill breaks     ", r.Time+r.TimeUHBreaks, d2, le)
			b = wF(b, "\tUphill break time      ", r.TimeUHBreaks, d2, le)
		}
		if p.UphillBreak.PowerLimit > 0 {
			b = append(b, "\tOver "...)
			// b = numconv.Itoa(b, ftoi(p.UphillBreak.PowerLimit), '%')
			b = numconv.Ftoa(b, p.UphillBreak.PowerLimit, 0, '%')
			b = wF(b, " uphill power  ", r.TimeFullPower, d2, le)
		}
		b = wF(b, "\tOver flat ground power ", r.TimeOverFlatPower, d2, le)
		return b
	}
	// Joules below are converted to Wh before
	energyrider := func(b []byte) []byte {
		b = wI(b, le+"Energy rider (Wh)          \t", r.JriderTotal, le)
		b = wI(b, "\tFrom target powers (Wh)  ", r.JfromTargetPower, le)
		b = wI(b, "\tFood (kcal)              ", r.FoodRider, le)
		b = wI(b, "\tBananas (pcs)            ", r.BananaRider, le)
		b = wI(b, "\tLard (g)                 ", r.FatRider, le)
		b = wI(b, "\tAverage power (W)        ", r.PowerRiderAvg, le)
		b = wF(b, "\tEnergy/distance (Wh/km)  ", r.JriderTotal/r.DistTotal, d2, le)
		return b
	}

	riderenergyusage := func(b []byte) []byte {
		c := 100.0 / r.JriderTotal
		b = wS(b, le+"Rider's energy usage \t\t  Wh", "%", le)
		b = wI(b, "\tGravity up          ", r.JriderGravUp, "  \t")
		b = numconv.Ftoa(b, c*r.JriderGravUp, d1, 0)
		b = append(b, le...)
		b = wI(b, "\tAir resistance      ", r.JriderDrag, "  \t")
		b = numconv.Ftoa(b, c*r.JriderDrag, d1, 0)
		b = append(b, le...)
		b = wI(b, "\tRolling resistance  ", r.JriderRoll, "  \t")
		b = numconv.Ftoa(b, c*r.JriderRoll, d1, 0)
		b = append(b, le...)
		b = wI(b, "\tDrivetrain loss     ", r.JlossDT, "  \t")
		b = numconv.Ftoa(b, c*r.JlossDT, d1, 0)
		b = append(b, le...)
		b = wF(b, "\tEnergy net sum      ", r.EnergySumRider, d1, le)
		b = wI(b, "\tAcceleration        ", r.JriderAcce, "  \t(included above)"+le)
		return b
	}

	totalenergybalance := func(b []byte) []byte {
		b = append(b, le+"Total energy balance (Wh)"+le...)
		b = wI(b, "\tRider               ", r.JriderTotal, le)
		b = wI(b, "\tDrivetrain loss     ", -r.JlossDT, le)
		b = wF(b, "\tKinetic resistance  ", r.JkineticAcce, d1, le)
		b = wF(b, "\tKinetic push        ", r.JkineticDece, d1, le)
		b = wI(b, "\tGravity up          ", r.JgravUp, le)
		b = wI(b, "\tGravity down        ", r.JgravDown, le)
		b = wI(b, "\tAir resistance      ", r.JdragResist, le)
		b = wI(b, "\t    pedaling     ", r.JdragRider, le)
		b = wI(b, "\t    freewheeling ", r.JdragFreewheel, le)
		b = wI(b, "\t    braking      ", r.JdragBrake, le)
		if r.JdragPush >= 10 {
			b = wI(b, "\tWind push           ", r.JdragPush, le)
		} else if r.JdragPush > 0.1 {
			b = wF(b, "\tWind push           ", r.JdragPush, d1, le)
		}
		b = wI(b, "\tRolling resistance  ", r.Jroll, le)
		b = wI(b, "\tBraking             ", r.Jbraking, le)
		// b = wF(b, "\n\tEnergy net error (Wh)", r.EnergySumTotal, d2, le)
		plusEnergy := r.JriderTotal + r.JkineticDece + r.JgravDown + r.JdragPush
		b = wF(b, "\n\tEnergy net error (%)", 100*r.EnergySumTotal/plusEnergy, d3, le)
		return b
	}

	rider := func(b []byte) []byte {
		q := p.Powermodel
		b = append(b, le+"Rider/riding"+le...)
		b = wI(b, "\tTotal weight (kg)          ", p.Bike.Weight.Total, le)
		b = wF(b, "\tFlat ground speed (km/h)   ", q.FlatSpeed, d2, le)
		b = wI(b, "\tFlat ground power (W)      ", q.FlatPower, le)
		b = wF(b, "\tAir drag coefficient CdA   ", p.Bike.CdA, d3, le)
		b = wF(b, "\tRolling res. coeff. Crr    ", p.Bike.Crr, d3, le)
		b = append(b, " "+le...)
		b = wI(b, "\tUphill power (W)           ", q.UphillPower, le)
		b = wI(b, "\tVertical speed up (m/h)    ", q.VerticalUpSpeed, le)
		b = wF(b, "\tUphill power speed (km/h)  ", q.UphillPowerSpeed, d1, le)
		b = wF(b, "\tUphill power grade (%)     ", q.UphillPowerGrade, d1, le)
		if r.MaxGradeUp > 0 {
			b = wF(b, "\tUphill power max grade (%)", r.MaxGradeUp, d1, "\t(")
			b = numconv.Ftoa(b, p.Ride.MinSpeed, d1, 0)
			b = append(b, " km/h)"+le...)
		}
		b = append(b, " "+le...)
		b = wF(b, "\tMax pedaled speed (km/h)  ", q.MaxPedaledSpeed, d1, le)
		b = wF(b, "\tMin pedaled grade (%)     ", q.MinPedaledGrade, d2, "\t(")
		b = numconv.Ftoa(b, q.MaxPedaledSpeed, d1, 0)
		b = append(b, " km/h)"+le...)
		if p.ReportTech {
			b = wF(b, "\tDownhill power grade (%)  ", q.DownhillPowerGrade, d2, le)
			b = wF(b, "\tDownhill power (%)        ", q.DownhillPower, d1, le)
			b = wF(b, "\tDownhill power speed (km/h)", r.DownhillPowerSpeed, d1, le)
		}
		if !p.Ride.LimitDownSpeeds {

			b = append(b, " "+le...)
			if p.Ride.BrakingDist <= 0 {
				b = wI(b, "\tVertical speed down (m/h)", p.Ride.VerticalDownSpeed, le)
			}
			if p.Ride.BrakingDist > 0 {
				b = wI(b, "\tBraking distance (m)    ", p.Ride.BrakingDist, le)
				b = wF(b, "\tBraking flatground g-force", p.Bike.Cbf, d2, le)
			}
			b = wF(b, "\tDownhill (-6%) max speed", r.DownhillMaxSpeed, d1, le)
		}
		return b
	}
	technical := func(b []byte) []byte {
		b = append(b, le+"Calculation"+le...)
		b = wF(b, "\tSegment energy mean error (J)     ", r.SegEnergyMean, d2, le)
		b = wF(b, "\tSegment energy mean abs(error) (J)", r.SegEnergyMeanAbs, d2, le)
		b = wF(b, "\tJoule waste bin (J)               ", r.Jsink, d2, le)
		b = wF(b, "\tMean latitude (deg)                ", r.LatMean, d2, le)
		b = wF(b, "\tLocal gravity                      ", r.Gravity, d3, le)

		b = append(b, "\n\tStepping "...)
		switch p.AcceStepMode {
		case stepVel:
			b = append(b, "velocity de/increments"...)
		case stepDist:
			b = append(b, "distance increments"...)
		case stepTime:
			b = append(b, "time increments"...)
		}
		b = wF(b, "\n\t  steps per road segment (mean)     ", r.CalcStepsAvg, d2, le)
		b = wF(b, "\t  single step segments (%)          ", r.SingleStepPros, d1, le)
		switch p.AcceStepMode {
		case stepVel:
			b = wF(b, "\t  step lenght (m/s)                 ", p.DeltaVel, d4, le)
		case stepTime:
			b = wF(b, "\t  step lenght (s)                   ", p.DeltaTime, d4, le)
		case stepDist:
			b = wF(b, "\t  step lenght  (m = dTime x v)      ", p.DeltaTime, d4, " x speed"+le)
		}

		b = append(b, le+"\tVelocity solver: "...)
		switch p.VelSolver {
		case newtonRaphson:
			b = append(b, "Newton-Raphson"...)
		case newtonHalley:
			b = append(b, "Newton-Halley"...)
		case householder3:
			b = append(b, "Householder3"...)
		case singleQuadratic:
			b = append(b, "Quadratic interpolation"...)
		case doubleQuadratic:
			b = append(b, "Two quadratic interpolations"...)
		case doubleLinear:
			b = append(b, "Two linear interpolations"...)
		case singleLinear:
			b = append(b, "Linear interpolation"...)
		default:
			b = append(b, "Newton-Halley"...)
		}

		b = append(b, le...)
		if p.VelSolver == newtonRaphson || p.VelSolver == newtonHalley ||
			p.VelSolver == householder3 {
			b = wF(b, "\t  iterations (mean)           ", r.SolverRoundsAvg, d2, le)
			b = wI(b, "\t  iterations (max)            ", float64(r.MaxIter), le)
		} else {
			b = wF(b, "\t  function evaluations (mean)", r.SolverRoundsAvg, d2, le)
		}
		b = wF(b, "\t  solver tolerance (m/s)      ", r.VelTol, d7, le)
		if r.SolverCalls > 0 {
			b = wI(b, "\t  calls solver                ", float64(r.SolverCalls), le)
		}
		if r.Counter > 0 {
			b = wI(b, "\t  event counter               ", float64(r.Counter), le)
		}
		if !p.VelErrors {
			return b
		}
		b = wS(b, le+"\tTarget velocity error (m/s)", " ", le)
		b = wF(b, "\t  solver tolerance (m/s)   ", r.VelTol, d7, le)
		b = wF(b, "\t  mean abs(error)    \t", r.VelErrorAbsMean, d10, le)
		b = wF(b, "\t  mean error, bias   \t", r.VelErrorMean, d10, le)
		b = wF(b, "\t  max abs(error)     \t", r.VelErrorMax, d10, le)
		b = wF(b, "\t  errors > 0 (%)     \t", r.VelErrorPos*100, d1, le)
		return b
	}

	b = header(b)
	b = environment(b)
	b = filtering(b)
	b = elevation(b)
	b = roadsegments(b)
	b = distance(b)
	b = speed(b)
	b = drivingtime(b)
	b = energyrider(b)
	b = riderenergyusage(b)
	b = rider(b)
	b = totalenergybalance(b)
	if p.ReportTech {
		b = technical(b)
	}
	return b

}

func tohhmmss(hours float64, l *logerr.Logerr) string {
	h := int(hours)
	m := int((hours - float64(h)) * 60)
	s := int((hours - float64(h) - float64(m)/60) * 3600)
	return l.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (r *Results) Display(p par, l *logerr.Logerr) {

	r.checkEnergySums(l)

	l.Printf("%s %s\n", "Route                  ", p.GPXfile)
	l.Printf("%s %d\n", "Road segments          ", r.Segments)
	if r.TrkpRejected > 0 {
		l.Printf("%s %d\n", "Track points dropped   ", r.TrkpRejected)
	}
	l.Printf("%s\n", "Distance (km) ")
	l.Printf("%s %5.3f\n", "    GPX              ", r.DistGPX)
	l.Printf("%s %5.3f\n", "    Filtered         ", r.DistTotal)
	l.Printf("%s\n", "Elevation (m)")
	l.Printf("%s %4.0f\n", "    Up               ", r.EleUp)
	l.Printf("%s %4.0f\n", "    Down             ", r.EleDown)
	if r.Filtered > 0 || p.Filter.SmoothingWeight > 0 {
		l.Printf("%s %4.0f\n", "    Up GPX           ", r.EleUpGPX)
		l.Printf("%s %4.0f\n", "    Down GPX         ", r.EleDownGPX)
		l.Printf("%s %4.0f\n", "    Max              ", r.EleMax)
		l.Printf("%s %4.0f\n", "    Min              ", r.EleMin)
		l.Printf("%s %4.0f \n", "    Filtered         ", r.Filtered)
		l.Printf("%s  %4.1f%s\n", "    From filterable  ", r.FilteredPros, " %")
		if r.EleLevelled > 0 {
			l.Printf("%s %4.0f\n", "    Leveled          ", r.EleLevelled)
		}
		if r.Ipolations > 0 {
			l.Printf("%s %6d\n", "Interpolations     ", r.Ipolations)
			if r.FilterRounds > 0 {
				l.Printf("%s %6d\n", "    rounds         ", r.FilterRounds)
			}
		}
		if r.Levelations > 0 {
			l.Printf("%s %6d\n", "Levelations        ", r.Levelations)
		}
	}
	l.Printf("%s %5.1f\n", "Min grade %          ", r.MinGrade)
	l.Printf("%s %4.1f\n", "Max grade %           ", r.MaxGrade)
	l.Printf("%s %5.2f\n", "Road smoothess index ", r.RelGradeChange)
	l.Printf("%s %5.2f\n", "Energy/dist (Wh/km)  ", r.JriderTotal/r.DistTotal)
	l.Printf("\n%s %6.1f\n", "Energy rider (Wh)    ", r.JriderTotal)
	l.Printf("%s %8s\n", "Ride time (hh:mm:ss)  ", tohhmmss(r.Time, l))
	if r.TimeUHBreaks > 0 {
		l.Printf("%s %8s\n", "Time with breaks (h)  ", tohhmmss(r.Time+r.TimeUHBreaks, l))
	}

}
