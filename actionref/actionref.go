// Package actionref resolves an action that a step's `uses` names outside
// Probe to the executable that serves it.
//
// Such an action is published in a GitHub repository and pinned to a commit:
//
//	uses: github.com/<owner>/<repo>[/<dir>]@<40-character commit SHA>
//
// or lives in a local directory:
//
//	uses: ./path/to/action
//
// Either way the directory holds an action.yml that says which executable
// serves the action. The executable speaks the protocol in package actionrpc.
package actionref

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
)

// ManifestFile is the file in an action's directory that describes it.
const ManifestFile = "action.yml"

// UsingBinary is the only kind of action an action.yml can describe: an
// executable that serves the action over the actionrpc protocol.
const UsingBinary = "binary"

const githubHost = "github.com"

var (
	commitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)
	pathPart  = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
)

// IsExternal reports whether uses names an action outside Probe. No built-in
// action has a slash in its name.
func IsExternal(uses string) bool {
	return strings.Contains(uses, "/")
}

// Ref is a parsed `uses` that names an external action.
type Ref struct {
	// Local is the directory of a local action, as written. The other fields
	// are empty when it is set.
	Local string
	// Owner and Repo name the GitHub repository of a remote action.
	Owner string
	Repo  string
	// Dir is the directory of the action within the repository, slash
	// separated, or empty for the repository root.
	Dir string
	// SHA is the commit the action is pinned to.
	SHA string
}

// Parse parses uses as a reference to an external action.
func Parse(uses string) (Ref, error) {
	if isLocal(uses) {
		return Ref{Local: uses}, nil
	}

	rest, ok := strings.CutPrefix(uses, githubHost+"/")
	if !ok {
		return Ref{}, fmt.Errorf("unsupported action %q: use github.com/<owner>/<repo>@<commit SHA> or a local path starting with ./", uses)
	}

	repoPath, sha, ok := strings.Cut(rest, "@")
	if !ok || sha == "" {
		return Ref{}, fmt.Errorf("action %q must be pinned to a commit: append @<40-character commit SHA>", uses)
	}
	if !commitSHA.MatchString(sha) {
		return Ref{}, fmt.Errorf("action %q must be pinned to a full 40-character commit SHA, not %q: tags and branches can be moved", uses, sha)
	}

	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return Ref{}, fmt.Errorf("action %q must name a repository as github.com/<owner>/<repo>", uses)
	}
	for _, p := range parts {
		if !pathPart.MatchString(p) || p == "." || p == ".." {
			return Ref{}, fmt.Errorf("action %q has an invalid path element %q", uses, p)
		}
	}

	return Ref{
		Owner: parts[0],
		Repo:  parts[1],
		Dir:   strings.Join(parts[2:], "/"),
		SHA:   sha,
	}, nil
}

func isLocal(uses string) bool {
	return strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "../") || filepath.IsAbs(uses)
}

// Manifest is the content of an action.yml.
type Manifest struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Runs        Runs   `yaml:"runs"`
}

// Runs says how an action is run.
type Runs struct {
	// Using is the kind of action. It must be "binary".
	Using string `yaml:"using"`
	// URL is where the executable is downloaded from. {os} and {arch} are
	// replaced with the GOOS and GOARCH of the machine running Probe.
	URL string `yaml:"url"`
	// Path is the executable relative to the action's directory, with the
	// same placeholders as URL. Only a local action can use it.
	Path string `yaml:"path"`
	// Checksums maps "<os>_<arch>" to the SHA-256 digest of the executable
	// for that platform, in hex. A URL needs one for the running platform.
	Checksums map[string]string `yaml:"checksums"`
}

// ParseManifest parses the content of an action.yml.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Runs.Using != UsingBinary {
		return nil, fmt.Errorf("runs.using must be %q, not %q", UsingBinary, m.Runs.Using)
	}
	if (m.Runs.URL == "") == (m.Runs.Path == "") {
		return nil, errors.New("runs must have exactly one of url and path")
	}
	for platform, sum := range m.Runs.Checksums {
		if !sha256Hex.MatchString(sum) {
			return nil, fmt.Errorf("runs.checksums.%s must be a SHA-256 digest in lowercase hex", platform)
		}
	}
	return &m, nil
}

// Executable is the program that serves an action.
type Executable struct {
	// Path is the executable's location on this machine.
	Path string
	// SHA256 is the digest the executable must have when it is started, or
	// empty for a local action that does not pin one.
	SHA256 []byte
}

// Resolver finds and downloads the executables of external actions. A
// Resolver remembers what it resolved, so an action is fetched once however
// many steps use it. Its zero value is ready to use.
type Resolver struct {
	// CacheDir is where downloaded actions are kept. It defaults to probe/actions
	// under the user's cache directory.
	CacheDir string
	// Client fetches action.yml files and executables. It defaults to a client
	// with a five-minute timeout.
	Client *http.Client
	// RawBaseURL is where file contents of a GitHub repository are served. It
	// defaults to https://raw.githubusercontent.com.
	RawBaseURL string
	// OS and Arch are the platform to resolve for. They default to the
	// platform Probe runs on.
	OS   string
	Arch string

	mu   sync.Mutex
	memo map[string]*resolution
}

type resolution struct {
	once sync.Once
	exe  *Executable
	err  error
}

var defaultResolver = &Resolver{}

// Resolve resolves uses with a Resolver shared by the whole process.
func Resolve(uses, baseDir string) (*Executable, error) {
	return defaultResolver.Resolve(uses, baseDir)
}

// Resolve returns the executable of the external action uses names. A local
// action's path is taken relative to baseDir, or to the working directory
// when baseDir is empty.
func (r *Resolver) Resolve(uses, baseDir string) (*Executable, error) {
	ref, err := Parse(uses)
	if err != nil {
		return nil, err
	}

	key := uses
	if ref.Local != "" {
		dir := ref.Local
		if !filepath.IsAbs(dir) && baseDir != "" {
			dir = filepath.Join(baseDir, dir)
		}
		if dir, err = filepath.Abs(dir); err != nil {
			return nil, err
		}
		ref.Local = dir
		key = dir
	}

	r.mu.Lock()
	if r.memo == nil {
		r.memo = map[string]*resolution{}
	}
	res, ok := r.memo[key]
	if !ok {
		res = &resolution{}
		r.memo[key] = res
	}
	r.mu.Unlock()

	res.once.Do(func() {
		if ref.Local != "" {
			res.exe, res.err = r.resolveLocal(ref.Local)
		} else {
			res.exe, res.err = r.resolveRemote(ref)
		}
		if res.err != nil {
			res.err = fmt.Errorf("action %s: %w", uses, res.err)
		}
	})
	return res.exe, res.err
}

func (r *Resolver) resolveLocal(dir string) (*Executable, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		return nil, err
	}
	m, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestFile, err)
	}

	if m.Runs.URL != "" {
		return r.download(m)
	}

	exe := &Executable{Path: filepath.Join(dir, filepath.FromSlash(r.expand(m.Runs.Path)))}
	got, err := fileSHA256(exe.Path)
	if err != nil {
		return nil, err
	}
	if sum, ok := m.Runs.Checksums[r.platform()]; ok {
		if exe.SHA256, _ = hex.DecodeString(sum); !bytes.Equal(got, exe.SHA256) {
			return nil, fmt.Errorf("%s has SHA-256 %x, but %s says %s", exe.Path, got, ManifestFile, sum)
		}
	}
	return exe, nil
}

func (r *Resolver) resolveRemote(ref Ref) (*Executable, error) {
	cacheDir, err := r.cacheDir()
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(cacheDir, githubHost, ref.Owner, ref.Repo, ref.SHA, filepath.FromSlash(ref.Dir), ManifestFile)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		url := strings.TrimSuffix(r.rawBaseURL(), "/") + "/" + path.Join(ref.Owner, ref.Repo, ref.SHA, ref.Dir, ManifestFile)
		if data, err = r.fetch(url); err != nil {
			return nil, err
		}
		if err := writeFileAtomic(manifestPath, bytes.NewReader(data), 0o644); err != nil {
			return nil, err
		}
	}

	m, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestFile, err)
	}
	if m.Runs.Path != "" {
		return nil, fmt.Errorf("%s: runs.path is only for local actions; a remote action needs runs.url and runs.checksums", ManifestFile)
	}
	return r.download(m)
}

// download fetches the executable that m points to, unless an executable
// with the expected digest is already in the cache. Executables are cached by
// digest, so actions that ship the same build share it.
func (r *Resolver) download(m *Manifest) (*Executable, error) {
	platform := r.platform()
	sum, ok := m.Runs.Checksums[platform]
	if !ok {
		return nil, fmt.Errorf("%s has no checksum for %s: the action does not support this platform", ManifestFile, platform)
	}
	want, _ := hex.DecodeString(sum)

	cacheDir, err := r.cacheDir()
	if err != nil {
		return nil, err
	}
	exe := &Executable{Path: filepath.Join(cacheDir, "sha256", sum), SHA256: want}

	if got, err := fileSHA256(exe.Path); err == nil && bytes.Equal(got, want) {
		return exe, nil
	}

	url := r.expand(m.Runs.URL)
	data, err := r.fetch(url)
	if err != nil {
		return nil, err
	}
	if got := sha256.Sum256(data); !bytes.Equal(got[:], want) {
		return nil, fmt.Errorf("%s has SHA-256 %x, but %s says %s", url, got, ManifestFile, sum)
	}
	if err := writeFileAtomic(exe.Path, bytes.NewReader(data), 0o755); err != nil {
		return nil, err
	}
	return exe, nil
}

func (r *Resolver) fetch(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "probe")

	res, err := r.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, res.Status)
	}
	return io.ReadAll(res.Body)
}

func (r *Resolver) expand(s string) string {
	return strings.NewReplacer("{os}", r.goos(), "{arch}", r.goarch()).Replace(s)
}

func (r *Resolver) platform() string {
	return r.goos() + "_" + r.goarch()
}

func (r *Resolver) goos() string {
	if r.OS != "" {
		return r.OS
	}
	return runtime.GOOS
}

func (r *Resolver) goarch() string {
	if r.Arch != "" {
		return r.Arch
	}
	return runtime.GOARCH
}

func (r *Resolver) cacheDir() (string, error) {
	if r.CacheDir != "" {
		return r.CacheDir, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no cache directory for external actions: %w", err)
	}
	return filepath.Join(dir, "probe", "actions"), nil
}

func (r *Resolver) client() *http.Client {
	if r.Client != nil {
		return r.Client
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

func (r *Resolver) rawBaseURL() string {
	if r.RawBaseURL != "" {
		return r.RawBaseURL
	}
	return "https://raw.githubusercontent.com"
}

func fileSHA256(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// writeFileAtomic writes name so that a concurrent reader, such as another
// probe sharing the cache, never sees it half written.
func writeFileAtomic(name string, r io.Reader, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(name), ".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), name)
}
