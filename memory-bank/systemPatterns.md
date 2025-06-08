# System Patterns: Balancer

## Architecture Overview
```
[Clients] → [Auth] → [Balancer] → [Backend Pool] → [WireMock Servers]
```

## Core Components

### ProxyServerPool
- **Pattern**: Semaphore-based capacity control
- **Implementation**: Buffered channel (`make(chan struct{}, maxCapacity)`)
- **Behavior**: Blocks requests when capacity exhausted
- **Distribution**: Round-robin with health-aware selection

### Health Checking
- **Pattern**: Continuous background monitoring
- **Implementation**: Goroutine per backend with ticker
- **State**: Atomic boolean for thread-safe status updates
- **Recovery**: Automatic when health checks pass

### Request Flow
1. Client registration via `/register`
2. Authentication check (except whitelisted paths)
3. Capacity acquisition with timeout
4. Backend selection (round-robin + health)
5. Reverse proxy to backend
6. Capacity release on completion

## Design Decisions

### Capacity Management
- **Semaphore over Rate Limiting**: Protects backend from overload
- **System-wide vs Per-Backend**: Single capacity pool for simplicity
- **Timeout Strategy**: Fail fast when capacity unavailable

### Backend Selection
- **Round-robin over Random**: Predictable load distribution
- **Health-aware**: Skip unhealthy backends automatically
- **Retry Logic**: Check all backends before failing

### Error Handling
- **Graceful Degradation**: Continue with healthy backends
- **Client Feedback**: Clear error messages for capacity limits
- **Logging**: Comprehensive request and health check logging

## Scalability Patterns
- Stateless design enables horizontal scaling
- Configuration-driven backend pool management
- Resource pooling for HTTP clients and connections 