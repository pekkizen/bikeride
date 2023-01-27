package main

import (
	"strconv"

	"bikeride/logerr"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
	"motion"
)

func min(x, y float64) float64 {
	if x < y {
		return x
	}
	return y
}
func max(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}


func ftoa(x float64, i int) string { return strconv.FormatFloat(x, 'f', i, 64) }

func setupEnvironment(c *motion.BikeCalc, m *power.Generator, p *param.Parameters,
	o *route.Route, l *logerr.Logerr) error {
	l.SetPrefix("setup:      ")

	setupCalculator(c, o, p)
	if err := setupParameters(p, o, c, l); err != nil {
		return err
	}
	setupPowerModel(m, p)
	l.SetPrefix("")
	return nil
}

func setupCalculator(c *motion.BikeCalc, o *route.Route, p *param.Parameters) {

	c.SetCdA(p.CdA)
	c.SetCrr(p.Crr)
	c.SetCbf(p.Ride.Cbf)
	if p.Ride.Ccf == 0 {
		p.Ride.Ccf = p.Ride.Cbf
	}
	c.SetWeightWheels(p.Weight.Wheels)
	c.SetWeight(p.Weight.Total)
	c.SetTolNR(p.NRtol)
	c.SetBracket(p.Bracket)
	c.SetVelErrors(p.VelErrors)
	c.SetVelSolver(p.VelSolver)

	gravity := p.Gravity
	if gravity <= 0 {
		gravity = c.LocalGravity(o.LatMean, o.EleMean)
	}
	c.SetGravity(gravity)
	o.Gravity = gravity

	if p.AirDensity > 0 {
		const L = -0.0065 //temperature lapse rate (per 1 m up)
		c.SetRho(p.AirDensity)
		o.Temperature = p.Temperature + (o.EleMean-p.BaseElevation)*L
		o.Rho = p.AirDensity
		return
	}
	c.SetBaseElevation(p.BaseElevation)
	c.SetAirPressure(p.AirPressure)
	c.SetTemperature(p.Temperature)
	// RhoFromEle uses base elevation, temperature, air pressure and gravity
	o.Rho, o.Temperature = c.RhoFromEle(o.EleMean)
	p.AirPressure = c.AirPressure()
	c.SetRho(o.Rho)
}

func setupPowerModel(m *power.Generator, p *param.Parameters) {

	P := &p.Powermodel

	m.SysGradeUp(P.UphillPowerGrade)
	m.SysGradeDown(P.DownhillPowerGrade)
	m.SysTailwind(P.SysTailwind)
	m.SysHeadwind(P.SysHeadwind)
	m.PowerTailRatio(P.TailWindPower / 100)
	m.PowerHeadRatio(P.HeadWindPower / 100)
	m.PowerUpRatio(P.UphillPower / P.FlatPower)
	m.PowerDownRatio(P.DownhillPower / 100)
	m.PowerDownTailRatio(P.DownhillTailwindPower / 100)
	m.PowerDownHeadRatio(P.DownhillHeadwindPower / 100)
	m.CUT(P.CUT)
	m.CUH(P.CUH)
	m.CDT(P.CDT)
	m.CDH(P.CDH)
	m.PowerModelType(P.PowermodelType)
	m.Setup() //must be done
}

func setupParameters(p *param.Parameters, o *route.Route, c *motion.BikeCalc, l *logerr.Logerr) error {

	p.DeltaVel = min(1, max(0.0001, p.DeltaVel))
	p.DeltaTime = min(1.5, max(0.0001, p.DeltaTime))

	if p.DiffCalc < 1 || p.DiffCalc > 3 {
		p.DiffCalc = 1
	}
	if !p.Ride.LimitTurnSpeeds && !p.Ride.LimitDownSpeeds {
		p.Ride.LimitEntrySpeeds = false
	}
	c.SetWind(0)

	if e := flatPowerSpeedCdA(p, c, l); e != nil {
		return e
	}
	if e := uphillPowerGradeSpeed(p, c, l); e != nil {
		return e
	}
	if e := maxPedaledSpeed(p, c, l); e != nil {
		return e
	}
	P := &p.Powermodel
	headWindPower := P.FlatPower * P.HeadWindPower / 100
	po := p.PowerOut
	if P.UphillPower < headWindPower {
		l.Err("uphillPower", ftoa(P.UphillPower*po, 0), "<= headWindPower", ftoa(headWindPower*po, 0))
	}
	if P.UphillPower <= P.FlatPower {
		l.Err("uphillPower", ftoa(P.UphillPower*po, 0), "<= flatPower", ftoa(P.FlatPower*po, 0))
	}
	if P.UphillPowerSpeed > P.FlatSpeed {
		l.Err("uphillPowerSpeed", ftoa(P.UphillPowerSpeed*ms2kmh, 1), "> flatSpeed", ftoa(P.FlatSpeed*ms2kmh, 1))
	}
	if P.UphillPowerSpeed < p.Ride.MinSpeed {
		l.Err("uphillPowerSpeed", ftoa(P.UphillPowerSpeed*ms2kmh, 1), "< MinSpeed", ftoa(p.Ride.MinSpeed*ms2kmh, 1))
	}
	if P.MaxPedaledSpeed < P.FlatSpeed+2.0/3.6 {
		l.Err("maxPedalledSpeed", ftoa(P.MaxPedaledSpeed*ms2kmh, 1), "< flatSpeed + 2 =", ftoa(2+P.FlatSpeed*ms2kmh, 1), "km/h")
	}
	if l.Errors() > 0 {
		return l.Errorf(" ")
	}
	return nil
}

func uphillPowerGradeSpeed(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {
	P := &p.Powermodel

	switch {
	case P.UphillPower > 0:
		P.VerticalUpSpeed, _ = c.VerticalUpFromPower(P.UphillPower, p.VerticalUpGrade)

	case P.VerticalUpSpeed > 0:
		P.UphillPower = c.PowerFromVerticalUp(P.VerticalUpSpeed, p.VerticalUpGrade)

	default:
		return l.Errorf("UphillPower <= 0 and VerticalUpSpeed <= 0")
	}

	switch {
	case P.UphillPowerSpeed > 0:
		P.UphillPowerGrade = c.GradeFromVelAndPower(P.UphillPowerSpeed, P.UphillPower)

	case P.UphillPowerGrade > 0:
		c.SetGrade(P.UphillPowerGrade)
		P.UphillPowerSpeed, _ = c.NewtonRaphson(P.UphillPower, 0, 3)

	default:
		return l.Errorf("uphillPowerSpeed <= 0 and uphillPowerGrade <= 0 ")
	}
	return nil
}

func flatPowerSpeedCdA(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {
	P := &p.Powermodel

	switch {
	case P.FlatPower > 0 && P.FlatSpeed > 0 && p.CdA <= 0:
		if p.CdA = c.CdAfromVelAndPower(P.FlatSpeed, P.FlatPower); p.CdA < 0.01 {
			return l.Errorf("%s%4.3f", "Not enough power for rolling resistance and proper cdA. cdA =", p.CdA)
		}
		c.SetCdA(p.CdA)
	case P.FlatPower > 0:
		P.FlatSpeed, _ = c.FlatSpeed(P.FlatPower)

	case P.FlatSpeed > 0:
		P.FlatPower = c.FlatPower(P.FlatSpeed)

	default:
		return l.Errorf("flaGroundtSpeed <= 0 and flatGroundPower <= 0 ")
	}
	return nil
}

func maxPedaledSpeed(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {
	P := &p.Powermodel
	const (
		minGRADE = -10.0 / 100.0
		velDIFF  = 1.5 / 3.6
	)

	downhillPower := P.FlatPower * P.DownhillPower / 100
	// Now MaxPedalledSpeed > 0 must be given
	// if P.MaxPedaledSpeed <= 0 {
	// 	c.SetGrade(P.DownhillPowerGrade)
	// 	vel, _ := c.NewtonRaphson(downhillPower, 0, 8)
	// 	P.MaxPedaledSpeed = vel + velDIFF
	// 	p.MinPedaledGrade = c.GradeFromVelAndPower(P.MaxPedaledSpeed, 0)
	// 	return nil
	// }
	p.MinPedaledGrade = c.GradeFromVelAndPower(P.MaxPedaledSpeed, 0)
	if p.MinPedaledGrade < minGRADE {
		p.MinPedaledGrade = minGRADE
		c.SetGrade(p.MinPedaledGrade)
		P.MaxPedaledSpeed = c.VelFreewheeling()
	}
	P.DownhillPowerGrade = c.GradeFromVelAndPower(P.MaxPedaledSpeed-velDIFF, downhillPower)
	return nil
}
