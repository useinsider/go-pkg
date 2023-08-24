# Simple Route Package

This package is designed to provide a simple way to create HTTP routes in your application. It provides a clean and flexible API for integrating into your codebase.

## Usage in Apps

```go
localeHealthHandler := echo.WrapHandler(
    inssimpleroute.NewServer(
        r.ucHandler.LocaleHealth,
        server.DecodeLocaleHealthRequest,
        server.EncodeResponse[dto.LocaleHealthResult],
        server.APIErrorEncoder,
    ),
)
g.GET("/locale-health", localeHealthHandler, common.WrapMiddlewares(r.mws)...)
```
