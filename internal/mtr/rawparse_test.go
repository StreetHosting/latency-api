package mtr

import (
	"strings"
	"testing"
)

func TestSortedHopsFromRawAccumulator(t *testing.T) {
	states := map[int]*hopAccumulator{
		0: {Hop: 0, Host: "10.0.0.1"},
		1: {Hop: 1, Host: "1.1.1.1"},
	}
	states[0].addSample(1000)
	states[0].addSample(1200)
	states[1].addSample(5000)

	hops := sortedHops(states)
	if len(hops) != 2 {
		t.Fatalf("len=%d", len(hops))
	}
	if hops[0].AvgMs <= 0 || hops[1].Host != "1.1.1.1" {
		t.Fatalf("%+v", hops)
	}
}

func TestSanitizeOutput(t *testing.T) {
	out := sanitizeOutput([]byte("sudo: warn\nh 0 1.2.3.4\n"))
	if strings.Contains(string(out), "sudo:") {
		t.Fatal("sudo line should be stripped")
	}
}
