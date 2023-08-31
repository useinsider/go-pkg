# Simple Route Package

This package is designed to provide a simple way to create HTTP routes in your application. It provides a clean and flexible API for integrating into your codebase.

## Usage in Apps

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/useinsider/inssimpleroute"
)

// Request type for the use case
type GreetingRequest struct {
	Name string `json:"name"`
}

// Response type for the use case
type GreetingResponse struct {
	Message string `json:"message"`
}

// GreetingUseCase is the core business logic for generating a greeting message
func GreetingUseCase(ctx context.Context, request *GreetingRequest) (*GreetingResponse, error) {
	// Generate a greeting message
	greeting := fmt.Sprintf("Hello, %s!", strings.TrimSpace(request.Name))
	response := &GreetingResponse{
		Message: greeting,
	}
	return response, nil
}

// GreetingRequestDecoder is a custom decoder for GreetingRequest
func GreetingRequestDecoder(_ context.Context, r *http.Request) (*GreetingRequest, error) {
	var request GreetingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return &request, nil
}

func main() {
	// Create a new server route with the GreetingUseCase and a request decoder
	server := inssimpleroute.NewServerWithDefaults(GreetingUseCase, GreetingRequestDecoder)

	// Register the server route
	http.Handle("/greet", server)

	// Start the HTTP server
	port := 8080
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server listening on %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
}

```
