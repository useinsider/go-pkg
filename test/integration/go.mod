// Its own module because go-pkg has no root module — every package here is a
// separately released module. replace directives point at the working tree so
// the integration test always exercises the code in this checkout, never a
// published tag.
module github.com/useinsider/go-pkg/test/integration

go 1.25.0

require (
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/useinsider/go-pkg/inscacheable v1.0.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

require (
	github.com/golang/mock v1.6.0 // indirect
	github.com/jellydator/ttlcache/v3 v3.0.0 // indirect
	github.com/useinsider/go-pkg/insredis v1.0.0
	go.uber.org/goleak v1.2.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220908164124-27713097b956 // indirect
)

replace github.com/useinsider/go-pkg/inscacheable => ../../inscacheable

replace github.com/useinsider/go-pkg/insredis => ../../insredis
