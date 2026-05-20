package mtr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// StreamEvent is one SSE message payload.
type StreamEvent struct {
	Type       string   `json:"type"`
	Progress   float64  `json:"progress,omitempty"`
	Cycles     int      `json:"cycles,omitempty"`
	Hop        *Hop     `json:"hop,omitempty"`
	Hops       []Hop    `json:"hops,omitempty"`
	Target     string   `json:"target,omitempty"`
	DurationMs int64    `json:"durationMs,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// StreamCallback receives events during MTR; return error to abort.
type StreamCallback func(StreamEvent) error

// RunStream runs mtr --raw and emits hop/progress updates until complete.
func RunStream(ctx context.Context, target string, opt Options, cb StreamCallback) error {
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
	if err := cb(StreamEvent{Type: "start", Target: target, Cycles: opt.Cycles}); err != nil {
		return err
	}

	cmd := opt.command(ctx,
		"--raw",
		"-n",
		"-c", strconv.Itoa(opt.Cycles),
		target,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mtr start: %w", err)
	}

	states := make(map[int]*hopAccumulator)
	probeCount := 0
	maxHop := 0
	expectedProbes := opt.Cycles // grows as hops appear

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "sudo:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "h":
			if len(fields) < 3 {
				continue
			}
			hop, _ := strconv.Atoi(fields[1])
			host := fields[2]
			if hop > maxHop {
				maxHop = hop
				expectedProbes = opt.Cycles * (maxHop + 1)
			}
			s, ok := states[hop]
			if !ok {
				s = &hopAccumulator{Hop: hop}
				states[hop] = s
			}
			s.Host = host
			h := s.toHop()
			if err := cb(StreamEvent{Type: "hop", Hop: &h, Target: target, Cycles: opt.Cycles}); err != nil {
				_ = cmd.Process.Kill()
				return err
			}

		case "p":
			if len(fields) < 4 {
				continue
			}
			hop, _ := strconv.Atoi(fields[1])
			timeUs, _ := strconv.ParseInt(fields[3], 10, 64)
			if hop > maxHop {
				maxHop = hop
				expectedProbes = opt.Cycles * (maxHop + 1)
			}
			s, ok := states[hop]
			if !ok {
				s = &hopAccumulator{Hop: hop, Host: "???"}
				states[hop] = s
			}
			s.addSample(timeUs)
			probeCount++

			progress := 0.0
			if expectedProbes > 0 {
				progress = float64(probeCount) / float64(expectedProbes)
				if progress > 0.99 {
					progress = 0.99
				}
			}

			h := s.toHop()
			if err := cb(StreamEvent{
				Type:     "hop",
				Progress: progress,
				Hop:      &h,
				Target:   target,
				Cycles:   opt.Cycles,
			}); err != nil {
				_ = cmd.Process.Kill()
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return fmt.Errorf("mtr read: %w", err)
	}

	waitErr := cmd.Wait()
	hops := sortedHops(states)
	if len(hops) == 0 {
		if waitErr != nil {
			return fmt.Errorf("mtr: %w", waitErr)
		}
		return fmt.Errorf("no hops in mtr stream")
	}

	durationMs := time.Since(start).Milliseconds()
	if waitErr != nil {
		_ = cb(StreamEvent{Type: "warn", Message: waitErr.Error(), Target: target})
	}

	return cb(StreamEvent{
		Type:       "done",
		Progress:   1,
		Target:     target,
		Cycles:     opt.Cycles,
		Hops:       hops,
		DurationMs: durationMs,
	})
}
