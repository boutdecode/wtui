package sysstat

import "testing"

func TestParseCPU(t *testing.T) {
	idle, total, ok := parseCPU("cpu  100 0 50 800 50 0 0 0 0 0")
	if !ok || idle != 850 || total != 1000 {
		t.Fatalf("idle=%d total=%d ok=%v", idle, total, ok)
	}
	if _, _, ok := parseCPU("cpu0 1 2 3 4 5"); ok {
		t.Fatal("per-core line must be rejected")
	}
}

func TestParseMem(t *testing.T) {
	p, ok := parseMem("MemTotal:       1000 kB\nMemFree: 10 kB\nMemAvailable:    250 kB\n")
	if !ok || p != 75 {
		t.Fatalf("p=%v ok=%v", p, ok)
	}
}

func TestHistoryCapped(t *testing.T) {
	s := &Stats{n: 3}
	for i := 0; i < 5; i++ {
		s.push(float64(i*10), 50)
	}
	cpu, _ := s.Snapshot()
	if len(cpu) != 3 || cpu[0] != 20 || cpu[2] != 40 {
		t.Fatalf("cpu = %v", cpu)
	}
}
