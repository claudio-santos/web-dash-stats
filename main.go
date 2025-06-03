package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func sseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			v, _ := mem.VirtualMemory()
			cpuPercent, _ := cpu.Percent(time.Second, false)

			fmt.Fprintf(w, "data: {\"mem\": %.2f, \"cpu\": %.2f}\n\n", v.UsedPercent, cpuPercent[0])
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
