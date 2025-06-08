# Project Brief: Balancer

## Core Purpose
Load balancer service that manages limited backend capacity distribution across multiple clients while ensuring optimal utilization.

## Technical Requirements
- **Capacity Management**: Never exceed server capacity limits, never under-utilize available capacity
- **Client Registration**: Clients register with balancer before receiving service
- **Request Distribution**: Fair distribution strategies (round-robin, weighted)
- **Health Monitoring**: Continuous backend server health checks
- **Graceful Degradation**: Handle server failures transparently

## Current Implementation
- **Language**: Go 1.23.6
- **Architecture**: HTTP reverse proxy with semaphore-based capacity control
- **Backend Pool**: 3x WireMock servers (development)
- **Max Capacity**: 5 concurrent requests system-wide
- **Distribution**: Round-robin with health checking

## Success Criteria
- Zero request loss during normal operation
- Capacity limits strictly enforced
- Fair client access under load
- Sub-second response times
- Graceful failure handling 