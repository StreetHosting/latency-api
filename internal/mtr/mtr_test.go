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

const sampleReportGames = `sudo: unable to resolve host latency-sp-games-01: Name or service not known
Start: 2026-05-20T10:07:03-0300
HOST: latency-sp-games-01         Loss%   Snt   Last   Avg  Best  Wrst StDev
  1.|-- 82.38.28.254               0.0%     3    1.4   3.8   1.4   7.5   3.2
  7.|-- ???                       100.0     3    0.0   0.0   0.0   0.0   0.0
  9.|-- 1.1.1.1                    0.0%     3    1.6   1.6   1.6   1.6   0.0
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

func TestParseReportText_ProductionFormat(t *testing.T) {
	hops, err := mtr.ParseOutputForTest([]byte(sampleReportGames))
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 3 {
		t.Fatalf("hops = %d, want 3", len(hops))
	}
	if hops[2].Host != "1.1.1.1" {
		t.Fatalf("last hop = %+v", hops[2])
	}
}
