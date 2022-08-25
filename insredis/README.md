# Redis Package


This is a simple mockable wrapper for Redis.

## Usage in Apps
```go
package main

import "github.com/useinsider/go-pkg/insredis"

func main() {
	redisClient := insredis.Init("localhost:6379", 100)

	err := redisClient.Ping()
	if err != nil {
		panic(err)
	}
}
```

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
