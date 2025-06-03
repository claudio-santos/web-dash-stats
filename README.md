# web-dash-stats 🖥️📊

A simple real-time web dashboard to monitor system CPU and memory usage using Go, Bootstrap, and Server-Sent Events (SSE).

> *Live updates without page reloads — built with simplicity in mind.*

---

## 🔍 Features

- ✅ Real-time CPU and memory usage updates
- 🚦 Powered by **Server-Sent Events (SSE)** for live streaming
- 🎨 Styled with **Bootstrap 5.3** (local static files)
- 🌙 Dark mode enabled by default via `data-bs-theme="dark"`
- 📁 Clean folder structure:
  - HTML templates in `templates/`
  - Static assets (CSS, JS) in `static/`

---

## 🧰 Technologies Used

- **Go (Golang)** – For the backend HTTP server
- **gopsutil/v4** – To fetch system metrics
- **Bootstrap 5.3** – For styling (no CDN, fully local/static)
- **HTML + Vanilla JS** – Frontend UI logic
- **SSE (Server-Sent Events)** – For real-time updates

---

## 📁 Project Structure

```
project/
├── main.go
├── templates/
│   └── index.html
└── static/
    ├── bootstrap.min.css
    └── bootstrap.bundle.min.js
```

---

## 🛠️ Setup Instructions

### 1. Install Dependencies

```bash
go get github.com/shirou/gopsutil/v4/cpu
go get github.com/shirou/gopsutil/v4/mem
```

### 2. Folder Setup

Make sure you have the following structure:

- Place Bootstrap CSS and JS files in the `static/` directory.
- Place `index.html` inside the `templates/` directory.

---

## ▶️ Run the App

```bash
go run main.go
```

Open your browser and navigate to:

```
http://localhost:3333
```

You'll see live CPU and memory usage updating every second.

---

## 📦 Releases

Pre-built binaries are available for download on GitHub:

🔗 [View latest releases](https://github.com/claudio-santos/web-dash-stats/releases)

---

## 📄 License

This project is licensed under the **GNU Affero General Public License v3.0**  
For more information, see the [`LICENSE`](LICENSE) file.

---
