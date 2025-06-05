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

	bootTime, _ := host.BootTime()
	osType := runtime.GOOS
	arch := runtime.GOARCH
	hostname, _ := os.Hostname()

	cpuInfo, err := cpu.Info()
	var cpuModel string = "Unknown"
	if err == nil && len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastRx, lastTx uint64 = 0, 0

	for {
		select {
		case <-ticker.C:
			v, _ := mem.VirtualMemory()
			cpuPercent, _ := cpu.Percent(time.Second, false)

			// Get all network interfaces
			ioCounters, _ := net.IOCounters(true)
			var rx, tx uint64 = 0, 0

			// Sum all interfaces
			for _, counter := range ioCounters {
				rx += counter.BytesRecv
				tx += counter.BytesSent
			}

			// Calculate rate per second
			rateRx := rx - lastRx
			rateTx := tx - lastTx
			lastRx, lastTx = rx, tx

			uptimeSec := time.Now().Unix() - int64(bootTime)
			uptime := prettyUptime(uptimeSec)

			data := map[string]any{
				"mem":       v.UsedPercent,
				"cpu":       cpuPercent[0],
				"platform":  fmt.Sprintf("%s/%s", osType, arch),
				"uptime":    uptime,
				"hostname":  hostname,
				"cpu_model": cpuModel,
				"rx":        rx,
				"tx":        tx,
				"rateRx":    rateRx,
				"rateTx":    rateTx,
			}
			jsonBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
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
