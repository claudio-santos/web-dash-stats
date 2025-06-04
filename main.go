package main

import (
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

			ioCounters, _ := net.IOCounters(false)
			var rx, tx uint64 = 0, 0
			if len(ioCounters) > 0 {
				rx = ioCounters[0].BytesRecv
				tx = ioCounters[0].BytesSent
			}

			rateRx := rx - lastRx
			rateTx := tx - lastTx

			lastRx = rx
			lastTx = tx

			uptimeSec := time.Now().Unix() - int64(bootTime)
			uptime := prettyUptime(uptimeSec)

			fmt.Fprintf(w, "data: {\"mem\": %.2f, \"cpu\": %.2f, \"platform\": \"%s/%s\", \"uptime\": \"%s\", \"hostname\": \"%s\", \"cpu_model\": \"%s\", \"rx\": %d, \"tx\": %d, \"rateRx\": %d, \"rateTx\": %d}\n\n",
				v.UsedPercent, cpuPercent[0],
				osType, arch,
				uptime,
				hostname,
				cpuModel,
				rx, tx,
				rateRx, rateTx,
			)
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
