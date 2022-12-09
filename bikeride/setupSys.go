package main

import (
	"strconv"

	"bikeride/logerr"
	"bikeride/motion"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
)

func ftoa(x float64, i int) string { return strconv.FormatFloat(x, 'f', i, 64) }

func setupEnvironment(c *motion.BikeCalc, m *power.Generator, p *param.Parameters,
	o *route.Route, l *logerr.Logerr) error {
	l.SetPrefix("setup:      ")

	setupCalculator(c, o, p, l)
	if err := setupParameters(p, o, c, l); err != nil {
		return err
	}
	setupPowerModel(m, p)
	l.SetPrefix("")
	return nil
}

func setupCalculator(c *motion.BikeCalc, o *route.Route, p *param.Parameters,
	l *logerr.Logerr) {

	c.SetCdA(p.CdA)
	c.SetCrr(p.Crr)
	c.SetCbf(p.Cbf)
	if p.Ccf == 0 {
		p.Ccf = p.Cbf
	}
	c.SetWeightWheels(p.Weight.Wheels)
	c.SetWeight(p.Weight.Total)
	c.SetVelTol(p.NRtol)
	c.SetBracket(p.Bracket)
	c.SetVelErrors(p.CalcVelError)
	c.SetMinPower(p.MinPower)
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

	if P.DownhillHeadwindPower <= 0 {
		P.DownhillHeadwindPower = 100
	}
	if P.DownhillTailwindPower <= 0 {
		P.DownhillTailwindPower = 5
	}
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

	if p.DiffCalc < 1 || p.DiffCalc > 3 {
		p.DiffCalc = 1
	}
	if !p.LimitTurnSpeeds && !p.LimitDownSpeeds {
		p.LimitEntrySpeeds = false
	}
	c.SetWind(0)

	if e := flatPowerSpeedCdA(p, c, l); e != nil {
		return e
	}
	if e := uphillPowerGradeSpeed(p, c, l); e != nil {
		return e
	}
	if e := maxPedalledSpeed(p, c, l); e != nil {
		return e
	}

	P := &p.Powermodel
	headWindPower := P.FlatPower * P.HeadWindPower / 100
	pout := p.PowerOut
	if P.UphillPower < headWindPower {
		l.Err("uphillPower", ftoa(P.UphillPower*pout, 0), "<= headWindPower", ftoa(headWindPower*pout, 0))
	}
	if P.UphillPower <= P.FlatPower {
		l.Err("uphillPower", ftoa(P.UphillPower*pout, 0), "<= flatPower", ftoa(P.FlatPower*pout, 0))
	}
	if P.UphillPowerSpeed > P.FlatSpeed {
		l.Err("uphillPowerSpeed", ftoa(P.UphillPowerSpeed*ms2kmh, 1), "> flatSpeed", ftoa(P.FlatSpeed*ms2kmh, 1))
	}
	if P.UphillPowerSpeed < p.MinSpeed {
		l.Err("uphillPowerSpeed", ftoa(P.UphillPowerSpeed*ms2kmh, 1), "< MinSpeed", ftoa(p.MinSpeed*ms2kmh, 1))
	}
	if P.MaxPedalledSpeed < P.FlatSpeed+2.0/3.6 {
		l.Err("maxPedalledSpeed", ftoa(P.MaxPedalledSpeed*ms2kmh, 1), "< flatSpeed + 2 =", ftoa(2+P.FlatSpeed*ms2kmh, 1), "km/h")
	}
	if l.Errors() > 0 {
		return l.Errorf(" ")
	}
	return nil
}

func uphillPowerGradeSpeed(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {

	P := &p.Powermodel
	switch {
	case P.UphillPower > 0 && P.VerticalUpSpeed <= 0:
		P.VerticalUpSpeed, _ = c.VerticalUpFromPower(P.UphillPower, p.VerticalUpGrade)

	case P.UphillPower <= 0 && P.VerticalUpSpeed > 0:
		P.UphillPower = c.PowerFromVerticalUp(P.VerticalUpSpeed, p.VerticalUpGrade)

	case P.UphillPower <= 0 && P.VerticalUpSpeed <= 0:
		return l.Err("UphillPower <= 0 and VerticalUpSpeed <= 0")
	}
	switch {
	case P.UphillPowerSpeed > 0 && P.UphillPowerGrade <= 0:
		P.UphillPowerGrade = c.GradeFromVelAndPower(P.UphillPowerSpeed, P.UphillPower)

	case P.UphillPowerGrade > 0 && P.UphillPowerSpeed <= 0:
		c.SetGrade(P.UphillPowerGrade)
		P.UphillPowerSpeed, _ = c.NewtonRaphson(P.UphillPower, 0, 3)

	default:
		return l.Err("uphillPowerSpeed <= 0 and uphillPowerGrade <= 0")
		// return l.Errorf(" ")
	}
	return nil
}

func flatPowerSpeedCdA(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {

	P := &p.Powermodel
	switch {
	case P.FlatPower > 0 && P.FlatSpeed > 0:
		if p.CdA <= 0 {
			p.CdA = c.CdAfromVelAndPower(P.FlatSpeed, P.FlatPower)
			if p.CdA < 0.01 {
				return l.Errorf("%s%4.3f",
					"Not enough power for rolling resistance and proper cdA. cdA =", p.CdA)
			}
			c.SetCdA(p.CdA)
		}
	case P.FlatPower > 0:
		P.FlatSpeed, _ = c.FlatSpeed(P.FlatPower)

	case P.FlatSpeed > 0:
		P.FlatPower = c.FlatPower(P.FlatSpeed)

	default:
		l.Err("flatSpeed <= 0 and flatPower <= 0")
		return l.Errorf(" ")
	}
	return nil
}

func maxPedalledSpeed(p *param.Parameters, c *motion.BikeCalc, l *logerr.Logerr) error {
	const minGrade = -10.0 / 100.0
	const defaultDownhillPower = 20.0
	const velDiff = 1.5 / 3.6
	P := &p.Powermodel

	if P.DownhillPower <= 0 {
		P.DownhillPower = defaultDownhillPower
	}
	dhpower := P.FlatPower * P.DownhillPower / 100
	if P.MaxPedalledSpeed <= 0 { // Now MaxPedalledSpeed > 0 must be given
		c.SetGrade(P.DownhillPowerGrade)
		vel, _ := c.NewtonRaphson(dhpower, 0, 8)
		P.MaxPedalledSpeed = vel + velDiff
		p.MinPedalledGrade = c.GradeFromVelAndPower(P.MaxPedalledSpeed, 0)
		return nil
	}
	p.MinPedalledGrade = c.GradeFromVelAndPower(P.MaxPedalledSpeed, 0)
	if p.MinPedalledGrade < minGrade {
		p.MinPedalledGrade = minGrade
		c.SetGrade(p.MinPedalledGrade)
		P.MaxPedalledSpeed = c.VelFreewheeling()
	}
	P.DownhillPowerGrade = c.GradeFromVelAndPower(P.MaxPedalledSpeed-velDiff, dhpower)
	return nil
}
