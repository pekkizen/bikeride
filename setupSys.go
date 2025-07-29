package main

import (
	"strconv"

	"github.com/pekkizen/bikeride/logerr"
	"github.com/pekkizen/bikeride/param"
	"github.com/pekkizen/bikeride/power"
	"github.com/pekkizen/bikeride/route"

	"github.com/pekkizen/motion"
)

func ftoa(x float64, i int) string { return strconv.FormatFloat(x, 'f', i, 64) }

func setupSystem(c *motion.BikeCalc, m *power.Generator, p *param.Parameters,
	o *route.Route, l *logerr.Logerr) error {
	l.SetPrefix("setup:      ")

	setupCalculator(c, o, p)
	if err := setupParameters(p, c, l); err != nil {
		return err
	}
	setupPowerModel(m, p)
	l.SetPrefix("")
	return nil
}

func setupCalculator(c *motion.BikeCalc, o *route.Route, p *param.Parameters) {
	b := &p.Bike
	e := &p.Environment

	c.SetCdA(b.CdA)
	c.SetCrr(b.Crr)
	c.SetCbf(b.Cbf)

	c.SetCcf(b.Ccf)
	c.SetWeightRotating(b.Weight.Rotating)
	c.SetWeight(b.Weight.Total)
	c.SetVelTol(p.VelTol)
	c.SetBracket(p.Bracket)
	c.SetVelErrors(p.VelErrors)
	if p.VelSolver < 1 {
		p.VelSolver = motion.NewtonRaphsonM
		if p.Environment.WindSpeed == 0 {
			p.VelSolver = motion.Householder3M
		}
	}

	c.SetVelSolver(p.VelSolver)
	p.VelSolver = c.SolverFunc()

	gravity := e.Gravity
	if gravity <= 0 {
		gravity = c.LocalGravity(o.LatMean, o.EleMean)
	}
	c.SetGravity(gravity)
	o.Gravity = gravity

	if e.AirDensity > 0 {
		const L = -0.0065 //temperature lapse rate (per 1 m up)
		c.SetRho(e.AirDensity)
		o.Temperature = e.Temperature + (o.EleMean-e.BaseElevation)*L
		o.Rho = e.AirDensity
		return
	}
	if e.AirPressure <= 0 {
		e.BaseElevation = 0
	}
	if e.AirPressure > 0 {
		c.SetAirPressure(e.AirPressure)
	}
	e.AirPressure = c.AirPressure()
	if e.BaseElevation < 0 {
		e.BaseElevation = 0
	}
	c.SetBaseElevation(e.BaseElevation)
	c.SetTemperature(e.Temperature)
	// RhoFromEle uses base elevation, temperature, air pressure and gravity
	// to calculate air density and temperature at elevation EleMean.
	o.Rho, o.Temperature = c.RhoFromEle(o.EleMean)
	c.SetRho(o.Rho)
}

func setupPowerModel(m *power.Generator, p *param.Parameters) {
	q := &p.Powermodel

	m.SysGradeUp(q.UphillPowerGrade)
	m.SysGradeDown(q.DownhillPowerGrade)
	m.SysTailwind(q.SysTailwind)
	m.SysHeadwind(q.SysHeadwind)
	m.PowerTailRatio(q.TailWindPower / 100)
	m.PowerHeadRatio(q.HeadWindPower / 100)
	m.PowerUpRatio(q.UphillPower / q.FlatPower)
	m.PowerDownRatio(q.DownhillPower / 100)
	if q.DownhillTailwindPower > 0 {
		m.PowerDownTailRatio(q.DownhillTailwindPower / 100)
	}
	if q.DownhillHeadwindPower > 0 {
		m.PowerDownHeadRatio(q.DownhillHeadwindPower / 100)
	}
	if q.CUT > 0 {
		m.CUT(q.CUT)
	}
	if q.CDT > 0 {
		m.CDT(q.CDT)
	}
	if q.CUH > 0 {
		m.CUH(q.CUH)
	}
	if q.CDT > 0 {
		m.CDT(q.CDT)
	}
	if q.CDH > 0 {
		m.CDH(q.CDH)
	}
	m.PowerModelType(q.PowermodelType)
	m.Setup() //must be done
}

func setupParameters(p *param.Parameters,
	c *motion.BikeCalc, l *logerr.Logerr) error {

	if p.DeltaVel < 0 {
		p.DeltaVel = 0.7
	}
	if p.DeltaTime < 0 {
		p.DeltaTime = 1
	}
	p.DeltaVel = min(2, max(0.0001, p.DeltaVel))
	p.DeltaTime = min(3, max(0.0001, p.DeltaTime))

	if p.AcceStepMode < 1 {
		p.AcceStepMode = 1
	}
	if p.AcceStepMode > 1 { // for braking to keep accuracy
		p.DeltaVel = min(p.DeltaTime*0.5, p.DeltaVel)
	}
	p.OdeltaVel = 1 / p.DeltaVel // for func velsteps

	if !p.Ride.LimitTurnSpeeds && !p.Ride.LimitDownSpeeds {
		p.Ride.LimitExitSpeeds = false
	}
	if !p.ReportTech {
		p.VelErrors = false
	}
	if e := flatPowerSpeedCdA(p, c, l); e != nil {
		return e
	}
	if e := uphillPowerGradeSpeed(p, c, l); e != nil {
		return e
	}
	if e := maxPedaledSpeed(p, c); e != nil {
		return e
	}
	q := &p.Powermodel
	if p.Ride.PowerAcceMin < 0 {
		p.Ride.PowerAcceMin = 0.75 * q.FlatPower
	}
	headWindPower := q.FlatPower * q.HeadWindPower / 100
	po := p.PowerOut
	if q.UphillPower < headWindPower {
		l.Err("uphillPower", ftoa(q.UphillPower*po, 0), "<= headWindPower",
			ftoa(headWindPower*po, 0))
	}
	if q.UphillPower <= q.FlatPower {
		l.Err("uphillPower", ftoa(q.UphillPower*po, 0), "<= flatPower",
			ftoa(q.FlatPower*po, 0))
	}
	if q.UphillPowerSpeed > q.FlatSpeed {
		l.Err("uphillPowerSpeed", ftoa(q.UphillPowerSpeed*ms2kmh, 1), "> flatSpeed",
			ftoa(q.FlatSpeed*ms2kmh, 1))
	}
	if q.UphillPowerSpeed < p.Ride.MinSpeed {
		l.Err("uphillPowerSpeed", ftoa(q.UphillPowerSpeed*ms2kmh, 1), "< MinSpeed",
			ftoa(p.Ride.MinSpeed*ms2kmh, 1))
	}
	if q.MaxPedaledSpeed < q.FlatSpeed {
		l.Err("maxPedalledSpeed", ftoa(q.MaxPedaledSpeed*ms2kmh, 1), "< flatSpeed",
			ftoa(q.FlatSpeed*ms2kmh, 1), "km/h")
	}
	if l.Errors() > 0 {
		return l.Errorf(" ")
	}
	return nil
}

func uphillPowerGradeSpeed(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {
	q := &p.Powermodel
	switch {
	case q.UphillPower > 0:
		q.VerticalUpSpeed = c.VerticalUpFromPower(q.UphillPower, q.VerticalUpGrade)

	case q.VerticalUpSpeed > 0:
		q.UphillPower = c.PowerFromVerticalUp(q.VerticalUpSpeed, q.VerticalUpGrade)

	default:
		return l.Errorf("UphillPower <= 0 and VerticalUpSpeed <= 0")
	}
	switch {
	case q.UphillPowerSpeed > 0:
		q.UphillPowerGrade = c.GradeFromVelAndPower(q.UphillPowerSpeed, q.UphillPower)

	case q.UphillPowerGrade > 0:
		c.SetGradeExact(q.UphillPowerGrade)
		q.UphillPowerSpeed, _ = c.NewtonRaphson(q.UphillPower, minTolNR, 3)

	default:
		return l.Errorf("uphillPowerSpeed <= 0 and uphillPowerGrade <= 0 ")
	}
	return nil
}

func flatPowerSpeedCdA(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {
	q := &p.Powermodel
	b := &p.Bike
	switch {
	case q.FlatPower > 0 && q.FlatSpeed > 0 && b.Crr > 0 && b.CdA <= 0:
		if b.CdA = c.CdAfromVelAndPower(q.FlatSpeed, q.FlatPower); b.CdA < 0.001 {
			return l.Errorf("%s%4.3f",
				"Not enough power for rolling resistance and proper CdA. CdA =", b.CdA)
		}
		c.SetCdA(b.CdA)

	case q.FlatPower > 0 && q.FlatSpeed > 0 && b.Crr <= 0 && b.CdA > 0:
		if b.Crr = c.CrrFromVelAndPower(q.FlatSpeed, q.FlatPower); b.Crr <= 0 {
			return l.Errorf("%s%4.3f",
				"Not enough power for air resistance and proper Crr. Crr =", b.Crr)
		}
		c.SetCrr(b.Crr)

	case q.FlatPower > 0 && b.Crr > 0 && b.CdA > 0:
		q.FlatSpeed = c.FlatSpeed(q.FlatPower)

	case q.FlatSpeed > 0 && b.Crr > 0 && b.CdA > 0:
		q.FlatPower = c.FlatPower(q.FlatSpeed)

	default:
		return l.Errorf("flatGroundSpeed <= 0 and flatGroundPower <= 0 ")
	}
	return nil
}

func maxPedaledSpeed(p *param.Parameters, c *motion.BikeCalc) error {
	q := &p.Powermodel
	const (
		minGRADE    = -12.0 / 100.0
		velDIFF     = 1.5 / 3.6
		dhPowerPros = 20
	)
	if q.DownhillPower < 0 {
		q.DownhillPower = dhPowerPros
	}
	downhillPower := q.FlatPower * q.DownhillPower / 100
	// Now MaxPedaledSpeed > 0 must be given
	// if P.MaxPedaledSpeed <= 0 {
	// 	c.SetGradeExact(P.DownhillPowerGrade)
	// 	vel, _ := c.NewtonRaphson(downhillPower, 0, 8)
	// 	P.MaxPedaledSpeed = vel + velDIFF
	// 	p.MinPedaledGrade = c.GradeFromVelAndPower(P.MaxPedaledSpeed, 0)
	// 	return nil
	// }
	q.MinPedaledGrade = c.GradeFromVelAndPower(q.MaxPedaledSpeed, 0)
	if q.MinPedaledGrade < minGRADE {
		q.MinPedaledGrade = minGRADE
		c.SetGradeExact(q.MinPedaledGrade)
		q.MaxPedaledSpeed = c.VelFreewheel() //+ 0.001
	}
	q.DownhillPowerGrade = c.GradeFromVelAndPower(q.MaxPedaledSpeed-velDIFF, downhillPower)
	return nil
}
