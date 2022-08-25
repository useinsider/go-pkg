# Cacheable Package


This is a memory level cache wrapper. You just need to give it a function, and it will handle caching for you.

## Usage in Apps
```go
package main

import (
	"time"
	
	"github.com/useinsider/go-pkg/inscacheable"
)

var ttl = 1 * time.Minute
var cache = inscacheable.Cacheable(somefunctocache, &ttl)

func somefunctocache(anyparam interface{}) interface{} {
	return nil
}
```
