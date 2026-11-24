# go-ere

Go bindings for the zkVM proof verifiers of [ere], usable as a plain Go
dependency: `go get` and `go build`, with nothing to download or configure.

```go
import ere "github.com/nalepae/go-ere"

verifier, err := ere.New(ere.SP1, programVK)
if err != nil {
    return err
}
defer verifier.Close()

publicValues, err := verifier.Verify(proof)
```

```
go get github.com/nalepae/go-ere
```

## Why this exists

ere is written in Rust. Its verifiers are exposed to C as a static library,
`libere_verifier_c.a`, which ere publishes prebuilt for each platform on its
releases page. ere's own Go binding (`bindings/golang`) is a thin cgo wrapper
around that library, but it doesn't ship it. Each consumer has to download the
right archive, extract it somewhere, and point `CGO_CFLAGS` and `CGO_LDFLAGS`
at it on every build.

A Go module can't build Rust, but it can carry the prebuilt library. This module
commits ere's release libraries under `lib/<goos>_<goarch>/` and links them
through `${SRCDIR}`, so the library comes with the module like any Go source.
The approach is the one [wasmtime-go] takes.

The Go checksum database pins the module's content, libraries included, so
`go.sum` gives the same tamper protection as a pinned sha256 of the archive.

## Supported platforms

| GOOS/GOARCH | |
| --- | --- |
| linux/amd64 | glibc |
| linux/arm64 | glibc |
| darwin/arm64 | |

These are the platforms ere publishes a library for. cgo is required. On any
other platform, or with `CGO_ENABLED=0`, the package fails to compile with an
error naming the supported platforms, instead of a link error.

A consumer that must still build elsewhere should import this package only from
files behind a build tag of its own, with a stub for the other builds.

The Linux libraries link against glibc. They don't link with musl (Alpine).

## Versioning

The module version is the ere version it ships: `v0.17.2` holds ere v0.17.2.

## Updating

Bump `ereVersion` in `internal/update/main.go`, then:

```
GITHUB_TOKEN=$(gh auth token) go run ./internal/update
```

The tool:

- downloads each platform's archive and checks it against the sha256 digest
  GitHub records for the release asset;
- checks that every archive holds the same header;
- copies the test fixtures and licences, each checked against its git blob hash
  at the release tag;
- fails if ere's Go binding changed, so that `verifier.go` is ported by hand
  instead of silently going stale.

CI runs the same tool and fails if anything differs from what is committed. So
every committed library is the one ere released.

`verifier.go` and `verifier_test.go` are copied from ere's `bindings/golang`.
Only their package name, build constraint, cgo flags and fixture paths differ.

## Size

The libraries are large: about 17 MB each for Linux and 32 MB for darwin
(about 25 MB for the whole module, compressed). Each ere bump adds them to the
git history again, and GitHub warns about files over 50 MB. Consumers download
the module once per version.

## Licence

ere, and therefore this module, is distributed under either of `LICENSE-MIT` or
`LICENSE-APACHE`, at your option.

[ere]: https://github.com/eth-act/ere
[wasmtime-go]: https://github.com/bytecodealliance/wasmtime-go
