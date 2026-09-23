# Git hooks

Enable the tracked hooks for this clone:

```bash
git config core.hooksPath .githooks
```

The pre-push hook runs the complete Go test suite with `go test ./...` and
blocks the push if a test fails.
