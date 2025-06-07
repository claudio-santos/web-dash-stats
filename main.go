package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// prettyUptime returns uptime as "Xd Yh Zm"
func prettyUptime(seconds int64) string {
	days := seconds / (24 * 3600)
	hours := (seconds % (24 * 3600)) / 3600
	minutes := (seconds % 3600) / 60
	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", minutes))
	return strings.Join(parts, " ")
}

// getStorageInfo returns a slice of storage info for all drives
func getStorageInfo() []map[string]any {
	partitions, _ := disk.Partitions(false)
	var drives []map[string]any
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}
		drives = append(drives, map[string]any{
			"mount":   p.Mountpoint,
			"used":    usage.Used,
			"total":   usage.Total,
			"percent": usage.UsedPercent,
		})
	}
	return drives
}

// sendSSE sends a Server-Sent Event to the client and flushes the response.
func sendSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data any, name ...string) {
	dataBytes, _ := json.Marshal(data)
	var payload string
	if len(name) == 1 {
		payload = fmt.Sprintf("data: {\"type\":\"%s\",\"name\":\"%s\",\"data\":%s}\n\n", eventType, name[0], dataBytes)
	} else {
		payload = fmt.Sprintf("data: {\"type\":\"%s\",\"data\":%s}\n\n", eventType, dataBytes)
	}
	fmt.Fprint(w, payload)
	flusher.Flush()
}

// checkServiceAndSend checks a service URL and sends its status as SSE.
func checkServiceAndSend(w http.ResponseWriter, flusher http.Flusher, name, url string, timeout time.Duration) {
	status := "down"
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		status = "up"
	}
	data := map[string]any{"status": status}
	sendSSE(w, flusher, "service", data, name)
}

// sseHandler streams system and service stats to the frontend via SSE.
func sseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	bootTime, _ := host.BootTime()
	arch := runtime.GOARCH
	hostname, _ := os.Hostname()
	cpuInfo, _ := cpu.Info()
	cpuModel := "Unknown"
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	// Platform string
	hostInfo, _ := host.Info()
	platform := arch
	if hostInfo != nil {
		switch {
		case hostInfo.Platform != "" && hostInfo.PlatformVersion != "":
			platform = fmt.Sprintf("%s %s %s", hostInfo.Platform, hostInfo.PlatformVersion, arch)
		case hostInfo.Platform != "":
			platform = fmt.Sprintf("%s %s", hostInfo.Platform, arch)
		}
	}

	// Initial info sent to frontend
	sendSSE(w, flusher, "sysinfo", map[string]any{
		"hostname":  hostname,
		"platform":  platform,
		"cpu_model": cpuModel,
	})
	sendSSE(w, flusher, "services", services)
	uptimeSec := time.Now().Unix() - int64(bootTime)
	sendSSE(w, flusher, "uptime", map[string]any{"uptime": prettyUptime(uptimeSec)})
	sendSSE(w, flusher, "storage", getStorageInfo())

	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		sendSSE(w, flusher, "cpu", map[string]any{"cpu": cpuPercent[0]})
	}
	if v, err := mem.VirtualMemory(); err == nil {
		sendSSE(w, flusher, "mem", map[string]any{
			"mem":   v.UsedPercent,
			"used":  v.Used,
			"total": v.Total,
		})
	}

	// Network counters for rate calculation
	ioCounters, _ := net.IOCounters(true)
	var lastRx, lastTx uint64
	for _, counter := range ioCounters {
		lastRx += counter.BytesRecv
		lastTx += counter.BytesSent
	}
	sendSSE(w, flusher, "network", map[string]any{
		"rx":     lastRx,
		"tx":     lastTx,
		"rateRx": float64(0),
		"rateTx": float64(0),
	})

	// Tickers for periodic updates
	networkTicker := time.NewTicker(1 * time.Second)
	defer networkTicker.Stop()
	cpuTicker := time.NewTicker(1 * time.Second)
	defer cpuTicker.Stop()
	memTicker := time.NewTicker(1 * time.Second)
	defer memTicker.Stop()
	uptimeTicker := time.NewTicker(1 * time.Minute)
	defer uptimeTicker.Stop()
	storageTicker := time.NewTicker(5 * time.Second)
	defer storageTicker.Stop()
	serviceTicker := time.NewTicker(5 * time.Minute)
	defer serviceTicker.Stop()

	// Initial service status check
	for _, svc := range services {
		checkServiceAndSend(w, flusher, svc.Name, svc.URL, 2*time.Second)
	}

	// Main event loop: send updates on each ticker
	for {
		select {
		case <-cpuTicker.C:
			if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
				sendSSE(w, flusher, "cpu", map[string]any{"cpu": cpuPercent[0]})
			}
		case <-memTicker.C:
			if v, err := mem.VirtualMemory(); err == nil {
				sendSSE(w, flusher, "mem", map[string]any{
					"mem":   v.UsedPercent,
					"used":  v.Used,
					"total": v.Total,
				})
			}
		case <-uptimeTicker.C:
			uptimeSec := time.Now().Unix() - int64(bootTime)
			sendSSE(w, flusher, "uptime", map[string]any{"uptime": prettyUptime(uptimeSec)})
		case <-networkTicker.C:
			if ioCounters, err := net.IOCounters(true); err == nil {
				var rx, tx uint64
				for _, counter := range ioCounters {
					rx += counter.BytesRecv
					tx += counter.BytesSent
				}
				rateRx := rx - lastRx
				rateTx := tx - lastTx
				lastRx, lastTx = rx, tx
				sendSSE(w, flusher, "network", map[string]any{
					"rx":     rx,
					"tx":     tx,
					"rateRx": float64(rateRx),
					"rateTx": float64(rateTx),
				})
			}
		case <-storageTicker.C:
			sendSSE(w, flusher, "storage", getStorageInfo())
		case <-serviceTicker.C:
			for _, svc := range services {
				checkServiceAndSend(w, flusher, svc.Name, svc.URL, 2*time.Second)
			}
		case <-r.Context().Done():
			return
		}
	}
}

var services []Service

func main() {
	// Load services from config.yaml
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	services = cfg.Services

	// Serve static files and dashboard
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})
	http.HandleFunc("/events", sseHandler)

	log.Println("Starting server at :3333")
	log.Fatal(http.ListenAndServe(":3333", nil))
}
