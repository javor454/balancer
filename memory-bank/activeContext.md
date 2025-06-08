# Active Context: Balancer

## Current State
- **Phase**: Implementing weighted client distribution feature
- **Status**: Planning major architecture refactor to support weighted balancing
- **Last Focus**: Strategy pattern design for balancing algorithms

## Recently Implemented
- Reverse proxy with capacity management
- Round-robin backend selection with health checks
- Client registration and authentication with weight capture
- Graceful shutdown handling
- Docker-based development environment
- String vs byte slice performance optimizations
- **BalancingStrategy interface with clean separation of concerns**
- **RoundRobinStrategy extraction maintaining backward compatibility**
- **ProxyServerPool refactor to use strategy pattern delegation**

## Current Focus Areas
1. **🎯 Weighted Distribution Implementation**: Main priority - fair capacity allocation based on client weights
2. **🏗️ Strategy Pattern Abstraction**: Extract balancing logic into pluggable strategies
3. **🔄 Architecture Refactor**: Move from global semaphore to per-client capacity management
4. **🧪 Testing**: Validate weighted distribution fairness and efficiency

## Active Development Patterns
- **Capacity Control**: Moving from global semaphore to proportional client quotas + spillover pool
- **Strategy Pattern**: BalancingStrategy interface with RoundRobin and Weighted implementations
- **Client Identification**: Extract client name from requests for capacity tracking
- **Fair Distribution**: Quota calculation: (clientWeight / totalActiveWeights) * totalCapacity

## Architecture Changes in Progress
- **ProxyServerPool**: Will use BalancingStrategy interface instead of direct capacity management
- **WeightedStrategy**: New proportional capacity allocation with spillover pool
- **RoundRobinStrategy**: Extract existing logic to maintain backward compatibility
- **Client Integration**: Hook quota recalculation into client registration/cleanup

## Next Priorities
1. **✅ Strategy Interface Design**: Define BalancingStrategy interface - COMPLETED
2. **✅ Extract Round-Robin**: Move existing logic to RoundRobinStrategy - COMPLETED
3. **⚖️ Implement Weighted**: Build proportional capacity allocation system
4. **🔗 Integration**: Wire strategies into main application flow
5. **🧪 Validation**: Test weighted distribution fairness

## Technical Implementation Details
- **Quota Calculation**: `quota = (clientWeight / totalActiveWeights) * totalCapacity`
- **Spillover Pool**: Unused capacity redistributed fairly among clients
- **Strategy Selection**: Build-time configuration, no runtime switching needed
- **Client Tracking**: Use existing auth.Client with weight field

## Technical Debt
- Need to extract balancing logic from ProxyServerPool
- Client identification mechanism needs implementation
- Comprehensive testing for weighted scenarios needed
- Memory bank documentation requires updates post-implementation

## Current Focus Areas
1. **✅ Performance Optimization**: String vs byte slice usage optimization completed
2. **Capacity Management**: Semaphore implementation refinement
3. **Testing**: Comprehensive benchmark suite development
4. **Observability**: Enhanced logging and metrics

## Active Development Patterns
- **Capacity Control**: Buffered channel semaphore pattern
- **Health Monitoring**: Concurrent goroutine per backend
- **Request Routing**: Atomic state management for thread safety
- **Error Handling**: Graceful degradation with clear client feedback
- **Performance**: Optimized byte slice usage throughout request/response pipeline

## Next Priorities
1. **Weighted Distribution**: Client-specific capacity allocation
2. **Benchmarking**: RPS, latency, and capacity efficiency testing
3. **Abstraction**: Pluggable balancer strategies
4. **Monitoring**: Structured logging with levels

## Technical Debt
- Hard-coded configuration values
- Limited error handling granularity
- Missing comprehensive test coverage
- No production-ready logging system

## Recent Performance Optimizations
- **Request Body Processing**: Direct byte slice handling instead of string conversions
- **Response Body Logging**: Added Write method to responseWriter for efficient capture
- **JSON Processing**: Eliminated unnecessary string→[]byte conversions
- **Buffer Operations**: Using Bytes() instead of String() method where possible

## Architecture Decisions Needed
- Balancer strategy abstraction design
- Weighted client implementation approach
- Metrics collection and exposure strategy 