package route

import (
	"bikeride/gpx"
	"math"
)

// New ---------
func New(gpx *gpx.GPX, p par) (*Route, error) {

	points := gpx.TrkpCount()
	o := &Route{
		route:           make(route, points+2),
		segments:        points - 1,
		limitTurnSpeeds: p.Ride.LimitTurnSpeeds,
		trkpErrors:      gpx.Errcnt,
		filter: filter{
			rounds:          p.Filter.Rounds,
			backsteps:       p.Filter.Backsteps,
			initRelgrade:    p.Filter.InitialRelGrade,
			minRelGrade:     p.Filter.MinRelGrade,
			maxFilteredEle:  p.Filter.MaxFilteredEle,
			maxIpolations:   p.Filter.MaxIpolations,
			levelFactor:     p.Filter.LevelFactor,
			levelMax:        p.Filter.LevelMax,
			levelMin:        p.Filter.LevelMin,
			ipoDist:         p.Filter.IpoDist,
			ipoSumDist:      p.Filter.IpoSumDist,
			movingAvgWeight: p.Filter.MovingAvgWeight,
			minSegDist:      p.Filter.MinSegDist,
		},
	}
	if points < 3 {
		return o, errf("Less than three track points")
	}
	o.importGPXdata(gpx)
	return o, nil
}

func (o *Route) importGPXdata(gpx *gpx.GPX) {

	var (
		mLat, mLon, eleMean     float64
		latMean, distMean, dist float64
		i                       int
		s                       *segment
	)
	for _, trk := range gpx.Trks {
		for _, seg := range trk.Trksegs {
			for _, point := range seg.Trkpts {

				if o.filter.minSegDist > 0 {
					if i == 0 {
						mLon = metersLon(point.Lat)
						mLat = metersLat(point.Lat)
					} else {
						dLon := (point.Lon - s.lon) * mLon
						dLat := (point.Lat - s.lat) * mLat
						dist = math.Sqrt(dLon*dLon + dLat*dLat)
						if dist < o.filter.minSegDist {
							o.segments--
							o.trkpRejected++
							continue
						}
						// s.distHor = dist // not here, mLat and mLon are not final
					}
				}
				i++
				s = &o.route[i]
				s.segnum = i
				s.lon = point.Lon
				s.lat = point.Lat
				s.ele = point.Ele
				s.eleGPX = point.Ele
				eleMean += point.Ele
				latMean += point.Lat
				distMean += dist
			}
		}
	}
	o.EleMean = eleMean / float64(i)
	o.LatMean = latMean / float64(i)
	o.distMean = distMean / float64(i) // horisontal here, not final
}
