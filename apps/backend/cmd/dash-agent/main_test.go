package main

import "testing"

func TestContainerCPUPct(t *testing.T) {
	var s dockerStatsJSON
	s.CPUStats.CPUUsage.TotalUsage = 300
	s.PreCPUStats.CPUUsage.TotalUsage = 100
	s.CPUStats.SystemCPUUsage = 1000
	s.PreCPUStats.SystemCPUUsage = 500
	s.CPUStats.OnlineCPUs = 4
	if got := containerCPUPct(&s); got != 160 {
		t.Fatalf("expected 160%%, got %v", got)
	}
	// No deltas -> 0, no div-by-zero.
	s.PreCPUStats.CPUUsage.TotalUsage = s.CPUStats.CPUUsage.TotalUsage
	s.PreCPUStats.SystemCPUUsage = s.CPUStats.SystemCPUUsage
	if got := containerCPUPct(&s); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestParseZpoolText(t *testing.T) {
	out := parseZpoolText([]byte("tank\tONLINE\t1000000000\t400000000\nfast\tDEGRADED\t500\t100\n"))
	if len(out) != 2 || out[0].Name != "tank" || out[0].Size != 1e9 || out[1].Health != "DEGRADED" {
		t.Fatalf("bad parse: %+v", out)
	}
	if parseZpoolText(nil) != nil {
		t.Fatal("nil input should yield nil")
	}
}

func TestParseZpoolJSON(t *testing.T) {
	b := []byte(`{"output_version":{"major":0},"pools":{"tank":{"name":"tank","properties":{
		"health":{"value":"ONLINE"},"size":{"value":1000000000},"free":{"value":250000000}}}}}`)
	out := parseZpoolJSON(b)
	if len(out) != 1 || out[0].Health != "ONLINE" || out[0].Free != 250000000 {
		t.Fatalf("bad parse: %+v", out)
	}
}

func TestParseNvidiaCSV(t *testing.T) {
	out := parseNvidiaCSV([]byte("RTX 3080, 62, 14, 2048, 10240\n"))
	if len(out) != 1 || out[0].Name != "RTX 3080" || out[0].TempC != 62 ||
		out[0].MemUsed != 2048*1<<20 || out[0].MemTotal != 10240*1<<20 {
		t.Fatalf("bad parse: %+v", out)
	}
}
