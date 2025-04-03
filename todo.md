Semaphore-based Request Limiting
Concept: Use a semaphore to control the maximum number of concurrent requests
Implementation Ideas:
Add a semaphore (using a buffered channel in Go) to the ProxyServerPool
Before forwarding a request, acquire a token from the semaphore
After the request completes, release the token back to the semaphore
If no tokens are available, either queue the request or return a "service busy" response