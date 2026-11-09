# AGENTS.md

Guidance for AI coding agents (and humans) working in this repository.

## Repository shape

This is a Go workspace: `go.work` lists independent modules (`gin`, `grpc`,
`http`, `otel`, `resty`, `task`, `test`). The `test` module is an integration
consumer of the others, so package patterns like `./...` do not work from the
repository root — use `github.com/LinPr/go-bootstrap/...` or a Taskfile task.

`main` carries every module. `go-1.23` carries the same set except `gin`,
which is never in its `go.work`; `resty` and `task` are present there and build
on Go 1.23.

## Branches target different Go versions

Two long-lived branches are maintained, each pinned to a different Go release.
Always resolve the toolchain from the checked-out branch, never from memory.

| Branch | Go version | Notes |
| ------ | ---------- | ----- |
| `main` | 1.27.x | development branch; `go.work` includes every module |
| `go-1.23` | 1.23.0 | maintenance branch; `gin` is not in `go.work` |

The authoritative value is the `go` directive in `go.work` — read it rather
than hard-coding a version.

## Use Taskfile tasks, not raw go commands

`Taskfile.yml` reads that directive and runs every command with the matching
toolchain, so the branch decides the Go version automatically. Tasks work on a
machine with no Go version manager: when no local binary matches, they fall
back to `GOTOOLCHAIN=go<version>`, which downloads the release.

```bash
task env        # branch, required Go version, actual toolchain
task build      # build every workspace module
task vet        # vet every workspace module
task test       # all tests, including the integration module
task test:unit  # library modules only
task tidy       # go mod tidy in each module, with the branch toolchain
task ci         # vet + test
```

Rules for agents:

1. Prefer the Taskfile over hand-written `go` invocations: `task build`,
   `task vet`, and `task test:unit` need no environment setup.
2. If you must run `go` directly, confirm `task env` first. A version mismatch
   is a hard error, not a warning.
3. Only run `task tidy` when the dependency graph actually changed; it rewrites
   `go.mod`/`go.sum` and the result differs per Go release.
4. Do not hard-code a Go version in commands or docs — read `go.work`.

## Known pitfalls

- Code shared with `go-1.23` must not use APIs added in **Go 1.24**, notably
  `t.Context()`, `t.Chdir()`, and `t.Output()`. They do not compile there, so
  `task vet` and `task test` fail with `t.Context undefined`. Use
  `context.Background()` in test files that must build on both branches; do not
  bump the branch's Go version to work around it.
- Likewise avoid **Go 1.25** APIs such as `sync.WaitGroup.Go`. Use
  `wg.Add(1)` plus `go func() { defer wg.Done(); ... }()` instead.
- `task tidy` only visits modules listed in `go.work`, so it skips `gin` on
  `go-1.23`. Tidy `gin` by hand if it changes.
- `go.work.sum` is not gitignored; commit it whenever the toolchain generates or
  updates it. It is tracked on both branches and its contents differ between
  them, so never copy it across a branch switch.
- `otel/go.mod` on `go-1.23` pins `github.com/prometheus/otlptranslator` to
  `v0.0.2`, the version matching `go.opentelemetry.io/otel/exporters/prometheus`
  `v0.60.0`.
- The Prometheus exporter is created with `otlptranslator.NoTranslation`, so
  metric names are exported verbatim. Do not remove that option; it is not the
  default and exported names change silently without it.

## Commits

Conventional Commits with a module scope: `fix(otel): ...`, `docs(gin): ...`.
Keep independent modules' changes in separate commits.
