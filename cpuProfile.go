package main

import (
	"github.com/pkg/profile"

	"github.com/pekkizen/bikeride/gpx"
	"github.com/pekkizen/bikeride/logerr"
	"github.com/pekkizen/bikeride/param"
	"github.com/pekkizen/bikeride/power"
	"github.com/pekkizen/bikeride/route"

	"github.com/pekkizen/motion"
)

type devNull int

func (devNull) Close() error {
	return nil
}
func (devNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}
func cpuProfile(gpz *gpx.GPX, cal *motion.BikeCalc,
	gen *power.Generator, p *param.Parameters, l *logerr.Logerr) *route.Results {
	_ = devNull(0)

	millions := 50
	rou, _ := route.New(gpz, p)
	rounds := millions * 1000 * 1000 / rou.Segments()
	l.Printf("Calculating %d million road segments. Wait... or press ctrl+c\n", millions)
	// rou0, _ := route.New(gpz, p)
	var res *route.Results
	prof := profile.Start(profile.ProfilePath("."))
	for i := 0; i < rounds; i++ {
		gpz, _ := gpx.New(p.GPXdir+p.GPXfile, p.GPXuseXMLparser, p.GPXignoreErrors)
		rou, _ = route.New(gpz, p)
		// rou.NewCopy(rou0) // to avoid garbage collection timing
		rou.SetupRoad(p)
		rou.Filter()
		rou.SetupRide(cal, gen, p)
		rou.Ride(cal, p)
		// rou.UphillBreaks(p)
		res = rou.Results(cal, p, l)
		_ = res
		res.WriteTXT(p, devNull(0))
		// res.WriteJSON(devNull(0))
		rou.WriteCSV(p, devNull(0))
	}
	prof.Stop()
	return res
}
