package mtr

import "math"

type hopAccumulator struct {
	Hop   int
	Host  string
	Sent  int
	Lost  int
	Times []float64
}

func (a *hopAccumulator) addSample(timeUs int64) {
	a.Sent++
	if timeUs < 0 {
		a.Lost++
		return
	}
	a.Times = append(a.Times, float64(timeUs)/1000.0)
}

func (a *hopAccumulator) toHop() Hop {
	loss := 0.0
	if a.Sent > 0 {
		loss = float64(a.Lost) / float64(a.Sent) * 100
	}

	h := Hop{
		Hop:         a.Hop,
		Host:        a.Host,
		LossPercent: loss,
		Sent:        a.Sent,
	}

	if len(a.Times) == 0 {
		return h
	}

	last := a.Times[len(a.Times)-1]
	sum, best, worst := 0.0, a.Times[0], a.Times[0]
	for _, t := range a.Times {
		sum += t
		if t < best {
			best = t
		}
		if t > worst {
			worst = t
		}
	}
	avg := sum / float64(len(a.Times))
	var variance float64
	for _, t := range a.Times {
		d := t - avg
		variance += d * d
	}
	stdev := 0.0
	if len(a.Times) > 1 {
		stdev = math.Sqrt(variance / float64(len(a.Times)-1))
	}

	h.LastMs = round1(last)
	h.AvgMs = round1(avg)
	h.BestMs = round1(best)
	h.WorstMs = round1(worst)
	h.StdevMs = round1(stdev)
	return h
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func sortedHops(states map[int]*hopAccumulator) []Hop {
	if len(states) == 0 {
		return nil
	}
	max := 0
	for k := range states {
		if k > max {
			max = k
		}
	}
	out := make([]Hop, 0, max+1)
	for i := 0; i <= max; i++ {
		if s, ok := states[i]; ok {
			out = append(out, s.toHop())
		}
	}
	return out
}
