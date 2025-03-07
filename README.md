# Balancer
Distributes limited service capacity optimally between several clients based on a distribution strategy.

## Requirements
- docker
- docker-compose

## How to run
1. Start containers with `make up`
2. Simulate traffic with `make traffic`
3. Observe logs
4. Stop containers with `make down`

## Assignment
Imagine a standard client-server relationship, only in our case, the server is very fragile (and very expensive). We can't let an arbitrary number of clients smashing the server with arbitrary number of requests, but as it's so expensive to run, we'd like to utilize it as much as possible. And that's why we need a balancer.

A balancer is a gateway between the clients and the server. A client registers themselves in balancer and the balancer is responsible for distributing the service capacity between registered clients.

The balancer must ensure that at any given time, the number of parallel requests sent to the server does not exceed the provided limit.

Based on your skill, you can choose to implement a very simple balancer or production-ready balancer.

The implementation of a simple balancer can, for example, be one of:

A balancer that serves one client at a time, enqueuing others.
A balancer that serves registered clients in a round-robin fashion.
A balancer that is processing batches of registered clients and distributing the capacity among them, while enqueuing incoming clients and once one batch is done, process the next batch.
The ultimate, production balancer would run the requests from multiple clients in parallel according to their weights and process the work fairly from all the clients. A client with twice the weight of another client would be allowed to run twice more parallel requests at any given time.

In any case, the balancer must ensure that the number of requests in process at any given time equals the provided limit of the server, aka we never can over-utilise the server, but we also don't want to under-utilise it.

## TODOs
- abstraction for balancer type?
- add weights and batch processing
- add tests