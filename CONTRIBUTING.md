# Contributing to go-pkg

## Development

Each package is an independent Go module. Work within the package directory:

```bash
cd insredis
go mod tidy
go test ./...
```

## Releasing

See [RELEASING.md](RELEASING.md) for how to tag and release new versions.

**Quick reference:**
```bash
git tag <module>/v1.0.1
git push origin <module>/v1.0.1
gh release create <module>/v1.0.1 --title "<module> v1.0.1" --notes "Description"
```

## Module Dependencies

When updating these modules, release dependencies first:

| Module | Depends On |
|--------|------------|
| inssqs | insdash, inslogger |
| insssm | inscacheable |
