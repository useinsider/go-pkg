# Releasing New Versions

Each module is versioned independently using tags: `<module>/v<version>`

## Release a Module

```bash
# 1. Commit your changes
git add .
git commit -m "inslogger: description of change"
git push

# 2. Create and push tag
git tag inslogger/v1.0.1
git push origin inslogger/v1.0.1

# 3. Create GitHub release
gh release create inslogger/v1.0.1 --title "inslogger v1.0.1" --notes "Description of changes"
```

## Module Dependencies

Release these first if updating dependent modules:

- `inssqs` depends on: `insdash`, `inslogger`
- `insssm` depends on: `inscacheable`
