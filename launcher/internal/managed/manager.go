// Package managed implements the v2 installation transaction, independent of
// the terminal UI, course resources, and specific optional components.
package managed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"reflect"
	"sort"
	"strings"
)

const App = "@routilet/accoding-companion"
const schema = 1

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$`)
var releasePattern = regexp.MustCompile(`^release-[0-9A-Za-z-]+$`)

// Runner calls the launcher-managed Node/npm, never a shell. Implementations
// return stdout (machine-readable commands) and retain useful errors.
type Runner interface {
	NPM(context.Context, string, ...string) ([]byte, error)
	CLI(context.Context, string, ...string) ([]byte, error)
}

type Component struct {
	Name     string `json:"name"`
	Package  string `json:"package"`
	Version  string `json:"version,omitempty"`
	Required bool   `json:"required"`
	Status   string `json:"status"`
}
type Info struct {
	SchemaVersion int         `json:"schemaVersion"`
	Version       string      `json:"version"`
	Ready         bool        `json:"ready"`
	Components    []Component `json:"components"`
}
type Activation struct {
	SchemaVersion int    `json:"schemaVersion"`
	Current       string `json:"current"`
	Previous      string `json:"previous,omitempty"`
}
type Change struct {
	Path, Before, After string
	Optional            bool
}
type Manager struct {
	Root    string
	Channel string
	Runner  Runner
}
type Candidate struct {
	manager Manager
	stage   string
	free    func()
	base    Activation
	digest  [32]byte
	version string
	Changes []Change
	Info Info
}

func (m Manager) dir() (string, error) {
	if m.Channel != "latest" && m.Channel != "next" {
		return "", errors.New("unknown release channel")
	}
	if !filepath.IsAbs(m.Root) || m.Runner == nil {
		return "", errors.New("invalid manager configuration")
	}
	return filepath.Join(m.Root, "v2", "channels", m.Channel), nil
}
func (m Manager) State() (Activation, error) {
	dir, err := m.dir()
	if err != nil {
		return Activation{}, err
	}
	return readState(dir)
}
func readState(dir string) (Activation, error) {
	s := Activation{SchemaVersion: schema}
	data, err := os.ReadFile(filepath.Join(dir, "activation.json"))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	if s.SchemaVersion != schema || !releasePattern.MatchString(s.Current) || (s.Previous != "" && (!releasePattern.MatchString(s.Previous) || s.Previous == s.Current)) {
		return s, errors.New("invalid activation record; preserved")
	}
	return s, nil
}
func (m Manager) Current() (string, error) {
	dir, err := m.dir()
	if err != nil {
		return "", err
	}
	s, err := readState(dir)
	if err != nil || s.Current == "" {
		return "", err
	}
	return filepath.Join(dir, "releases", s.Current), nil
}
func validVersion(channel, v string) bool {
	return versionPattern.MatchString(v) && (channel != "latest" || !strings.Contains(v, "-"))
}
func (m Manager) Resolve(ctx context.Context) (string, error) {
	dir, err := m.dir()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	out, err := m.Runner.NPM(ctx, dir, "view", App+"@"+m.Channel, "version", "--json", "--prefer-online")
	if err != nil {
		return "", err
	}
	var version string
	if json.Unmarshal(out, &version) != nil || !validVersion(m.Channel, version) {
		return "", errors.New("invalid registry channel version")
	}
	return version, nil
}

type lockEntry struct {
	Version      string            `json:"version"`
	Optional     bool              `json:"optional"`
	Dependencies map[string]string `json:"dependencies"`
}
type lockFile struct {
	LockfileVersion int                  `json:"lockfileVersion"`
	Packages        map[string]lockEntry `json:"packages"`
}

func readLock(dir, version string) (lockFile, []byte, error) {
	var l lockFile
	raw, err := os.ReadFile(filepath.Join(dir, "package-lock.json"))
	if err != nil {
		return l, nil, err
	}
	if err = json.Unmarshal(raw, &l); err != nil {
		return l, nil, err
	}
	if l.LockfileVersion < 2 || l.Packages[""].Dependencies[App] != version || l.Packages["node_modules/"+App].Version != version {
		return l, nil, errors.New("lock does not preserve exact Companion version")
	}
	return l, raw, nil
}
func differences(before, after lockFile) []Change {
	keys := map[string]bool{}
	for k := range before.Packages {
		if k != "" {
			keys[k] = true
		}
	}
	for k := range after.Packages {
		if k != "" {
			keys[k] = true
		}
	}
	var changes []Change
	for k := range keys {
		a, b := before.Packages[k], after.Packages[k]
		if a.Version != b.Version {
			changes = append(changes, Change{k, a.Version, b.Version, b.Optional})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}
func writeJSON(file string, value any) error {
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(file, append(b, '\n'), 0600)
}

// Plan installs a complete candidate in a private directory. npm install/update
// tolerate absent optional dependencies on npm 10 and 11 where npm ci can
// reject a lock without their metadata. No activation occurs until approval.
// Caller must Close a canceled candidate.
func (m Manager) Plan(ctx context.Context, version string) (*Candidate, error) {
	dir, err := m.dir()
	if err != nil {
		return nil, err
	}
	if !validVersion(m.Channel, version) {
		return nil, errors.New("invalid target version")
	}
	if err = os.MkdirAll(filepath.Join(dir, "releases"), 0700); err != nil {
		return nil, err
	}
	base, err := readState(dir)
	if err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(filepath.Join(dir, "releases"), ".stage-")
	if err != nil {
		return nil, err
	}
	free, err := lockPath(leasePath(dir, filepath.Base(stage)), false)
	if err != nil {
		_ = os.RemoveAll(stage)
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(stage)
			free()
		}
	}()
	var previous lockFile
	if base.Current != "" {
		old := filepath.Join(dir, "releases", base.Current)
		// Missing or unreadable prior lock should not prevent an explicit
		// repair. Re-resolve from scratch and show the full candidate tree.
		raw, readErr := os.ReadFile(filepath.Join(old, "package-lock.json"))
		if readErr == nil && json.Unmarshal(raw, &previous) == nil && previous.LockfileVersion >= 2 && len(previous.Packages) != 0 {
			if err = os.WriteFile(filepath.Join(stage, "package-lock.json"), raw, 0600); err != nil {
				return nil, err
			}
		} else {
			previous = lockFile{}
		}
	}
	pkg := map[string]any{"name": "accoding-managed-app", "private": true, "dependencies": map[string]string{App: version}}
	if err = writeJSON(filepath.Join(stage, "package.json"), pkg); err != nil {
		return nil, err
	}
	if _, err = m.Runner.NPM(ctx, stage, "install", "--prefer-online"); err != nil {
		return nil, err
	}
	if _, err = m.Runner.NPM(ctx, stage, "update", "--save", "--prefer-online"); err != nil {
		return nil, err
	}
	next, raw, err := readLock(stage, version)
	if err != nil {
		return nil, err
	}
	info, err := verify(ctx, m.Runner, stage, version)
	if err != nil { return nil, err }
	keep = true
	return &Candidate{manager: m, stage: stage, free: free, base: base, digest: sha256.Sum256(raw), version: version, Info: info, Changes: differences(previous, next)}, nil
}
func (c *Candidate) Close() error {
	if c == nil {
		return nil
	}
	var err error
	if c.stage != "" {
		err = os.RemoveAll(c.stage)
		if err == nil {
			c.stage = ""
		}
	}
	if c.free != nil {
		c.free()
		c.free = nil
	}
	return err
}
func verify(ctx context.Context, r Runner, dir, version string) (Info, error) {
	var info Info
	out, err := r.CLI(ctx, dir, "--version")
	if err != nil {
		return info, fmt.Errorf("Companion %s --version check failed: %w", version, err)
	}
	if strings.TrimSpace(string(out)) != version {
		return info, errors.New("Companion version mismatch")
	}
	out, err = r.CLI(ctx, dir, "--installation-info")
	if err != nil {
		return info, fmt.Errorf("Companion %s --installation-info 校验失败（需要支持新版安装协议的 Companion）：%w", version, err)
	}
	if err = json.Unmarshal(out, &info); err != nil {
		return info, fmt.Errorf("installation-info: %w", err)
	}
	if info.SchemaVersion != schema || info.Version != version || !info.Ready {
		return info, errors.New("Companion installation is not ready")
	}
	for _, c := range info.Components {
		if c.Required && c.Status != "ready" {
			return info, fmt.Errorf("required component %s: %s", c.Name, c.Status)
		}
	}
	return info, nil
}
func writeState(dir string, s Activation) error {
	file, err := os.CreateTemp(dir, ".activation-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = json.NewEncoder(file).Encode(s)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), filepath.Join(dir, "activation.json"))
}

// Commit verifies exactly the reviewed staged tree and actual CLI, and
// swaps one activation record. A stale plan never replaces a newer install.
// No historical release is deleted here: cleanup needs running-process leases.
func (c *Candidate) Commit(ctx context.Context) (Info, error) {
	var info Info
	if c == nil || c.stage == "" {
		return info, errors.New("closed candidate")
	}
	m := c.manager
	dir, err := m.dir()
	if err != nil {
		return info, err
	}
	unlock, err := lockChannel(dir)
	if err != nil {
		return info, err
	}
	defer unlock()
	now, err := readState(dir)
	if err != nil {
		return info, err
	}
	if now != c.base {
		return info, errors.New("installation changed; resolve again")
	}
	_, raw, err := readLock(c.stage, c.version)
	if err != nil {
		return info, err
	}
	if sha256.Sum256(raw) != c.digest {
		return info, errors.New("candidate lock changed")
	}
	info, err = verify(ctx, m.Runner, c.stage, c.version)
	if err != nil {
		return info, err
	}
	if !reflect.DeepEqual(info,c.Info) { return info, errors.New("reviewed component state changed; resolve again") }
	if err = writeJSON(filepath.Join(c.stage, "installation.json"), info); err != nil {
		return info, err
	}
	name := "release-" + strings.TrimPrefix(filepath.Base(c.stage), ".stage-")
	destination := filepath.Join(dir, "releases", name)
	if _, err = os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		return info, errors.New("release name already exists")
	}
	if err = os.Rename(c.stage, destination); err != nil {
		return info, err
	}
	c.stage = ""
	if c.free != nil {
		c.free()
		c.free = nil
	}
	if err = writeState(dir, Activation{schema, name, now.Current}); err != nil {
		_ = os.RemoveAll(destination)
		return info, err
	}
	return info, nil
}
func (m Manager) Rollback(ctx context.Context) (Info, error) {
	var info Info
	dir, err := m.dir()
	if err != nil {
		return info, err
	}
	unlock, err := lockChannel(dir)
	if err != nil {
		return info, err
	}
	defer unlock()
	s, err := readState(dir)
	if err != nil {
		return info, err
	}
	if s.Previous == "" {
		return info, errors.New("no previous release")
	}
	previous := filepath.Join(dir, "releases", s.Previous)
	raw, err := os.ReadFile(filepath.Join(previous, "installation.json"))
	if err != nil {
		return info, err
	}
	if json.Unmarshal(raw, &info) != nil || !validVersion(m.Channel, info.Version) {
		return info, errors.New("invalid previous installation")
	}
	if _, _, err = readLock(previous, info.Version); err != nil {
		return info, err
	}
	info, err = verify(ctx, m.Runner, previous, info.Version)
	if err != nil {
		return info, err
	}
	return info, writeState(dir, Activation{schema, s.Previous, s.Current})
}

// InstalledInfo reads the last verified metadata without starting Node.
func (m Manager) InstalledInfo() (Info, error) {
	var info Info
	dir, err := m.Current()
	if err != nil || dir == "" {
		return info, err
	}
	return verifiedMetadata(dir, m.Channel)
}

func (m Manager) VerifyCurrent(ctx context.Context) (Info, error) {
	_, info, free, err := m.AcquireCurrent(ctx)
	if free != nil {
		defer free()
	}
	return info, err
}
