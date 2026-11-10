// Package sysstat samples CPU and memory usage from /proc (Linux).
package sysstat

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Stats keeps the last N samples (percentages, 0-100) of CPU and RAM usage.
type Stats struct {
	mu       sync.Mutex
	n        int
	cpu, mem []float64
}

// Start samples every interval in a goroutine and calls tick after each
// sample. The returned Stats is safe for concurrent use.
func Start(n int, interval time.Duration, tick func()) *Stats {
	s := &Stats{n: n}
	go func() {
		prevIdle, prevTotal, _ := readCPU()
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			idle, total, ok := readCPU()
			var cpu float64
			if ok && total > prevTotal {
				cpu = 100 * (1 - float64(idle-prevIdle)/float64(total-prevTotal))
			}
			prevIdle, prevTotal = idle, total
			mem, _ := readMem()
			s.push(cpu, mem)
			tick()
		}
	}()
	return s
}

func (s *Stats) push(cpu, mem float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cpu = appendCap(s.cpu, clampPct(cpu), s.n)
	s.mem = appendCap(s.mem, clampPct(mem), s.n)
}

// Snapshot returns copies of the CPU and RAM histories, oldest first.
func (s *Stats) Snapshot() (cpu, mem []float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]float64(nil), s.cpu...), append([]float64(nil), s.mem...)
}

func appendCap(h []float64, v float64, n int) []float64 {
	h = append(h, v)
	if len(h) > n {
		h = h[len(h)-n:]
	}
	return h
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func readCPU() (idle, total uint64, ok bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return 0, 0, false
	}
	return parseCPU(sc.Text())
}

// parseCPU parses the aggregate "cpu ..." line of /proc/stat.
func parseCPU(line string) (idle, total uint64, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	// user nice system idle iowait irq softirq steal (guest is inside user)
	for i, f := range fields[1:] {
		if i >= 8 {
			break
		}
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		total += v
		if i == 3 || i == 4 {
			idle += v
		}
	}
	return idle, total, true
}

func readMem() (float64, bool) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	return parseMem(string(b))
}

// parseMem returns the used-memory percentage from /proc/meminfo contents.
func parseMem(s string) (float64, bool) {
	var total, avail float64
	var haveT, haveA bool
	for _, line := range strings.Split(s, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(f[1], 64)
		if err != nil {
			continue
		}
		switch f[0] {
		case "MemTotal:":
			total, haveT = v, true
		case "MemAvailable:":
			avail, haveA = v, true
		}
	}
	if !haveT || !haveA || total == 0 {
		return 0, false
	}
	return 100 * (total - avail) / total, true
}
