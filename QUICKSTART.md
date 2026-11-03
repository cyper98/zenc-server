# Zenc Server - Quick Start Guide

This guide will help you get Zenc Server up and running in just a few minutes.

## Prerequisites

Before you begin, ensure you have:

- **Go 1.22+** installed ([Download](https://golang.org/dl/))
- **Redis 7.0+** installed ([Download](https://redis.io/download))
- **Docker & Docker Compose** (optional, for containerized deployment)

## Installation Methods

### Method 1: Docker Compose (Easiest) 🐳

This method requires only Docker and Docker Compose installed.

```bash
# 1. Clone the repository
git clone https://github.com/cyper98/zenc-server.git
cd zenc-server

# 2. Start everything (server + Redis)
docker-compose up -d

# 3. Check if it's running
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","server":"zenc-server","time":1706140800}

# 4. View logs
docker-compose logs -f zenc-server
```

That's it! The server is now running at `http://localhost:8080`.

---

### Method 2: Build from Source 🔨

```bash
# 1. Clone the repository
git clone https://github.com/cyper98/zenc-server.git
cd zenc-server

# 2. Install dependencies
go mod download

# 3. Ensure Redis is running
redis-server &
# Or start Redis in another terminal

# 4. Build and run
make build
./zenc-server

# Or run directly
make run
```

---

### Method 3: Quick Development Setup 🚀

```bash
# 1. Start Redis (if not already running)
docker run -d -p 6379:6379 --name zenc-redis redis:7-alpine

# 2. Clone and run
git clone https://github.com/cyper98/zenc-server.git
cd zenc-server
go run ./cmd/server
```

---

## Testing the API

### 1. Check Server Health

```bash
curl http://localhost:8080/health
```

### 2. Create a Room

```bash
curl -X POST http://localhost:8080/api/v1/rooms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Stream",
    "max_users": 50,
    "zone": "default"
  }'
```

Expected response:
```json
{
  "id": "room-uuid-here",
  "name": "My First Stream",
  "max_users": 50,
  "created_at": "2026-01-26T10:00:00Z",
  "zone": "default",
  "metadata": null
}
```

### 3. List All Rooms

```bash
curl http://localhost:8080/api/v1/rooms
```

### 4. Join a Room

Replace `ROOM_ID` with the ID from step 2:

```bash
curl -X POST http://localhost:8080/api/v1/rooms/ROOM_ID/join \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "role": "publisher"
  }'
```

### 5. Get Room Participants

```bash
curl http://localhost:8080/api/v1/rooms/ROOM_ID/participants
```

---

## Testing WebSocket Connection

You can test WebSocket connectivity using a simple HTML page:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Zenc WebSocket Test</title>
</head>
<body>
    <h1>Zenc WebSocket Test</h1>
    <div id="status">Connecting...</div>
    <div id="messages"></div>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        
        ws.onopen = () => {
            document.getElementById('status').textContent = 'Connected!';
            console.log('WebSocket connected');
            
            // Send a join message
            ws.send(JSON.stringify({
                type: 'join',
                payload: {
                    room_id: 'test-room',
                    user_id: 'test-user'
                }
            }));
        };
        
        ws.onmessage = (event) => {
            console.log('Received:', event.data);
            const div = document.createElement('div');
            div.textContent = 'Received: ' + event.data;
            document.getElementById('messages').appendChild(div);
        };
        
        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            document.getElementById('status').textContent = 'Error!';
        };
        
        ws.onclose = () => {
            console.log('WebSocket disconnected');
            document.getElementById('status').textContent = 'Disconnected';
        };
    </script>
</body>
</html>
```

Save this as `test.html` and open it in your browser.

---

## View Metrics

If metrics are enabled (default), you can view Prometheus metrics:

```bash
curl http://localhost:9090/metrics
```

---

## Configuration

Edit `config.yaml` to customize settings:

```yaml
server:
  port: 8080        # Change HTTP port
  
webrtc:
  port_range:
    min: 10000      # WebRTC UDP port range
    max: 20000

redis:
  mode: single      # single, cluster, sentinel
  addresses:
    - localhost:6379

logging:
  level: info       # debug, info, warn, error
  format: json      # json, console
```

---

## Useful Make Commands

```bash
make build         # Build the server binary
make run           # Run the server
make test          # Run tests
make clean         # Clean build artifacts
make docker-build  # Build Docker image
make docker-run    # Start with Docker Compose
make docker-stop   # Stop Docker Compose
make fmt           # Format code
make help          # Show all commands
```

---

## Stopping the Server

### If using Docker Compose:
```bash
docker-compose down
```

### If running directly:
Press `Ctrl+C` in the terminal

### If running in background:
```bash
pkill zenc-server
```

---

## Troubleshooting

### "Connection refused" error

**Solution:** Make sure Redis is running
```bash
# Check if Redis is running
redis-cli ping
# Should return: PONG

# If not, start Redis
redis-server &
# Or with Docker:
docker run -d -p 6379:6379 redis:7-alpine
```

### "Port already in use" error

**Solution:** Change the port in `config.yaml` or stop the process using the port
```bash
# Find process using port 8080
lsof -i :8080
# Kill it
kill -9 <PID>
```

### WebSocket connection fails

**Solution:** Check browser console for errors. Ensure:
- Server is running
- Using correct WebSocket URL (`ws://` not `wss://` for local development)
- No firewall blocking the connection

---

## Next Steps

1. ✅ Server is running
2. 📖 Read the [full README](README.md) for detailed API documentation
3. 🔧 Check the [Architecture Documentation](ARCHITECTURE.md) (coming soon)
4. 💻 Try the [zenc-sdk](https://github.com/cyper98/zenc-sdk) for client integration
5. 🎯 Build your first livestreaming app!

---

## Getting Help

- 🐛 **Found a bug?** [Open an issue](https://github.com/cyper98/zenc-server/issues)
- 💬 **Have a question?** [Start a discussion](https://github.com/cyper98/zenc-server/discussions)
- 📧 **Need support?** Contact cyper98

---

**Happy Streaming! 🎥**
