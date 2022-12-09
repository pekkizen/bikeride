package route

//UphillBreaks --
func (o *Route) UphillBreaks(p par) {

	u := p.UphillBreak
	if u.ClimbDuration <= 0 || u.BreakDuration <= 0 || u.PowerLimit <= 0 {
		return
	}
	var (
		time, joule float64
		powerlim    = u.PowerLimit * p.Powermodel.UphillPower
		r           = o.route
		left, right = 1, 1
		last        = o.segments - o.segments/50
	)
	for right <= last {
		for time < u.ClimbDuration && right <= last {
			time += r[right].time
			joule += r[right].jouleRider
			right++
		}
		if joule >= time*powerlim {
			r[right].timeBreak = u.BreakDuration
			time, joule = 0, 0
			right++
			left = right
			continue
		}
		for left < right && (joule < 0.95*time*powerlim || time > u.ClimbDuration) {
			time -= r[left].time
			joule -= r[left].jouleRider
			left++
		}
	}
}
