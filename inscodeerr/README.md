# CodeErr Package


This provides an error type called codeerr. This can be used with simpleroute package for error responses.

## Usage in Apps
```go
res, err := sc.getPartner(partnerName)

if err != nil {
    return nil, inscodeerr.NewCodeErr(http.StatusUnauthorized, err, uc.ErrPartnerNotFound.Error())
}
```
