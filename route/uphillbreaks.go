package route

//UphillBreaks --
func (o *Route) UphillBreaks(p par) {

	U := p.UphillBreak
	if U.ClimbDuration <= 0 || U.BreakDuration <= 0 || U.PowerLimit <= 0 {
		return
	}
	var (
		time, joule float64
		powerlim    = U.PowerLimit * p.Powermodel.UphillPower
		r           = o.route
		left, right = 1, 1
		last        = o.segments - o.segments/50
	)
	for right <= last {
		for time < U.ClimbDuration && right <= last {
			time += r[right].time
			joule += r[right].jouleRider
			right++
		}
		if joule > time*powerlim {
			r[right].timeBreak = U.BreakDuration
			time, joule = 0, 0
			right++
			left = right
			continue
		}
		for left < right && (joule < 0.95*time*powerlim || time > U.ClimbDuration) {
			time -= r[left].time
			joule -= r[left].jouleRider
			left++
		}
	}
}
