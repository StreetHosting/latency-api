package mtr_test

import (
	"testing"

	"github.com/streethosting/latency-api/internal/mtr"
)

const sampleJSON = `{
  "report": {
    "mtr": {
      "dst": "8.8.8.8",
      "hubs": {
        "hub": [
          {
            "count": "1",
            "host": "10.0.0.1",
            "Loss%": 0,
            "Snt": 10,
            "Last": 1.1,
            "Avg": 1.2,
            "Best": 1.0,
            "Wrst": 1.5,
            "StDev": 0.1
          },
          {
            "count": "2",
            "host": "8.8.8.8",
            "Loss%": 0,
            "Snt": 10,
            "Last": 5.0,
            "Avg": 5.1,
            "Best": 4.9,
            "Wrst": 5.5,
            "StDev": 0.2
          }
        ]
      }
    }
  }
}`

func TestParseMTRJSON(t *testing.T) {
	hops, err := mtr.ParseOutputForTest([]byte(sampleJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 2 {
		t.Fatalf("hops = %d, want 2", len(hops))
	}
	if hops[0].Host != "10.0.0.1" || hops[1].Hop != 2 {
		t.Fatalf("unexpected hops: %+v", hops)
	}
}

const sampleReport = `Start: 2026-05-19T12:00:00+0000
HOST: probe                 Loss%   Snt   Last   Avg  Best  Wrst StDev
  1.|-- 10.0.0.1             0.0%    10    1.0   1.1   0.9   1.3   0.1
  2.|-- 8.8.8.8              0.0%    10    5.0   5.1   4.9   5.5   0.2
`

func TestParseReportText(t *testing.T) {
	hops, err := mtr.ParseOutputForTest([]byte(sampleReport))
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 2 {
		t.Fatalf("hops = %d, want 2", len(hops))
	}
}
