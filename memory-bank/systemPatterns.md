# System Patterns: Balancer

## Architecture Overview
```
[Clients] → [Auth] → [Balancer] → [Backend Pool] → [WireMock Servers]
```

## Core Components

### ProxyServerPool  
- **Pattern**: Strategy pattern for pluggable balancing algorithms
- **Implementation**: Delegates capacity management to BalancingStrategy interface
- **Strategies**: RoundRobinStrategy (semaphore) and WeightedStrategy (per-client quotas)
- **Behavior**: Strategy-specific capacity control and backend selection

### Balancing Strategies

#### RoundRobinStrategy
- **Pattern**: Semaphore-based capacity control
- **Implementation**: Buffered channel (`make(chan struct{}, maxCapacity)`)
- **Distribution**: Round-robin with health-aware selection
- **Use Case**: Simple equal distribution among all clients

#### WeightedStrategy  
- **Pattern**: Per-client quota management with spillover pool
- **Implementation**: Map of client-specific buffered channels + shared spillover
- **Distribution**: Proportional to client weights with fair spillover access
- **Quota Formula**: `(clientWeight / totalActiveWeights) * totalCapacity`
- **Features**: Event-driven recalculation, concurrency-safe updates, usage preservation

### Health Checking
- **Pattern**: Continuous background monitoring
- **Implementation**: Goroutine per backend with ticker
- **State**: Atomic boolean for thread-safe status updates
- **Recovery**: Automatic when health checks pass

### Request Flow
1. Client registration via `/register` (captures weight for weighted strategy)
2. Authentication check (except whitelisted paths)  
3. Client identification injection into request context
4. Strategy-specific capacity acquisition with timeout
5. Backend selection (strategy-dependent: round-robin or weighted)
6. Reverse proxy to backend
7. Strategy-specific capacity release on completion

## Design Decisions

### Capacity Management
- **Strategy Pattern**: Pluggable algorithms (RoundRobin vs Weighted)
- **RoundRobin**: System-wide semaphore for simplicity and equal distribution
- **Weighted**: Per-client quotas + spillover pool for proportional fairness
- **Event-Driven Recalculation**: Quota updates on client registration changes
- **Concurrency Safety**: Preserves current usage during quota transitions
- **Timeout Strategy**: Fail fast when capacity unavailable (both strategies)

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