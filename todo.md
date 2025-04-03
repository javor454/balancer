Semaphore-based Request Limiting
Concept: Use a semaphore to control the maximum number of concurrent requests
Implementation Ideas:
Add a semaphore (using a buffered channel in Go) to the ProxyServerPool
Before forwarding a request, acquire a token from the semaphore
After the request completes, release the token back to the semaphore
If no tokens are available, either queue the request or return a "service busy" response


Throughput Testing: Requests per second at different concurrency levels
Latency Testing: Response time distribution under various loads
Capacity Limit Validation: Verify behavior when capacity is reached

Key Metrics to Track
Requests Per Second (RPS): Maximum sustainable throughput
Latency Distribution: p50, p95, p99 percentiles
Error Rates: Percentage of 429 (capacity exceeded) responses
Resource Utilization: CPU, memory, network during peak loads
Capacity Efficiency: How effectively you utilize your configured capacity
By implementing these benchmarks, you'll gain valuable insights into your load balancer's performance characteristics and ensure it can reliably handle the expected traffic while maintaining the configured capacity limits.