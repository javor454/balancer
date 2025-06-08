# Technical Context: Balancer

## Technology Stack
- **Runtime**: Go 1.23.6
- **HTTP**: Standard library (`net/http`, `net/http/httputil`)
- **Concurrency**: Goroutines, channels, atomic operations
- **Testing**: Testify framework
- **Containerization**: Docker + Docker Compose

## Development Environment
- **Build**: `make up` (Docker Compose)
- **Testing**: `make traffic` (traffic simulation)
- **Backend**: 3x WireMock servers (ports 8081-8083)
- **Balancer**: Port 8080

## Key Dependencies
```go
net/http/httputil     // Reverse proxy implementation
sync/atomic          // Thread-safe health status
context              // Request cancellation, timeouts, client identification
sync                 // RWMutex for thread-safe quota management
log                  // Structured logging for capacity operations
```

## Configuration
- **Max Capacity**: Configurable per strategy (default 5 concurrent requests)
- **Health Check**: 5-second intervals
- **Request Timeout**: 10 seconds
- **Acquire Timeout**: 10 seconds (configurable per strategy)
- **Shutdown**: 10-second graceful period
- **Strategy Selection**: Build-time configuration (RoundRobin vs Weighted)

## Strategy-Specific Configuration

### RoundRobinStrategy
- **Capacity Model**: Global semaphore (traditional approach)
- **Distribution**: Equal among all clients

### WeightedStrategy  
- **Capacity Model**: Per-client quotas + spillover pool
- **Quota Calculation**: `(clientWeight / totalActiveWeights) * totalCapacity`
- **Recalculation**: Event-driven on client registration changes
- **Minimum Quota**: 1 slot per active client
- **Fairness**: Proportional with spillover redistribution

## Deployment Architecture
```yaml
balancer:8080 → wiremock1:8081
              → wiremock2:8082  
              → wiremock3:8083
```

## Development Commands
- `make up`: Start all services
- `make down`: Stop all services
- `make traffic`: Generate test load
- `go build -o ./target/balancer`: Build binary

## Security
- JWT-based authentication (auth package)
- Path-based access control
- CORS enabled on backends

## Monitoring
- Health check endpoints (`/health`)
- Request logging
- Capacity metrics available 