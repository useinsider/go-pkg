package inskinesis

import (
	"errors"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go/aws/request"
)

// CustomRetryer retries on "connection reset by peer"
type CustomRetryer struct {
	request.Retryer
}

func (r CustomRetryer) ShouldRetry(req *request.Request) bool {
	var netErr net.Error
	if errors.As(req.Error, &netErr) && netErr.Timeout() {
		return true
	}

	var opErr *net.OpError
	if errors.As(req.Error, &opErr) && strings.Contains(opErr.Err.Error(), "connection reset by peer") {
		return true
	}

	return r.Retryer.ShouldRetry(req)
}
