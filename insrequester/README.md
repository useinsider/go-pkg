# Requester Package

The insrequester package is designed to provide a resilient way to make HTTP requests while incorporating various features like retry, circuit breaking, and timeout handling. This package aims to simplify the process of making HTTP requests by handling common scenarios like network failures, high request loads, and service timeouts, all while providing a clean and flexible API for integrating into your codebase.


## Usage in Apps

```go
import (
    "github.com/useinsider/go-pkg/insrequester"
)
```

### Creating a Requester
To create a new instance of the Requester, you can use the `NewRequester()` function:

```go
requester := insrequester.NewRequester()
```

### Making a Request
The `Requester` interface provides methods for making various HTTP requests, such as GET, POST, PUT, and DELETE. Here's an example of making a GET request:

```go
requestEntity := insrequester.RequestEntity{
    Endpoint: "https://api.example.com/resource",
}

response, err := requester.Get(requestEntity)
if err != nil {
    // Handle the error
} else {
    // Process the response
}
```

### Adding Resilience Features
#### Retry
You can add retry functionality to your requests by chaining the WithRetry method to the Requester:

```go
retryConfig := insrequester.RetryConfig{
    WaitBase: time.Millisecond * 200,
    Times:    3,
}
requester.WithRetry(retryConfig)
```
#### Circuit Breaker
To implement a circuit breaker pattern, use the WithCircuitbreaker method:

```go
circuitBreakerConfig := insrequester.CircuitBreakerConfig{
    MinimumRequestToOpen:         3,
    SuccessfulRequiredOnHalfOpen: 1,
    WaitDurationInOpenState:      5 * time.Second,
}

requester.WithCircuitbreaker(circuitBreakerConfig)
```
#### Timeout
For setting a timeout on requests, you can utilize the WithTimeout method:

```go
timeoutSeconds := 30
requester.WithTimeout(timeoutSeconds) // this timeout overrides the default timeout
```

#### Default Headers
For applying default headers to all requests, you can use the WithDefaultHeaders method:

```go
headers := insrequester.Headers{{"Authorization": "Bearer token"}}
requester.WithHeaders(headers)
```
It should be noted that you can still override these default headers by providing the same header in the request entity.


### Loading Middlewares
After configuring the desired resilience features, load the configured middlewares using the Load method:

```go
requester.Load()
```
