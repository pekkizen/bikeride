package main

import (
	"io"

	"github.com/pkg/profile"

	"bikeride/gpx"
	"bikeride/logerr"
	"motion"

	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
)

type devNull int

func (devNull) Close() error {
	return nil
}
func (devNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// if proFILE && len(os.Args) > 2 && os.Args[2] == "-prof" {
// 		cpuProfile(rou, gpxs, cal, gen, par, l, devNull(0))
// 	}

func cpuProfile(rou *route.Route, gpxs *gpx.GPX, cal *motion.BikeCalc,
	gen *power.Generator, par *param.Parameters, log *logerr.Logerr, writer io.WriteCloser) {
	if proFILE {
		prof := profile.Start(profile.ProfilePath("."))
		millions := 40
		rounds := millions * 1000 * 1000 / rou.Segments()
		log.Printf("Calculating %d million road segments. Wait... or press ctrl+c\n", millions)
		for i := 0; i < rounds; i++ {
			// gpxs,_ := gpx.New(par.GPXdir+par.GPXfile, par.GPXuseXMLparser, par.GPXignoreErrors)
			// _ = gpxs
			rou, _ := route.New(gpxs, par)
			rou.SetupRoad(par)
			rou.Filter()
			rou.SetupRide(cal, gen, par)
			rou.Ride(cal, par)
			// rou.UphillBreaks(par)
			res := rou.Results(cal, par)
			_ = res
			// res.WriteTXT(par, writer)
			// res.WriteJSON(par, writer)
			// rou.WriteCSV(par, writer)
		}
		prof.Stop()
	}
}
