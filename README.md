# Zenc Server

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://hub.docker.com/r/aminofox/zenc-server)

**Zenc Server** is the core media server for the Zenc livestreaming ecosystem. Built in Go for high performance and concurrency, it handles real-time media routing using WebRTC SFU (Selective Forwarding Unit) architecture.

## 🌟 Features

- **WebRTC SFU Implementation** - Selective forwarding for efficient media routing
- **Multi-Host Support** - Distributed architecture with automatic failover
- **Redis Integration** - State management and pub/sub for cross-server communication
- **Stream Zones** - Geographic/logical grouping for better performance
- **REST & WebSocket APIs** - Flexible integration options
- **Real-time Metrics** - Prometheus-compatible metrics endpoint
- **Docker Ready** - Easy deployment with Docker and Docker Compose

## 🚀 Quick Start

### Prerequisites

- Go 1.22 or higher
- Redis 7.0 or higher
- Docker & Docker Compose (optional)

### Installation

#### Option 1: Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/aminofox/zenc-server.git
cd zenc-server

# Start the server with Redis
docker-compose up -d

# View logs
docker-compose logs -f zenc-server
```

#### Option 2: Build from Source

```bash
# Install dependencies
make deps

# Build the server
make build

# Run the server
./zenc-server
```

#### Option 3: Run Directly

```bash
go run ./cmd/server
```

## 📋 Configuration

Configuration is managed via `config.yaml`. Here's a basic example:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  name: zenc-server-1

webrtc:
  ice_servers:
    - urls: 
        - stun:stun.l.google.com:19302
  port_range:
    min: 10000
    max: 20000

redis:
  mode: single
  addresses:
    - localhost:6379
  password: ""
  db: 0

zones:
  - id: default
    region: local
    primary: true

logging:
  level: info
  format: json

metrics:
  enabled: true
  port: 9090
  path: /metrics
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `server.port` | HTTP server port | 8080 |
| `webrtc.port_range` | UDP port range for WebRTC | 10000-20000 |
| `redis.mode` | Redis mode (single, cluster, sentinel) | single |
| `logging.level` | Log level (debug, info, warn, error) | info |
| `metrics.enabled` | Enable Prometheus metrics | true |

## 🔌 API Reference

### Health Check

```bash
GET /health
```

Response:
```json
{
  "status": "healthy",
  "server": "zenc-server-1",
  "time": 1706140800
}
```

### Room Management

#### Create Room

```bash
POST /api/v1/rooms
Content-Type: application/json

{
  "name": "My Livestream",
  "max_users": 100,
  "zone": "default"
}
```

#### Get Room

```bash
GET /api/v1/rooms/:id
```

#### List Rooms

```bash
GET /api/v1/rooms
```

#### Delete Room

```bash
DELETE /api/v1/rooms/:id
```

### Participant Management

#### Join Room

```bash
POST /api/v1/rooms/:id/join
Content-Type: application/json

{
  "user_id": "user123",
  "role": "publisher"
}
```

#### Leave Room

```bash
POST /api/v1/rooms/:id/leave
Content-Type: application/json

{
  "participant_id": "participant-uuid"
}
```

#### Get Participants

```bash
GET /api/v1/rooms/:id/participants
```

### Track Management

#### Publish Track

```bash
POST /api/v1/tracks/publish
Content-Type: application/json

{
  "participant_id": "participant-uuid",
  "kind": "video",
  "source": "camera",
  "sdp_offer": "v=0..."
}
```

#### Subscribe to Track

```bash
POST /api/v1/tracks/subscribe
Content-Type: application/json

{
  "participant_id": "participant-uuid",
  "track_id": "track-uuid"
}
```

#### Hold Track (Pause)

```bash
POST /api/v1/tracks/hold
Content-Type: application/json

{
  "track_id": "track-uuid"
}
```

#### Resume Track

```bash
POST /api/v1/tracks/resume
Content-Type: application/json

{
  "track_id": "track-uuid"
}
```

## 🔌 WebSocket API

Connect to `ws://localhost:8080/ws` for real-time signaling.

### Message Format

```json
{
  "type": "message_type",
  "payload": {
    "key": "value"
  }
}
```

### Message Types

- `join` - Join a room
- `leave` - Leave a room
- `offer` - WebRTC offer
- `answer` - WebRTC answer
- `ice_candidate` - ICE candidate

## 📊 Metrics

Prometheus metrics are available at `http://localhost:9090/metrics` (if enabled).

## 🏗️ Architecture

```
zenc-server/
├── cmd/
│   └── server/           # Main application entry point
├── internal/
│   ├── api/
│   │   ├── httpapi/      # REST API handlers
│   │   └── websocket/    # WebSocket handlers
│   ├── room/             # Room management
│   ├── participant/      # Participant logic
│   ├── media/            # Media routing (SFU core)
│   └── store/            # Redis storage layer
├── pkg/
│   ├── config/           # Configuration management
│   ├── logger/           # Logging utilities
│   └── protocol/         # Protocol definitions
├── config.yaml           # Configuration file
├── Dockerfile            # Docker image
└── docker-compose.yml    # Docker Compose setup
```

## 🧪 Development

### Run Tests

```bash
make test
```

### Generate Coverage Report

```bash
make test-coverage
```

### Format Code

```bash
make fmt
```

### Run Linter

```bash
make lint
```

### Build Docker Image

```bash
make docker-build
```

## 🔧 Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the server binary |
| `make run` | Run the server locally |
| `make test` | Run tests |
| `make clean` | Clean build artifacts |
| `make docker-build` | Build Docker image |
| `make docker-run` | Run with Docker Compose |
| `make docker-stop` | Stop Docker Compose |

## 🌐 Production Deployment

### Using Docker

```bash
# Build the image
docker build -t aminofox/zenc-server:latest .

# Run the container
docker run -d \
  --name zenc-server \
  -p 8080:8080 \
  -p 9090:9090 \
  -p 10000-20000:10000-20000/udp \
  -v $(pwd)/config.yaml:/root/config.yaml:ro \
  aminofox/zenc-server:latest
```

### Environment Variables

You can override configuration via environment variables:

```bash
export SERVER_PORT=8080
export REDIS_ADDR=redis:6379
export LOG_LEVEL=info
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 Roadmap

### Phase 1: MVP (Current)
- [x] Basic room management
- [x] REST API
- [x] WebSocket signaling
- [x] Redis integration
- [ ] Full WebRTC implementation with Pion

### Phase 2: Scalability
- [ ] Multi-server support
- [ ] Cross-server communication
- [ ] Load balancing
- [ ] Automatic failover

### Phase 3: Advanced Features
- [ ] Stream composition engine
- [ ] Quality adaptation
- [ ] Recording support
- [ ] RTMP ingestion/egress

## 🐛 Troubleshooting

### Server won't start

1. Check if Redis is running:
   ```bash
   redis-cli ping
   # Should return: PONG
   ```

2. Verify configuration file exists:
   ```bash
   ls -la config.yaml
   ```

3. Check logs for errors:
   ```bash
   docker-compose logs zenc-server
   ```

### WebRTC connection fails

1. Ensure UDP ports 10000-20000 are open
2. Check firewall settings
3. Verify STUN server is accessible

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

**aminofox**

- GitHub: [@aminofox](https://github.com/aminofox)
- Repository: [zenc-server](https://github.com/aminofox/zenc-server)

## 🔗 Related Projects

- [zenc-sdk](https://github.com/aminofox/zenc-sdk) - Client SDK
- [zenc-protocol](https://github.com/aminofox/zenc-protocol) - Protocol definitions
- [zenc-admin](https://github.com/aminofox/zenc-admin) - Admin dashboard

## ⭐ Support

If you find this project helpful, please give it a star! ⭐

## 📞 Contact

For questions or support, please open an issue on GitHub.

---

**Built with ❤️ by aminofox**

Part of the [Zenc Livestreaming Ecosystem](https://github.com/aminofox)
