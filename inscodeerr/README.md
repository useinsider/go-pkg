# CodeErr Package


`inscodeerr` is a Go package that provides a custom error type called CodeErr designed to handle HTTP status codes and messages for errors returned from functions. It allows you to control the HTTP status code and associated message for a given error, making it easier to handle and convey errors within your application.


## Usage

### Creating a CodeErr
To create a new CodeErr instance, you can use the NewCodeErr function. This function takes three arguments: the desired HTTP status code, an error instance, and a custom error message.

```go
err := inscodeerr.NewCodeErr(http.StatusBadRequest, fmt.Errorf("validation error"), "Invalid input data")
```
### Getting Status Code and Error Message
You can retrieve the HTTP status code and error message from a CodeErr instance using its fields:

```go
statusCode := err.Code
errorMessage := err.Message
```
### JSON Serialization
The CodeErr instances can be easily serialized to JSON format using the MarshalJSON method. The serialized JSON will include a status field indicating whether the HTTP status code represents a success status (2xx range) or not, and a message field containing the custom error message.

```go
jsonBytes, _ := json.Marshal(err)
```

### Checking Status Code
You can also check if a given error is a CodeErr and get its associated status code using the GetStatusCode function:

```go
status := inscodeerr.GetStatusCode(err)
```
### Using in HTTP Handlers
You can use CodeErr instances within your HTTP handlers to control the HTTP response status code and message. For example:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
    err := processRequest(r)
    if err != nil {
        if codeErr, ok := err.(inscodeerr.CodeErr); ok {
            http.Error(w, codeErr.Message, codeErr.Code)
            return
        }
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
    }
    // Handle successful request
}
```
