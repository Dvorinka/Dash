package main

// Docker collector: container liveness plus per-container CPU/memory from
// /containers/{id}/stats?stream=false. All best-effort — no socket, no data.

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const dockerSock = "/var/run/docker.sock"

var dockerClient = &http.Client{
	Timeout: 4 * time.Second,
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", dockerSock)
		},
	},
}

// dockerStatsJSON is the subset of /containers/{id}/stats we consume.
type dockerStatsJSON struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
}

// containerCPUPct applies the Docker dashboard formula: cpu delta over
// system delta, scaled by online CPUs. 0 when deltas don't advance.
func containerCPUPct(s *dockerStatsJSON) float64 {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemCPUUsage) - float64(s.PreCPUStats.SystemCPUUsage)
	online := s.CPUStats.OnlineCPUs
	if online == 0 {
		online = 1
	}
	if cpuDelta <= 0 || sysDelta <= 0 {
		return 0
	}
	return cpuDelta / sysDelta * float64(online) * 100
}

// readContainers lists containers and fans out to per-container stats.
// Stats fetches are bounded: 8 concurrent, each under the client timeout.
func readContainers() []containerStat {
	if _, err := net.DialTimeout("unix", dockerSock, time.Second); err != nil {
		return nil
	}
	res, err := dockerClient.Get("http://d/containers/json?all=1")
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	var raw []struct {
		ID    string   `json:"Id"`
		Names []string `json:"Names"`
		State string   `json:"State"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&raw) != nil {
		return nil
	}
	out := make([]containerStat, len(raw))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, r := range raw {
		name := ""
		if len(r.Names) > 0 {
			name = strings.TrimPrefix(r.Names[0], "/")
		}
		out[i] = containerStat{Name: name, State: r.State}
		if r.State != "running" || r.ID == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, id string) {
			defer wg.Done()
			defer func() { <-sem }()
			fillContainerStats(&out[i], id)
		}(i, r.ID)
	}
	wg.Wait()
	return out
}

// fillContainerStats fetches one container's one-shot stats into cs.
func fillContainerStats(cs *containerStat, id string) {
	res, err := dockerClient.Get("http://d/containers/" + id + "/stats?stream=false")
	if err != nil {
		return
	}
	defer res.Body.Close()
	var st dockerStatsJSON
	if json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&st) != nil {
		return
	}
	cs.CPU = containerCPUPct(&st)
	cs.MemUsed = float64(st.MemoryStats.Usage)
	cs.MemLimit = float64(st.MemoryStats.Limit)
}
