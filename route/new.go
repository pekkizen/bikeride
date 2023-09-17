package route

import (
	"bikeride/gpx"
	"math"
)

// New ---------
func New(gpx *gpx.GPX, p par) (*Route, error) {

	f := &p.Filter
	points := gpx.TrkpCount()
	o := &Route{
		route:           make(route, points+2),
		segments:        points - 1,
		limitTurnSpeeds: p.Ride.LimitTurnSpeeds,
		trkpErrors:      gpx.ErrCount(),
		filter: filter{
			ipoRounds:        f.IpoRounds,
			backsteps:        f.Backsteps,
			initRelgrade:     f.InitialRelGrade,
			minRelGrade:      f.MinRelGrade,
			levelFactor:      f.LevelFactor,
			levelMax:         f.LevelMax,
			levelMin:         f.LevelMin,
			ipoDist:          f.IpoDist,
			ipoSumDist:       f.IpoSumDist,
			smoothingWeight:  f.SmoothingWeight,
			smoothingDist:    f.SmoothingDist,
			maxAcceptedGrade: f.MaxAcceptedGrade,
			distFilterTol:    f.DistFilterTol,
			distFilterDist:   f.DistFilterDist,
			minSegDist:       f.MinSegDist,
		},
	}
	if points < 3 {
		return o, errNew("Less than three track points")
	}
	o.importGPXdata(gpx)
	o.route = o.route[0 : o.segments+2]
	return o, nil
}

func (o *Route) importGPXdata(gpx *gpx.GPX) {

	var (
		mLat, mLon, eleMean     float64
		latMean, distMean, dist float64
		i                       int
		r                       = o.route
		s                       *segment
	)
	for _, trk := range gpx.Trks {
		for _, seg := range trk.Trksegs {
			for _, point := range seg.Trkpts {
				if i == 0 {
					mLon = metersLon(point.Lat)
					mLat = metersLat(point.Lat)
				} else {
					dLon := (point.Lon - s.lon) * mLon
					dLat := (point.Lat - s.lat) * mLat
					dist = math.Sqrt(dLon*dLon + dLat*dLat)
					if dist <= o.filter.minSegDist {
						o.trkpRejected++
						continue
					}
				}
				i++
				s = &r[i]
				s.segnum = i
				lat := point.Lat
				ele := point.Ele
				s.lon = point.Lon
				s.lat = lat
				s.ele = ele
				s.eleGPX = ele
				eleMean += ele
				latMean += lat
				distMean += dist
			}
		}
	}
	o.segments -= o.trkpRejected
	o.EleMean = eleMean / float64(i)
	o.LatMean = latMean / float64(i)
	o.distMean = distMean / float64(i) // horisontal here, not final
}
