package inskinesis

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"net"
	"strings"
)

// CustomRetryer retries on "connection reset by peer"
type CustomRetryer struct {
	request.Retryer
}

func (r CustomRetryer) ShouldRetry(req *request.Request) bool {
	if err, ok := req.Error.(net.Error); ok && err.Timeout() {
		return true
	}

	if opErr, ok := req.Error.(*net.OpError); ok && strings.Contains(opErr.Err.Error(), "connection reset by peer") {
		return true
	}

	return r.Retryer.ShouldRetry(req)
}
