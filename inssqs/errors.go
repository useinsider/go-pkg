package inssqs

import "github.com/pkg/errors"

var ErrRetryCountExceeded = errors.New("retry count exceeded")
var ErrRegionNotSet = errors.New("region not set")
var ErrQueueNameNotSet = errors.New("queue name not set")
