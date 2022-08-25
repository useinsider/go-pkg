package insssm

import (
	awsssm "github.com/Jamil-Najafov/go-aws-ssm"
	"github.com/useinsider/go-pkg/inscacheable"
	"os"
	"time"
)

// ParameterStore is the aws client for parameter store
var ParameterStore *awsssm.ParameterStore

// ttl value is used to tell cacheable to long it should cache the value.
var ttl = 1 * time.Minute

//cache is the instance of cacheable wrapped ssm get function.
var cache = inscacheable.Cacheable(get, &ttl)

// Init
// If environment variable called "ENV" is set to value "LOCAL", Init function will not run.
func Init() {
	if os.Getenv("ENV") != "LOCAL" {
		pmStore, err := awsssm.NewParameterStore()
		if err != nil {
			panic(err)
		}

		ParameterStore = pmStore
	}
}

// Get is the main function for ssm, it takes the key and returns the value.
// It's wrapped with cacheable function so if it already called and ttl is still valid,
// it will use the cached value to return instead of reaching the aws servers.
func Get(key string) string {
	return cache.Get(key)
}

// get is internal function to getting the value from aws.
func get(key string) string {
	params, err := ParameterStore.GetParameter(key, true)
	if err != nil {
		panic(err)
	}

	return params.GetValue()
}
