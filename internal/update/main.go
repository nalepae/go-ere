// Command update refreshes the files this module copies from a release of github.com/eth-act/ere: the prebuilt
// verifier libraries, their header, the test fixtures and the licences. It also checks that the upstream Go binding
// is still the one verifier.go and verifier_test.go copy.
//
// Run it from the module root, after bumping ereVersion:
//
//	go run ./internal/update
//
// Every download is checked: a library archive against the sha256 digest GitHub records for the release asset, and
// every other file against its git blob hash at the release tag. Set GITHUB_TOKEN to avoid the API rate limit.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1" // #nosec G505 -- git blob hashes are SHA-1, used here to match upstream files, not for security.
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ereVersion is the ere release this module ships. The module is tagged with the same version.
const ereVersion = "0.20.2"

const (
	repo   = "eth-act/ere"
	header = "ere_verifier.h"
	lib    = "libere_verifier_c.a"
)

// targets maps the platform name of an ere release asset to its GOOS_GOARCH directory under lib/.
var targets = map[string]string{
	"linux-amd64":  "linux_amd64",
	"linux-arm64":  "linux_arm64",
	"darwin-arm64": "darwin_arm64",
}

// copies maps the upstream files copied verbatim to their path in this module.
var copies = map[string]string{
	"LICENSE-MIT":    "LICENSE-MIT",
	"LICENSE-APACHE": "LICENSE-APACHE",
}

// copiedSources maps the upstream Go files to their copy in this module. Only their header differs, up to and
// including the marker line. verifier_test.go is not listed: its fixture paths are adapted by hand.
var copiedSources = map[string]struct {
	path   string
	marker string
}{
	"bindings/golang/verifier.go": {"verifier.go", "*/\nimport \"C\"\n"},
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("update: ")

	tag := "v" + ereVersion
	if err := updateLibraries(tag); err != nil {
		log.Fatal(err)
	}

	for _, zkVM := range []string{"openvm", "sp1", "zisk", "lambdavm", "pico"} {
		for _, name := range []string{"program_vk.bin", "proof.bin", "public_values.bin"} {
			copies["crates/verifier/"+zkVM+"/tests/fixtures/"+name] = filepath.Join("testdata", zkVM, name)
		}
	}

	for upstream, local := range copies {
		content, err := fetchFile(tag, upstream)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(local, content); err != nil {
			log.Fatal(err)
		}
	}

	if err := checkBinding(tag); err != nil {
		log.Fatal(err)
	}

	log.Printf("ere %s is up to date", tag)
}

// updateLibraries downloads the verifier library archive of every target, checks it against its release digest,
// and extracts the library and the header, which every archive must hold identically.
func updateLibraries(tag string) error {
	var release struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
			URL    string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := getJSON("https://api.github.com/repos/"+repo+"/releases/tags/"+tag, &release); err != nil {
		return fmt.Errorf("release %s: %w", tag, err)
	}

	var commonHeader []byte
	for platform, dir := range targets {
		name := "libere_verifier_c." + platform + ".tar.gz"

		found := false
		for _, asset := range release.Assets {
			if asset.Name != name {
				continue
			}
			found = true

			archive, err := get(asset.URL)
			if err != nil {
				return fmt.Errorf("download %s: %w", name, err)
			}

			sum := sha256.Sum256(archive)
			if got := "sha256:" + hex.EncodeToString(sum[:]); got != asset.Digest {
				return fmt.Errorf("%s: digest %s, release says %s", name, got, asset.Digest)
			}

			files, err := untar(archive)
			if err != nil {
				return fmt.Errorf("extract %s: %w", name, err)
			}
			if files[lib] == nil || files[header] == nil {
				return fmt.Errorf("%s does not hold %s and %s", name, lib, header)
			}

			if commonHeader == nil {
				commonHeader = files[header]
			} else if !bytes.Equal(commonHeader, files[header]) {
				return fmt.Errorf("%s holds another %s than the other archives", name, header)
			}

			if err := write(filepath.Join("lib", dir, lib), files[lib]); err != nil {
				return err
			}
		}

		if !found {
			return fmt.Errorf("release %s has no asset %s", tag, name)
		}
	}

	return write(filepath.Join("include", header), commonHeader)
}

// checkBinding fails if an upstream Go file changed beyond the header this module rewrites, so that a bump does not
// silently keep an outdated binding.
func checkBinding(tag string) error {
	for upstream, local := range copiedSources {
		want, err := fetchFile(tag, upstream)
		if err != nil {
			return err
		}

		got, err := os.ReadFile(local.path)
		if err != nil {
			return err
		}

		if body(want, local.marker) != body(got, local.marker) {
			return fmt.Errorf("upstream %s changed: port it to %s", upstream, local.path)
		}
	}

	return nil
}

// body returns the content after the marker line.
func body(content []byte, marker string) string {
	text := string(content)
	if i := strings.Index(text, marker); i >= 0 {
		return strings.TrimLeft(text[i+len(marker):], "\n")
	}

	return text
}

// fetchFile downloads a file of the repository at the tag and checks it against its git blob hash.
func fetchFile(tag, path string) ([]byte, error) {
	var meta struct {
		SHA string `json:"sha"`
	}
	if err := getJSON("https://api.github.com/repos/"+repo+"/contents/"+path+"?ref="+tag, &meta); err != nil {
		return nil, fmt.Errorf("metadata of %s: %w", path, err)
	}

	content, err := get("https://raw.githubusercontent.com/" + repo + "/" + tag + "/" + path)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", path, err)
	}

	if got := blobHash(content); got != meta.SHA {
		return nil, fmt.Errorf("%s: blob hash %s, upstream says %s", path, got, meta.SHA)
	}

	return content, nil
}

func blobHash(content []byte) string {
	h := sha1.New() // #nosec G401 -- see the import.
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

func untar(archive []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	files := make(map[string][]byte)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		files[filepath.Base(hdr.Name)] = content
	}
}

func write(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func getJSON(url string, v any) error {
	content, err := get(url)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, v)
}

func get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && strings.HasPrefix(url, "https://api.github.com/") {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
