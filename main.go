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
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

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

func sseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	bootTime, err := host.BootTime()
	if err != nil {
		http.Error(w, "Could not get boot time", http.StatusInternalServerError)
		return
	}
	osType := runtime.GOOS
	arch := runtime.GOARCH
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Unknown"
	}
	cpuInfo, err := cpu.Info()
	cpuModel := "Unknown"
	if err == nil && len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	// Send system info once
	sysInfo := map[string]any{
		"hostname":  hostname,
		"platform":  fmt.Sprintf("%s/%s", osType, arch),
		"cpu_model": cpuModel,
	}
	sysInfoBytes, _ := json.Marshal(sysInfo)
	fmt.Fprintf(w, "data: {\"type\":\"sysinfo\",\"data\":%s}\n\n", sysInfoBytes)
	flusher.Flush()

	// Initialize lastRx and lastTx to avoid spike on first update
	ioCounters, err := net.IOCounters(true)
	var lastRx, lastTx uint64 = 0, 0
	if err == nil {
		for _, counter := range ioCounters {
			lastRx += counter.BytesRecv
			lastTx += counter.BytesSent
		}
	}

	networkTicker := time.NewTicker(1 * time.Second)
	defer networkTicker.Stop()
	cpuTicker := time.NewTicker(1 * time.Second)
	defer cpuTicker.Stop()
	memTicker := time.NewTicker(1 * time.Second)
	defer memTicker.Stop()
	uptimeTicker := time.NewTicker(1 * time.Minute)
	defer uptimeTicker.Stop()

	for {
		select {
		case <-cpuTicker.C:
			cpuPercent, err := cpu.Percent(0, false)
			if err != nil || len(cpuPercent) == 0 {
				continue
			}
			data := map[string]any{
				"cpu": cpuPercent[0],
			}
			jsonBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: {\"type\":\"cpu\",\"data\":%s}\n\n", jsonBytes)
			flusher.Flush()

		case <-memTicker.C:
			v, err := mem.VirtualMemory()
			if err != nil {
				continue
			}
			data := map[string]any{
				"mem": v.UsedPercent,
			}
			jsonBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: {\"type\":\"mem\",\"data\":%s}\n\n", jsonBytes)
			flusher.Flush()

		case <-uptimeTicker.C:
			uptimeSec := time.Now().Unix() - int64(bootTime)
			uptime := prettyUptime(uptimeSec)
			data := map[string]any{
				"uptime": uptime,
			}
			jsonBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: {\"type\":\"uptime\",\"data\":%s}\n\n", jsonBytes)
			flusher.Flush()

		case <-networkTicker.C:
			ioCounters, err := net.IOCounters(true)
			if err != nil {
				continue
			}
			var rx, tx uint64 = 0, 0
			for _, counter := range ioCounters {
				rx += counter.BytesRecv
				tx += counter.BytesSent
			}
			rateRx := rx - lastRx
			rateTx := tx - lastTx
			lastRx, lastTx = rx, tx

			data := map[string]any{
				"rx":     rx,
				"tx":     tx,
				"rateRx": float64(rateRx), // bytes/sec
				"rateTx": float64(rateTx),
			}
			jsonBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: {\"type\":\"network\",\"data\":%s}\n\n", jsonBytes)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func main() {
	// Serve static files from /static
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve HTML templates from /templates
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	// SSE endpoint
	http.HandleFunc("/events", sseHandler)

	log.Println("Starting server at :3333")
	log.Fatal(http.ListenAndServe(":3333", nil))
}
