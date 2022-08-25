# SSM Package


This is a simple aws ssm packet wrapper.

Currently, we use env variables to disable ssm client initialization for local development and testing purposes.


## Usage in Apps
```go
package main

import (
	"fmt"
	"os"
	
	"github.com/useinsider/go-pkg/insssm"
)

func main() {
	_ = os.Setenv("ENV", "NOT-LOCAL")
	insssm.Init()
	
	value := insssm.Get("/SOME/SSM/KEY")
	fmt.Printf("value: %s", value)
}


```
