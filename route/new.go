package route

import (
	"math"

	"github.com/pekkizen/bikeride/gpx"
)

// New returns a Route struct with parsed latitude, longitude and elevation data from gpx.
// Only 1st track's 1st segment of gpx is used.
func New(gpx *gpx.GPX, p par) (*Route, error) {

	tps := gpx.TrkpSlice()
	points := len(tps)
	if p.Ride.RoundTrip {
		points *= 2
	}
	f := &p.Filter

	o := &Route{
		route:      make(route, points+1),
		trkpErrors: gpx.ErrCount(),
		metersLon:  metersLon(tps[0].Lat),
		metersLat:  metersLat(tps[0].Lat),

		filter: filter{
			minSegDist:       f.MinSegDist,
			maxAcceptedGrade: f.MaxAcceptedGrade,

			distFilterTol:  f.DistFilterTol,
			distFilterDist: f.DistFilterDist,

			ipoRounds:    f.IpoRounds,
			backsteps:    f.Backsteps,
			ipoDist:      f.IpoDist,
			ipoSumDist:   f.IpoSumDist,
			initRelgrade: f.InitialRelGrade,
			minRelGrade:  f.MinRelGrade,

			smoothingWeight: f.SmoothingWeight,
			smoothingDist:   f.SmoothingDist,

			levelFactor: f.LevelFactor,
			levelMax:    f.LevelMax,
			levelMin:    f.LevelMin,
		},
	}
	if points < 2 {
		return o, errNew("Only one track point")
	}
	if p.Ride.RoundTrip {
		tps = roundTrip(tps)

	} else if p.Ride.ReverseRoute {
		tps = gpx.TrkpSliceCopy() //don't change the original
		reverseTrack(tps)
	}
	o.importTrackPoints(tps)
	return o, nil
}

// NewCopy is for profiling only. Resets the route to the
// original route without memory allocation.
func (o *Route) NewCopy(p *Route) {
	copy(o.route, p.route)
	o.JouleRider = 0
	o.JriderTarget = 0
	o.TimeRider = 0
	o.TimeTarget = 0
	o.filter.ipolations = 0
	o.filter.levelations = 0
	o.filter.eleLeveled = 0
}

// reverseTrack reverses the order of the track points in a track point slice s.
func reverseTrack(s []gpx.Trkpt) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// roundTrip returns a copy of track s appended by reversed s.
func roundTrip(s []gpx.Trkpt) []gpx.Trkpt {
	l := len(s)
	q := make([]gpx.Trkpt, 2*l-1)
	copy(q, s)
	copy(q[l:], s[:l-1]) //drop the last/first point
	reverseTrack(q[l:])
	return q
}

func (o *Route) importTrackPoints(tps []gpx.Trkpt) {
	const minMinDist = 1.0
	var (
		distMean, dist   float64
		eleMean, latMean float64
		seg              = 0
		s                *segment
		minDist          = max(o.filter.minSegDist, minMinDist)
	)
	for _, p := range tps {
		if seg > 0 {
			dLon := (p.Lon - s.lon) * o.metersLon
			dLat := (p.Lat - s.lat) * o.metersLat
			dist = math.Sqrt(dLon*dLon + dLat*dLat)
			if dist < minDist {
				o.trkpRejected++
				continue
			}
		}
		// Road/route segments are indexed from 1 on. o.route[0]
		// is not used. n track points creates n-1 road segments.
		seg++
		s = &o.route[seg]
		s.segnum = seg
		s.lon = p.Lon
		s.lat = p.Lat
		s.ele = p.Ele
		s.eleGPX = p.Ele
		eleMean += p.Ele
		latMean += p.Lat
		distMean += dist
	}
	o.segments = seg - 1
	o.EleMean = eleMean / float64(seg)
	o.LatMean = latMean / float64(seg)
	o.distMean = distMean / float64(seg) // horisontal, not final, for median calc.
	o.route = o.route[: seg+1 : seg+1]   // clip excess capacity, do not remove/change because
	//                                   // len(o.route) = o.segments+2 is used later
}
