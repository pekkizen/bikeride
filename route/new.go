package route

import (
	"bikeride/gpx"
	"math"
	// "fmt"
)

// New ---------
func New(gpx *gpx.GPX, p par) (*Route, error) {

	points := gpx.TrkpCount()
	// lenght := points + 2
	o := &Route{
		route:           make(route, points + 2),
		segments:        points - 1,
		limitTurnSpeeds: p.LimitTurnSpeeds,

		trkpRejected: 0,
		trkpErrors:   gpx.Errcnt,
		filter: filter{
			rounds: p.Filter.Rounds,
			// backstepRounds: p.Filter.BackstepRounds,
			backsteps:      p.Filter.Backsteps,
			initRelgrade:   p.Filter.InitialRelGrade,
			minRelGrade:    p.Filter.MinRelGrade,
			maxFilteredEle: p.Filter.MaxFilteredEle,
			levelFactor:    p.Filter.LevelFactor,
			levelMax:       p.Filter.LevelMax,
			levelMin:       p.Filter.LevelMin,
			ipoDist:        p.Filter.IpoDist,
			ipoSumDist:     p.Filter.IpoSumDist,
			backstepDist:   p.Filter.BackstepDist,
			minSegDist:     p.Filter.MinSegDist,
		},
	}
	if points < 3 {
		return o, errf("Less than three track points")
	}
	o.fillGPXData(gpx)

	if o.segments < 2 {
		return o, errf("Less than two road segments")
	}
	return o, nil
}

func (o *Route) fillGPXData(gpx *gpx.GPX) {

	var mLat, mLon, eleMean, latMean, distMean float64
	i, k := 0, 0
	s := &o.route[1]
	points := gpx.TrkpCount()
	minSegDist := o.filter.minSegDist

	for _, trk := range gpx.Trks {
		for _, seg := range trk.Trksegs {
			for _, point := range seg.Trkpts {
				k++

				if i == 0 {
					s.lon = point.Lon
					s.lat = point.Lat
					s.ele = point.Ele
					s.eleGPX = point.Ele
					s.segnum = 1
					mLon = metersLon(point.Lat)
					mLat = metersLat(point.Lat)
					i = 1
					continue
				}
				dLon := (point.Lon - s.lon) * mLon
				dLat := (point.Lat - s.lat) * mLat
				dist := math.Sqrt(dLon*dLon + dLat*dLat)

				if dist < minSegDist && k < points { //Don't drop the last one
					o.segments--
					o.trkpRejected++
					continue
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
	o.distMean = distMean / float64(i)  // horisontal here
}
