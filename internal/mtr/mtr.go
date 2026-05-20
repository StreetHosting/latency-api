package mtr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Hop is one row in the MTR report.
type Hop struct {
	Hop         int     `json:"hop"`
	Host        string  `json:"host"`
	LossPercent float64 `json:"lossPercent"`
	Sent        int     `json:"sent"`
	LastMs      float64 `json:"lastMs"`
	AvgMs       float64 `json:"avgMs"`
	BestMs      float64 `json:"bestMs"`
	WorstMs     float64 `json:"worstMs"`
	StdevMs     float64 `json:"stdevMs"`
}

// Report is returned by GET /mtr.
type Report struct {
	Target     string `json:"target"`
	Cycles     int    `json:"cycles"`
	DurationMs int64  `json:"durationMs"`
	Hops       []Hop  `json:"hops"`
}

// Options configures an MTR run.
type Options struct {
	Binary  string
	Cycles  int
	Timeout time.Duration
	UseSudo bool
}

// Matches mtr -r lines (mtr-tiny 0.95), e.g. "  1.|-- 82.38.28.254  0.0%  3 ..."
var reportLine = regexp.MustCompile(`^\s*(\d+)\.\|\s*--\s+(\S+)\s+([\d.]+%?)\s+(\d+)\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)`)

// Run executes mtr toward target and returns parsed hops.
func Run(ctx context.Context, target string, opt Options) (*Report, error) {
	if opt.Binary == "" {
		opt.Binary = "/usr/bin/mtr"
	}
	if opt.Cycles <= 0 {
		opt.Cycles = 10
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 45 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	start := time.Now()

	// Prefer -r (report): stable on mtr-tiny; --json varies by build and often breaks parsing.
	try := []func(context.Context, Options, string) ([]byte, error){
		runReport,
		runJSON,
	}

	var hops []Hop
	var lastErr error
	for _, run := range try {
		out, err := run(ctx, opt, target)
		if err != nil {
			lastErr = err
			continue
		}
		hops, err = parseOutput(sanitizeOutput(out))
		if err == nil && len(hops) > 0 {
			return &Report{
				Target:     target,
				Cycles:     opt.Cycles,
				DurationMs: time.Since(start).Milliseconds(),
				Hops:       hops,
			}, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no hops in mtr output")
	}
	return nil, lastErr
}

func sanitizeOutput(raw []byte) []byte {
	var b strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "sudo:") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func (opt Options) command(ctx context.Context, args ...string) *exec.Cmd {
	if opt.UseSudo {
		sudoArgs := append([]string{"-n", opt.Binary}, args...)
		return exec.CommandContext(ctx, "sudo", sudoArgs...)
	}
	return exec.CommandContext(ctx, opt.Binary, args...)
}

func runJSON(ctx context.Context, opt Options, target string) ([]byte, error) {
	return runCommand(ctx, opt.command(ctx,
		"--json",
		"--no-dns",
		"-c", strconv.Itoa(opt.Cycles),
		"-n",
		target,
	))
}

func runReport(ctx context.Context, opt Options, target string) ([]byte, error) {
	return runCommand(ctx, opt.command(ctx,
		"-r",
		"--no-dns",
		"-c", strconv.Itoa(opt.Cycles),
		"-n",
		target,
	))
}

func runCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, fmt.Errorf("%w", err)
		}
		return nil, fmt.Errorf("%w: %s", err, msg)
	}
	return out, nil
}

// ParseOutputForTest exposes output parsing for unit tests.
func ParseOutputForTest(raw []byte) ([]Hop, error) {
	return parseOutput(raw)
}

func parseOutput(raw []byte) ([]Hop, error) {
	if hops, err := parseMTRJSON(raw); err == nil && len(hops) > 0 {
		return hops, nil
	}
	return parseReportText(string(raw))
}

func parseMTRJSON(raw []byte) ([]Hop, error) {
	var doc struct {
		Report struct {
			MTR struct {
				Hubs struct {
					Hub json.RawMessage `json:"hub"`
				} `json:"hubs"`
			} `json:"mtr"`
		} `json:"report"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if len(doc.Report.MTR.Hubs.Hub) == 0 {
		return nil, fmt.Errorf("empty hubs")
	}

	var one hubJSON
	if err := json.Unmarshal(doc.Report.MTR.Hubs.Hub, &one); err == nil && one.count() > 0 {
		return []Hop{hubToHop(one)}, nil
	}

	var many []hubJSON
	if err := json.Unmarshal(doc.Report.MTR.Hubs.Hub, &many); err != nil {
		return nil, err
	}
	hops := make([]Hop, 0, len(many))
	for _, h := range many {
		hops = append(hops, hubToHop(h))
	}
	return hops, nil
}

type hubJSON struct {
	Count  jsonField `json:"count"`
	Host   jsonField `json:"host"`
	Loss   jsonField `json:"Loss%"`
	Snt    jsonField `json:"Snt"`
	Last   jsonField `json:"Last"`
	Avg    jsonField `json:"Avg"`
	Best   jsonField `json:"Best"`
	Wrst   jsonField `json:"Wrst"`
	Stdev  jsonField `json:"StDev"`
	CountA jsonField `json:"@count"`
	HostA  jsonField `json:"@host"`
	LossA  jsonField `json:"@Loss%"`
	SntA   jsonField `json:"@Snt"`
	LastA  jsonField `json:"@Last"`
	AvgA   jsonField `json:"@Avg"`
	BestA  jsonField `json:"@Best"`
	WrstA  jsonField `json:"@Wrst"`
	StdevA jsonField `json:"@StDev"`
}

type jsonField string

func (h hubJSON) count() int {
	if v := h.Count.pick(); v != "" {
		return atoi(v)
	}
	return atoi(h.CountA.pick())
}

func (h hubJSON) host() string {
	if v := h.Host.pick(); v != "" {
		return v
	}
	return h.HostA.pick()
}

func (h hubJSON) loss() float64 {
	if v := h.Loss.pick(); v != "" {
		return atof(v)
	}
	return atof(h.LossA.pick())
}

func (h hubJSON) snt() int {
	if v := h.Snt.pick(); v != "" {
		return atoi(v)
	}
	return atoi(h.SntA.pick())
}

func (h hubJSON) last() float64 {
	if v := h.Last.pick(); v != "" {
		return atof(v)
	}
	return atof(h.LastA.pick())
}

func (h hubJSON) avg() float64 {
	if v := h.Avg.pick(); v != "" {
		return atof(v)
	}
	return atof(h.AvgA.pick())
}

func (h hubJSON) best() float64 {
	if v := h.Best.pick(); v != "" {
		return atof(v)
	}
	return atof(h.BestA.pick())
}

func (h hubJSON) wrst() float64 {
	if v := h.Wrst.pick(); v != "" {
		return atof(v)
	}
	return atof(h.WrstA.pick())
}

func (h hubJSON) stdev() float64 {
	if v := h.Stdev.pick(); v != "" {
		return atof(v)
	}
	return atof(h.StdevA.pick())
}

func (f jsonField) pick() string {
	return strings.TrimSpace(string(f))
}

func (f *jsonField) UnmarshalJSON(b []byte) error {
	// mtr may emit numbers or strings
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = jsonField(s)
		return nil
	}
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = jsonField(strconv.FormatFloat(n, 'f', -1, 64))
		return nil
	}
	*f = jsonField(strings.Trim(string(b), `"`))
	return nil
}

func hubToHop(h hubJSON) Hop {
	return Hop{
		Hop:         h.count(),
		Host:        h.host(),
		LossPercent: h.loss(),
		Sent:        h.snt(),
		LastMs:      h.last(),
		AvgMs:       h.avg(),
		BestMs:      h.best(),
		WorstMs:     h.wrst(),
		StdevMs:     h.stdev(),
	}
}

func parseReportText(text string) ([]Hop, error) {
	var hops []Hop
	for _, line := range strings.Split(text, "\n") {
		m := reportLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		loss := m[3]
		if !strings.HasSuffix(loss, "%") {
			loss += "%"
		}
		hops = append(hops, Hop{
			Hop:         atoi(m[1]),
			Host:        m[2],
			LossPercent: atof(strings.TrimSuffix(loss, "%")),
			Sent:        atoi(m[4]),
			LastMs:      atof(m[5]),
			AvgMs:       atof(m[6]),
			BestMs:      atof(m[7]),
			WorstMs:     atof(m[8]),
			StdevMs:     atof(m[9]),
		})
	}
	if len(hops) == 0 {
		return nil, fmt.Errorf("no hops in report output")
	}
	return hops, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
