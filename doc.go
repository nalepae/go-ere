// Package ere verifies zkVM proofs with the verifiers of github.com/eth-act/ere,
// through its C library ere-verifier-c.
//
// The library is prebuilt and committed under lib/ for each supported platform:
// linux/amd64, linux/arm64 and darwin/arm64. It is linked statically, so
// `go build` needs a C compiler and nothing else. On any other platform, or
// without cgo, the package does not compile.
//
// A *Verifier may be used from multiple goroutines for concurrent Verify and
// Kind calls. Close waits for the calls in progress on the same Verifier, and
// a later call returns [ErrNullPtr].
//
// # Public values
//
// A proof that verifies shows only that the program ran. The caller must
// compare the public values that Verify returns with the expected bytes.
//
//   - OpenVM and ZisK return 256 bytes, the guest output padded with zeros.
//   - LambdaVM, Pico and SP1 return the guest output.
package ere
