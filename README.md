# web-dash-stats 📊

A minimal real-time web dashboard for monitoring system and custom service status using Go, Bootstrap, and Server-Sent Events (SSE).

---

## Features

- Real-time CPU, memory, network, storage, and uptime updates
- Live service status checks (configurable via YAML)
- Powered by **Server-Sent Events (SSE)** for instant UI updates
- Responsive, dark-themed UI with **Bootstrap 5**
- Easy configuration: add or remove monitored services without code changes

---

## Technologies Used

- **Go (Golang)** – Backend HTTP server and metrics
- **gopsutil/v4** – System stats collection
- **Bootstrap 5** – UI styling (local static files)
- **HTML + Vanilla JS** – Frontend logic
- **SSE (Server-Sent Events)** – Real-time data streaming
- **YAML** – Service configuration

---

## Project Structure

```
project/
├── main.go
├── config.go
├── config.yaml
├── templates/
│   └── index.html
└── static/
    ├── bootstrap.min.css
    └── bootstrap.bundle.min.js
```

---

## Configuration

Define your monitored services in `config.yaml`:

```yaml
services:
  - name: ExampleService
    url: http://127.0.0.1:1234
  # Add more services as needed
```

---

## Setup Instructions

### 1. Install Dependencies

```bash
go get github.com/shirou/gopsutil/v4/cpu
go get github.com/shirou/gopsutil/v4/mem
go get github.com/shirou/gopsutil/v4/net
go get github.com/shirou/gopsutil/v4/host
go get github.com/shirou/gopsutil/v4/disk
go get gopkg.in/yaml.v3
```

### 2. Folder Setup

- Place Bootstrap CSS and JS files in the `static/` directory.
- Place `index.html` inside the `templates/` directory.
- Make sure your `config.yaml` is in the project root.

---

## Run the App

```bash
go run main.go
```

Open your browser and go to:

```
http://localhost:3333
```

You'll see live system stats and service status updating automatically.

---

## License

This project is licensed under the **GNU Affero General Public License v3.0**.  
See the [`LICENSE`](LICENSE) file for details.

---
