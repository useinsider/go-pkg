# insredis

The insredis package provides a universal Golang Redis interface and a client for easy interaction with Redis databases.

## Overview

This package offers a set of methods covering various functionalities for working with Redis, encapsulated within the
RedisInterface. It includes methods for key-value operations, set operations, list operations, sorted set operations,
geospatial operations, and more.

## Installation

To use this package, install it using Go modules:

```bash
go get github.com/useinsider/go-pkg/insredis
```

## Usage

### Initialization

Initialize a Redis client instance by providing configuration settings using `Init()`

```go
import (
"github.com/yourusername/insredis"
"time"
)

func main() {
// Configure Redis settings
cfg := insredis.Config{
RedisHost:     "localhost:6379",
RedisPoolSize: 10,
DialTimeout:   500 * time.Millisecond,
ReadTimeout:   500 * time.Millisecond,
MaxRetries:    3,
}

// Initialize the Redis client
client := insredis.Init(cfg)

// Use the client for Redis operations
// e.g., client.Set("key", "value", 0)
}




````

// Default configs in table format

| Config Name   | Default Value  | Description                                                       |
|---------------|----------------|-------------------------------------------------------------------|
| RedisHost     | localhost:6379 | Redis host address                                                |
| RedisPoolSize | 10             | Maximum number of connections allocated by the Redis client       |
| DialTimeout   | 500ms          | Maximum amount of time to wait for a connection to be established |
| ReadTimeout   | 500ms          | Maximum amount of time to wait for a read operation to complete   |
| MaxRetries    | 0 (no retry)   | Maximum number of retries before giving up on a request           |

## Usage in Tests

```go
package main

import (
	"github.com/go-redis/redis"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go-pkg/insredis"
	"testing"
)

func TestUsingRedisMock(t *testing.T) {
	controller := gomock.NewController(t)
	redisClient := insredis.NewMockRedisInterface(controller)

	redisClient.
		EXPECT().
		Ping().
		Times(1).
		DoAndReturn(func() *redis.StatusCmd {
			return redis.NewStatusCmd("PONG")
		})

	err := ping(redisClient)
	assert.NoError(t, err)
}

func ping(redisClient insredis.RedisInterface) error {
	return redisClient.Ping().Err()
}
```

## How to Update Mock File

```
mockgen -source=./insredis/redis.go -destination=./insredis/redis_mock.go -package=insredis
```

## Contributing

Contributions to this package are welcome! Feel free to submit issues or pull requests.

