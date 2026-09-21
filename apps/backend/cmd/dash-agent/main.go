// Command dash-agent collects host metrics and pushes them to a Dash server.
// Linux-first: /proc for cpu/mem/net/load/uptime, /sys hwmon for temps,
// statfs for disk, and the Docker socket for container liveness when present.
//
//	env DASH_URL=http://dash:8080 DASH_TOKEN=agent_xxx dash-agent
//
// Flags override env: -url, -token, -interval (seconds), -once.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	url := flag.String("url", os.Getenv("DASH_URL"), "dash server base URL")
	token := flag.String("token", os.Getenv("DASH_TOKEN"), "system token")
	interval := flag.Int("interval", envInt("DASH_INTERVAL", 10), "push interval seconds")
	once := flag.Bool("once", false, "collect and print one sample, no push")
	flag.Parse()

	c := &collector{}
	c.host, _ = os.Hostname()
	c.os = runtime.GOOS
	c.arch = runtime.GOARCH
	c.cpuModel, c.cores = cpuInfo()
	c.prevCPU = readCPUTicks()
	c.prevNet = readNetBytes()
	c.prevAt = time.Now()

	if *once {
		// Rates need a delta window — idle briefly so cpu/net are real.
		time.Sleep(500 * time.Millisecond)
		s := c.sample(*interval)
		out, _ := json.MarshalIndent(s, "", "  ")
		fmt.Println(string(out))
		return
	}
	if *url == "" || *token == "" {
		log.Fatal("dash-agent: -url and -token (or DASH_URL/DASH_TOKEN) required")
	}
	log.Printf("dash-agent: pushing to %s every %ds", *url, *interval)
	for {
		s := c.sample(*interval)
		if err := push(*url, *token, s); err != nil {
			log.Printf("dash-agent: push: %v", err)
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}

// sample mirrors api.StatSample — duplicated here so the agent stays a
// standalone binary with zero imports from the server module.
type sample struct {
	Host       string             `json:"host"`
	OS         string             `json:"os"`
	Arch       string             `json:"arch"`
	CPUModel   string             `json:"cpuModel"`
	Cores      int                `json:"cores"`
	IntervalS  int                `json:"intervalS"`
	UptimeS    float64            `json:"uptimeS"`
	CPU        float64            `json:"cpu"`
	MemTotal   float64            `json:"memTotal"`
	MemUsed    float64            `json:"memUsed"`
	SwapTotal  float64            `json:"swapTotal"`
	SwapUsed   float64            `json:"swapUsed"`
	DiskTotal  float64            `json:"diskTotal"`
	DiskUsed   float64            `json:"diskUsed"`
	NetRx      float64            `json:"netRx"`
	NetTx      float64            `json:"netTx"`
	Load1      float64            `json:"load1"`
	Load5      float64            `json:"load5"`
	Load15     float64            `json:"load15"`
	Temps      map[string]float64 `json:"temps,omitempty"`
	Containers []containerStat    `json:"containers,omitempty"`
}

type containerStat struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type collector struct {
	host, os, arch, cpuModel string
	cores                    int
	prevCPU                  cpuTicks
	prevNet                  [2]uint64 // rx, tx bytes
	prevAt                   time.Time
}

type cpuTicks struct{ idle, total uint64 }

func (c *collector) sample(interval int) sample {
	s := sample{
		Host: c.host, OS: c.os, Arch: c.arch,
		CPUModel: c.cpuModel, Cores: c.cores, IntervalS: interval,
	}
	now := time.Now()
	dt := now.Sub(c.prevAt).Seconds()
	if dt <= 0 {
		dt = float64(interval)
	}

	if cur := readCPUTicks(); cur.total > c.prevCPU.total {
		busy := float64(cur.total-cur.idle) - float64(c.prevCPU.total-c.prevCPU.idle)
		s.CPU = busy / float64(cur.total-c.prevCPU.total) * 100
		c.prevCPU = cur
	}
	s.MemTotal, s.MemUsed, s.SwapTotal, s.SwapUsed = readMem()
	s.DiskTotal, s.DiskUsed = diskUsage("/")
	if cur := readNetBytes(); cur[0] >= c.prevNet[0] {
		s.NetRx = float64(cur[0]-c.prevNet[0]) / dt
		s.NetTx = float64(cur[1]-c.prevNet[1]) / dt
		c.prevNet = cur
	}
	c.prevAt = now
	s.Load1, s.Load5, s.Load15 = readLoad()
	s.UptimeS = readUptime()
	s.Temps = readTemps()
	s.Containers = readContainers()
	return s
}

func push(url, token string, s sample) error {
	body, _ := json.Marshal(s)
	req, _ := http.NewRequest(http.MethodPost,
		strings.TrimRight(url, "/")+"/api/systems/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("server returned %d", res.StatusCode)
	}
	return nil
}

// ---------- /proc + /sys readers (Linux; all no-op-safe elsewhere) ----------

// readCPUTicks parses the aggregate cpu line: fields are user nice system
// idle iowait irq softirq steal — everything but idle+iowait counts busy.
func readCPUTicks() cpuTicks {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTicks{}
	}
	var t cpuTicks
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)[1:]
		var vals []uint64
		for _, v := range f {
			n, _ := strconv.ParseUint(v, 10, 64)
			vals = append(vals, n)
			t.total += n
		}
		if len(vals) >= 5 {
			t.idle = vals[3] + vals[4]
		}
		break
	}
	return t
}

func cpuInfo() (model string, cores int) {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", runtime.NumCPU()
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "processor") {
			cores++
		}
		if model == "" && strings.HasPrefix(line, "model name") {
			model = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}
	if cores == 0 {
		cores = runtime.NumCPU()
	}
	return model, cores
}

// readMem returns total/used bytes for RAM and swap from /proc/meminfo.
func readMem() (memTotal, memUsed, swapTotal, swapUsed float64) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}
	kb := map[string]float64{}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 {
			kb[strings.TrimSuffix(f[0], ":")], _ = strconv.ParseFloat(f[1], 64)
		}
	}
	memTotal = kb["MemTotal"] * 1024
	memUsed = (kb["MemTotal"] - kb["MemAvailable"]) * 1024
	swapTotal = kb["SwapTotal"] * 1024
	swapUsed = (kb["SwapTotal"] - kb["SwapFree"]) * 1024
	return
}

func diskUsage(path string) (total, used float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	total = float64(st.Blocks) * float64(st.Bsize)
	used = float64(st.Blocks-st.Bfree) * float64(st.Bsize)
	return total, used
}

// readNetBytes sums rx/tx across interfaces, skipping loopback.
func readNetBytes() [2]uint64 {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return [2]uint64{}
	}
	var rx, tx uint64
	for _, line := range strings.Split(string(b), "\n") {
		i := strings.Index(line, ":")
		if i < 0 || strings.Contains(line[:i], "lo") {
			continue
		}
		f := strings.Fields(line[i+1:])
		if len(f) >= 9 {
			r, _ := strconv.ParseUint(f[0], 10, 64)
			t, _ := strconv.ParseUint(f[8], 10, 64)
			rx += r
			tx += t
		}
	}
	return [2]uint64{rx, tx}
}

func readLoad() (l1, l5, l15 float64) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	f := strings.Fields(string(b))
	if len(f) >= 3 {
		l1, _ = strconv.ParseFloat(f[0], 64)
		l5, _ = strconv.ParseFloat(f[1], 64)
		l15, _ = strconv.ParseFloat(f[2], 64)
	}
	return
}

func readUptime() float64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	f := strings.Fields(string(b))
	if len(f) > 0 {
		v, _ := strconv.ParseFloat(f[0], 64)
		return v
	}
	return 0
}

// readTemps walks /sys/class/hwmon for temp*_input sensors.
func readTemps() map[string]float64 {
	out := map[string]float64{}
	hwmons, _ := os.ReadDir("/sys/class/hwmon")
	for _, hw := range hwmons {
		dir := "/sys/class/hwmon/" + hw.Name()
		chip, _ := os.ReadFile(dir + "/name")
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(name, "temp") || !strings.HasSuffix(name, "_input") {
				continue
			}
			b, err := os.ReadFile(dir + "/" + name)
			if err != nil {
				continue
			}
			millideg, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
			if err != nil || millideg <= 0 || millideg > 150000 {
				continue
			}
			label := strings.TrimSpace(string(chip))
			if lb, err := os.ReadFile(dir + "/" + strings.TrimSuffix(name, "_input") + "_label"); err == nil {
				if l := strings.TrimSpace(string(lb)); l != "" {
					label = l
				}
			}
			if _, dup := out[label]; !dup {
				out[label] = millideg / 1000
			}
		}
	}
	return out
}

// readContainers lists Docker containers via the unix socket when present.
func readContainers() []containerStat {
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		return nil
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", "/var/run/docker.sock")
			},
		},
	}
	res, err := client.Get("http://d/containers/json?all=1")
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	var raw []struct {
		Names []string `json:"Names"`
		State string   `json:"State"`
	}
	if json.NewDecoder(res.Body).Decode(&raw) != nil {
		return nil
	}
	out := make([]containerStat, 0, len(raw))
	for _, r := range raw {
		name := ""
		if len(r.Names) > 0 {
			name = strings.TrimPrefix(r.Names[0], "/")
		}
		out = append(out, containerStat{Name: name, State: r.State})
	}
	return out
}
