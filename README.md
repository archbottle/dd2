# dd2

This repo contains:

- `php/`: **git submodule** containing upstream [`matomo-org/device-detector`](https://github.com/matomo-org/device-detector) (ported/vendor code).
- `go/`: Go implementation + tooling that consumes the upstream YAML regexes.

### Structure

```
dd2/
├── go/                 # Go port + tooling
└── php/                # Upstream device-detector (submodule)
```
