package main

// Optional collectors: SMART (smartctl), ZFS (zpool), GPU (nvidia-smi or
// /sys/class/drm). Every collector is exec-or-skip — missing tools or
// permissions produce nil, never errors, and each exec carries a timeout so a
// wedged tool can't stall reporting.

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// runOut executes a command with a timeout; empty output on any failure.
func runOut(timeout time.Duration, name string, args ...string) []byte {
	if _, err := exec.LookPath(name); err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return nil
	}
	return out
}

// ---------- SMART ----------

type smartctlScan struct {
	Devices []struct {
		Name string `json:"name"`
	} `json:"devices"`
}

type smartctlAll struct {
	ModelName    string `json:"model_name"`
	SmartStatus  *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature *struct {
		Current float64 `json:"current"`
	} `json:"temperature"`
}

// readSMART scans devices via smartctl -j --scan, then reads each drive.
// Needs smartmontools installed and usually root or sudo rights for /dev.
func readSMART() []smartDisk {
	scan := runOut(5*time.Second, "smartctl", "-j", "--scan")
	if scan == nil {
		return nil
	}
	var sc smartctlScan
	if json.Unmarshal(scan, &sc) != nil || len(sc.Devices) == 0 {
		return nil
	}
	out := make([]smartDisk, 0, len(sc.Devices))
	for i, d := range sc.Devices {
		if i >= 8 { // bound the fan-out
			break
		}
		b := runOut(4*time.Second, "smartctl", "-j", "-a", d.Name)
		if b == nil {
			continue
		}
		var a smartctlAll
		if json.Unmarshal(b, &a) != nil {
			continue
		}
		sd := smartDisk{Device: d.Name, Model: a.ModelName}
		if a.SmartStatus != nil {
			p := a.SmartStatus.Passed
			sd.Passed = &p
		}
		if a.Temperature != nil {
			sd.TempC = a.Temperature.Current
		}
		out = append(out, sd)
	}
	return out
}

// ---------- ZFS ----------

// readZFS prefers `zpool list -j` (OpenZFS 2.3+), falls back to the stable
// `-H -p` text format. Parse failures degrade to nil.
func readZFS() []zfsPool {
	if b := runOut(4*time.Second, "zpool", "list", "-j", "--json-int",
		"-o", "name,health,size,free"); b != nil {
		if pools := parseZpoolJSON(b); pools != nil {
			return pools
		}
	}
	return parseZpoolText(runOut(4*time.Second,
		"zpool", "list", "-H", "-p", "-o", "name,health,size,free"))
}

// parseZpoolJSON reads {"pools":{"name":{"properties":{...}}}}.
func parseZpoolJSON(b []byte) []zfsPool {
	var doc struct {
		Pools map[string]struct {
			Properties map[string]struct {
				Value any `json:"value"`
			} `json:"properties"`
		} `json:"pools"`
	}
	if json.Unmarshal(b, &doc) != nil || len(doc.Pools) == 0 {
		return nil
	}
	var out []zfsPool
	num := func(v any) float64 {
		switch n := v.(type) {
		case float64:
			return n
		case string:
			f, _ := strconv.ParseFloat(n, 64)
			return f
		}
		return 0
	}
	for name, p := range doc.Pools {
		z := zfsPool{Name: name}
		if v, ok := p.Properties["health"]; ok {
			if s, ok := v.Value.(string); ok {
				z.Health = s
			}
		}
		if v, ok := p.Properties["size"]; ok {
			z.Size = num(v.Value)
		}
		if v, ok := p.Properties["free"]; ok {
			z.Free = num(v.Value)
		}
		out = append(out, z)
	}
	return out
}

// parseZpoolText reads `zpool list -H -p` rows: name\thealth\tsize\tfree.
func parseZpoolText(b []byte) []zfsPool {
	var out []zfsPool
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		size, _ := strconv.ParseFloat(f[2], 64)
		free, _ := strconv.ParseFloat(f[3], 64)
		out = append(out, zfsPool{Name: f[0], Health: f[1], Size: size, Free: free})
	}
	return out
}

// ---------- GPU ----------

// readGPU prefers nvidia-smi (CSV query); otherwise walks /sys/class/drm for
// AMD/Intel cards exposing gpu_busy_percent and hwmon temps.
func readGPU() []gpuStat {
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return readNvidiaGPU()
	}
	return readGPUSysfs()
}

func readNvidiaGPU() []gpuStat {
	return parseNvidiaCSV(runOut(4*time.Second, "nvidia-smi",
		"--query-gpu=name,temperature.gpu,utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits"))
}

// parseNvidiaCSV reads rows: name, tempC, util%, memUsed MiB, memTotal MiB.
func parseNvidiaCSV(b []byte) []gpuStat {
	var out []gpuStat
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(line, ",")
		if len(f) < 5 {
			continue
		}
		g := gpuStat{Name: strings.TrimSpace(f[0])}
		g.TempC, _ = strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
		g.UtilPct, _ = strconv.ParseFloat(strings.TrimSpace(f[2]), 64)
		g.MemUsed, _ = strconv.ParseFloat(strings.TrimSpace(f[3]), 64)
		g.MemTotal, _ = strconv.ParseFloat(strings.TrimSpace(f[4]), 64)
		g.MemUsed *= 1 << 20
		g.MemTotal *= 1 << 20
		out = append(out, g)
	}
	return out
}

// readGPUSysfs pulls util/temp for each card from DRM sysfs.
func readGPUSysfs() []gpuStat {
	cards, _ := filepath.Glob("/sys/class/drm/card[0-9]")
	var out []gpuStat
	for _, card := range cards {
		dev := card + "/device"
		g := gpuStat{Name: filepath.Base(card)}
		if b, err := os.ReadFile(dev + "/gpu_busy_percent"); err == nil {
			g.UtilPct, _ = strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
		}
		// hwmon: first temp1_input under device/hwmon/hwmonN.
		if hws, err := filepath.Glob(dev + "/hwmon/hwmon*"); err == nil {
			for _, hw := range hws {
				if b, err := os.ReadFile(hw + "/temp1_input"); err == nil {
					if milli, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); err == nil {
						g.TempC = milli / 1000
					}
					break
				}
			}
		}
		// Driver name via uevent DRIVER= for a friendlier label.
		if b, err := os.ReadFile(dev + "/uevent"); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if v, ok := strings.CutPrefix(line, "DRIVER="); ok {
					g.Name += " (" + v + ")"
				}
			}
		}
		if g.UtilPct > 0 || g.TempC > 0 {
			out = append(out, g)
		}
	}
	return out
}
