
---

# Go-Transcoder 🚀

A distributed, event-driven video transcoding engine that turns raw uploads into production-ready HLS streams.

### ✨ Key Features

* **Distributed Architecture:** Decoupled API (Producer) and Worker (Consumer) using Kafka.
* **Adaptive Bitrate Streaming:** Generates HLS playlists with multiple resolutions (360p to 4K).
* **Smart Resolution Filtering:** Automatically detects source height to prevent useless upscaling.
* **Real-time Progress:** Multi-threaded terminal UI for tracking concurrent transcoding tasks.
* **Hardware Accelerated Ready:** Optimized FFmpeg configurations for H.264 encoding.

### 🏗️ Architecture

The system is divided into three main components:

1. **Web API:** Handles file uploads and validates video metadata.
2. **Kafka Message Bus:** Manages the task queue, ensuring fault tolerance.
3. **Transcoder Worker:** Consumes jobs and performs heavy-duty FFmpeg processing.

---

### 🚀 Getting Started

#### 1. Prerequisites

* [Go](https://golang.org/doc/install) 1.21+
* [FFmpeg](https://ffmpeg.org/download.html)
* [Docker & Docker Compose](https://docs.docker.com/get-docker/)
* **Librdkafka** (for Kafka Go client)
* macOS: `brew install librdkafka`
* Ubuntu: `sudo apt install librdkafka-dev`



#### 2. Run Infrastructure

Start Kafka and Zookeeper via Docker:

```bash
docker-compose up -d zookeeper kafka

```

#### 3. Run the Application

You can run the application in different modes. Open two terminals:

**Terminal 1: Start the Transcoder Worker**

```bash
go run main.go -mode=worker

```

**Terminal 2: Start the API Server**

```bash
go run main.go -mode=api

```

### 📂 Directory Structure

```text
.
├── infrastructure/kafka  # Producer & Consumer logic
├── server/               # HTTP Handlers & Upload validation
├── service/              # FFmpeg logic & Master Playlist generation
├── uploads/              # Temporary storage for raw videos
└── output/               # Final HLS segments and .m3u8 files

```

### 🧪 API Usage

**Upload a Video**

```bash
curl -X POST -F "file=@myvideo.mp4" http://localhost:8080/upload

```

**List Videos**
Access `http://localhost:8080/` to view the gallery and test adaptive quality switching.

---

### 🛠️ Configuration

You can adjust standard resolutions and bitrates in `service/utils.go` and `infrastructure/kafka/utils.go`. Default targets include:

* **360p, 720p, 1080p, 1440p (2K), 2160p (4K)**
