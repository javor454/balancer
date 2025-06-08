# Active Context: Balancer

## Current State
- **Phase**: Phase 1 Complete - Weighted client distribution implemented with event-driven quota management
- **Status**: WeightedStrategy fully functional with capacity overflow prevention and concurrency safety
- **Last Focus**: Simplified capacity release to avoid pool tracking complexity
- **Current Implementation**: Basic weighted balancing with spillover pool (Option 2 approach)

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
- **WeightedStrategy with per-client capacity quotas and spillover pool**
- **Client identification enhancement with context injection**
- **Comprehensive unit tests for weighted distribution logic**
- **Event-driven quota recalculation (instead of periodic timer)**
- **Concurrency-safe quota updates preserving current usage**
- **Capacity overflow prevention during quota recalculation**
- **~~Intelligent quota rebalancing on capacity release for fairness~~** - REMOVED (Option 2: simplified approach)
- **~~Optimized recalculation triggers to avoid performance overhead~~** - REMOVED (flawed pool tracking)
- **Simplified capacity release without automatic quota rebalancing** - CURRENT APPROACH

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
3. **✅ Implement Weighted**: Build proportional capacity allocation system - COMPLETED
4. **🔗 Integration**: Wire strategies into main application flow
5. **🧪 Validation**: Test weighted distribution fairness

## Future Implementation Plans

### Option 1: Proper Capacity Tracking (Phase 1 Enhancement)
**Goal**: Implement perfect capacity pool tracking to enable accurate release-based rebalancing
**Status**: Documented, not implemented (current system works with simplified approach)

**Required Steps:**
1. **Token-Based Capacity Tracking**:
   - Create `CapacityToken` struct with `ClientName` and `PoolType` fields
   - Modify `AcquireCapacity()` to return token instead of error-only
   - Update `ReleaseCapacity()` to accept token parameter

2. **Pool Source Tracking**:
   - Track which pool (client quota vs spillover) provided each token
   - Enable accurate return of capacity to correct pool
   - Support proper fairness algorithms based on actual usage patterns

3. **Enhanced Rebalancing**:
   - Re-implement `shouldRecalculateAfterRelease()` with accurate pool tracking
   - Add release-based quota rebalancing for gradual fairness improvement
   - Implement tests for gradual capacity redistribution scenarios

### Phase 2: Integration & Configuration
**Goal**: Wire WeightedStrategy into main application with proper configuration

**Required Steps:**
1. **Strategy Selection Configuration**:
   - Add command-line flag for balancing strategy selection (`--strategy=weighted|round-robin`)
   - Create strategy factory function in `main.go`
   - Configure strategy-specific parameters (capacity, timeouts)

2. **ProxyServerPool Integration**:
   - Update `ProxyServerPool.handleRequest()` to use strategy pattern
   - Replace direct capacity management with strategy delegation
   - Ensure backwards compatibility with existing round-robin behavior

3. **Client Registration Hooks**:
   - Add event hooks in `AuthHandler.RegisterClient()` to trigger quota recalculation
   - Add event hooks in `AuthHandler.cleanupClients()` for quota cleanup
   - Implement graceful strategy switching (if needed)

### Phase 3: Advanced Weighted Features
**Goal**: Enhance weighted balancing with sophisticated capacity management

**Required Steps:**
1. **Dynamic Weight Updates**:
   - API endpoint for runtime weight modification (`PUT /clients/{name}/weight`)
   - Real-time quota recalculation on weight changes
   - Validation and safety checks for weight updates

2. **Capacity Borrowing System**:
   - Allow high-priority clients to temporarily exceed quota during low usage
   - Implement capacity reclamation when lower-priority clients need resources
   - Add borrowing limits and policies

3. **Priority-Based Spillover**:
   - Replace simple spillover pool with priority-ordered spillover allocation
   - Higher-weight clients get first access to unused capacity
   - Implement spillover capacity redistribution algorithms

### Phase 4: Observability & Production Readiness
**Goal**: Add comprehensive monitoring, metrics, and production-grade features

**Required Steps:**
1. **Metrics & Monitoring**:
   - Prometheus metrics for quota utilization per client
   - Capacity acquisition/release rate tracking
   - Spillover pool usage statistics
   - Strategy performance metrics (latency, fairness distribution)

2. **Health Checks & Diagnostics**:
   - Health check endpoint showing quota status per client
   - Diagnostic API for quota allocation debugging
   - Capacity utilization reports and alerts

3. **Configuration Management**:
   - External configuration file support (YAML/JSON)
   - Hot-reload of non-critical configuration changes
   - Configuration validation and error handling

4. **Resilience Features**:
   - Circuit breaker integration for quota management failures
   - Graceful degradation when quota system encounters errors
   - Backup strategy fallback (round-robin) during quota system issues

## Technical Implementation Details
- **Quota Calculation**: `quota = (clientWeight / totalActiveWeights) * totalCapacity`
- **Spillover Pool**: Unused capacity redistributed fairly among clients
- **Strategy Selection**: Build-time configuration, no runtime switching needed
- **Client Tracking**: Use existing auth.Client with weight field

## Technical Debt
- ✅ ~~Need to extract balancing logic from ProxyServerPool~~ - COMPLETED
- ✅ ~~Client identification mechanism needs implementation~~ - COMPLETED  
- ✅ ~~Comprehensive testing for weighted scenarios needed~~ - COMPLETED
- **Capacity Pool Tracking**: Current system doesn't track which pool provided capacity (see Option 1 above)
- **Event Hooks**: Need client registration/deregistration hooks in AuthHandler for quota updates
- **Strategy Integration**: Strategy selection configuration in main application flow
- **Perfect Fairness**: Current approach prioritizes stability over perfect weight-based fairness

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