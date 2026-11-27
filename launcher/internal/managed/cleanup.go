package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func leasePath(dir, release string) string {
	// Keep the lease file outside the release; Windows cannot delete an open
	// lock file. The small lease files are retained to avoid lock-inode races.
	return filepath.Join(dir, "leases", release+".lock")
}

func verifiedMetadata(dir, channel string) (Info, error) {
	var info Info
	raw, err := os.ReadFile(filepath.Join(dir, "installation.json"))
	if err != nil {
		return info, err
	}
	if json.Unmarshal(raw, &info) != nil || info.SchemaVersion != schema || !info.Ready || !validVersion(channel, info.Version) {
		return info, errors.New("invalid installation information")
	}
	return info, nil
}

// AcquireCurrent pins the exact release while a child runs. The shared lease
// survives activation changes and excludes deletion, without blocking updates.
func (m Manager) AcquireCurrent(ctx context.Context) (string, Info, func(), error) {
	var info Info
	dir, err := m.dir()
	if err != nil {
		return "", info, nil, err
	}
	unlock, err := lockChannel(dir)
	if err != nil {
		return "", info, nil, err
	}
	s, err := readState(dir)
	if err != nil || s.Current == "" {
		unlock()
		if err == nil {
			err = errors.New("no active installation")
		}
		return "", info, nil, err
	}
	release := filepath.Join(dir, "releases", s.Current)
	free, err := lockPath(leasePath(dir, s.Current), false)
	unlock()
	if err != nil {
		return "", info, nil, err
	}
	info, err = verifiedMetadata(release, m.Channel)
	if err == nil {
		_, _, err = readLock(release, info.Version)
	}
	if err == nil {
		info, err = verify(ctx, m.Runner, release, info.Version)
	}
	if err != nil {
		free()
		return "", info, nil, err
	}
	return release, info, free, nil
}

// Cleanup retains current and previous. Skip any release with a live process
// lease. Stages are only removed when old and no live candidate owns them.
func (m Manager) Cleanup() (int, error) {
	dir, err := m.dir()
	if err != nil {
		return 0, err
	}
	unlock, err := lockChannel(dir)
	if err != nil {
		return 0, err
	}
	defer unlock()
	s, err := readState(dir)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(filepath.Join(dir, "releases"))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || name == s.Current || name == s.Previous {
			continue
		}
		if strings.HasPrefix(name, ".stage-") && releasePattern.MatchString("release-"+strings.TrimPrefix(name, ".stage-")) {
			info, err := e.Info()
			if err != nil || time.Since(info.ModTime()) < 24*time.Hour {
				continue
			}
		} else if !releasePattern.MatchString(name) {
			continue
		}
		free, err := lockPath(leasePath(dir, name), true)
		if err != nil {
			continue // active or unavailable; do not delete
		}
		err = os.RemoveAll(filepath.Join(dir, "releases", name))
		free()
		if err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// Uninstall removes this v2 channel only; never touches legacy installs,
// the shared Node runtime, another channel, credentials, or coursework.
// Refuse while any release is in use, before changing activation.
func (m Manager) Uninstall() error {
	dir, err := m.dir()
	if err != nil {
		return err
	}
	unlock, err := lockChannel(dir)
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := readState(dir); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(dir, "releases"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var names []string
	var leases []func()
	defer func() {
		for _, free := range leases {
			free()
		}
	}()
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !releasePattern.MatchString(name) {
			continue
		}
		free, err := lockPath(leasePath(dir, name), true)
		if err != nil {
			return fmt.Errorf("installation in use: %s", name)
		}
		leases = append(leases, free)
		names = append(names, name)
	}
	if err := os.Remove(filepath.Join(dir, "activation.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var failures []error
	for _, name := range names {
		if err := os.RemoveAll(filepath.Join(dir, "releases", name)); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
