package inscodeerr

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// CodeErr is an error that can be returned from the function wrapped by Err
// to control the HTTP status code returned from the pending request.
// Err is not necessary for now. However, we implemented it for later usages.
type CodeErr struct {
	Code    int
	Err     error
	Message string
}

func NewCodeErr(statusCode int, err error, message string) CodeErr {
	return CodeErr{
		Code:    statusCode,
		Err:     err,
		Message: message,
	}
}

// Error implements the error interface.
func (c CodeErr) Error() string {
	s := fmt.Sprintf("HTTP %d", c.Code)
	if t := http.StatusText(c.Code); t != "" {
		s += ": " + t
	}

	if c.Err != nil {
		s += ": " + c.Err.Error()
	}

	return s
}

type Response struct {
	Status  bool   `json:"status" example:"false"`
	Message string `json:"message"`
}

func (c CodeErr) MarshalJSON() ([]byte, error) {
	cer := Response{
		Status:  c.Code >= 200 && c.Code < 300,
		Message: c.Message,
	}

	res, err := json.Marshal(cer)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c CodeErr) StatusCode() int {
	return c.Code
}

func (c CodeErr) Headers() http.Header {
	return http.Header{}
}

func GetStatusCode(err error) int {
	e, ok := err.(CodeErr)
	if !ok {
		return http.StatusInternalServerError
	}

	return e.Code
}
