package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeRunner struct {
	version, dep string
	fail         bool
	npmCalls     int
}

func (r *fakeRunner) NPM(_ context.Context, dir string, args ...string) ([]byte, error) {
	r.npmCalls++
	switch args[0] {
	case "view":
		return json.Marshal(r.version)
	case "ci":
		if r.fail {
			return nil, errors.New("simulated failure")
		}
		return nil, nil
	case "install", "update":
		raw, e := os.ReadFile(filepath.Join(dir, "package.json"))
		if e != nil {
			return nil, e
		}
		var p struct{ Dependencies map[string]string }
		if e = json.Unmarshal(raw, &p); e != nil {
			return nil, e
		}
		r.version = p.Dependencies[App]
		l := lockFile{3, map[string]lockEntry{"": {Dependencies: p.Dependencies}, "node_modules/" + App: {Version: r.version}, "node_modules/resource": {Version: r.dep}}}
		return nil, writeJSON(filepath.Join(dir, "package-lock.json"), l)
	}
	return nil, errors.New("unexpected npm command")
}
func (r *fakeRunner) CLI(_ context.Context, dir string, args ...string) ([]byte, error) {
	if r.fail { return nil, errors.New("simulated verification failure") }
	raw, e := os.ReadFile(filepath.Join(dir, "package-lock.json"))
	if e != nil {
		return nil, e
	}
	var l lockFile
	_ = json.Unmarshal(raw, &l)
	v := l.Packages["node_modules/"+App].Version
	if args[0] == "--version" {
		return []byte(v + "\n"), nil
	}
	return json.Marshal(Info{1, v, true, []Component{{Name: "Resource", Required: true, Status: "ready", Version: l.Packages["node_modules/resource"].Version}, {Name: "Optional", Status: "not-installed"}}})
}
func TestTransactionAndOfflineRollback(t *testing.T) {
	ctx := context.Background()
	r := &fakeRunner{version: "0.2.8", dep: "0.6.9"}
	m := Manager{t.TempDir(), "latest", r}
	c, e := m.Plan(ctx, r.version)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if _, e = c.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	first, _ := m.State()
	r.dep = "0.6.10"
	c2, e := m.Plan(ctx, r.version)
	if e != nil {
		t.Fatal(e)
	}
	defer c2.Close()
	if len(c2.Changes) != 1 || c2.Changes[0].Before != "0.6.9" || c2.Changes[0].After != "0.6.10" {
		t.Fatalf("changes: %+v", c2.Changes)
	}
	r.fail = true
	if _, e = c2.Commit(ctx); e == nil {
		t.Fatal("failed install accepted")
	}
	if s, _ := m.State(); s != first {
		t.Fatal("failure changed activation")
	}
	r.fail = false
	stale, e := m.Plan(ctx, r.version)
	if e != nil {
		t.Fatal(e)
	}
	defer stale.Close()
	if _, e = c2.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	second, _ := m.State()
	if second.Previous != first.Current {
		t.Fatal("previous not retained")
	}
	if _, e = stale.Commit(ctx); e == nil {
		t.Fatal("stale plan accepted")
	}
	calls := r.npmCalls
	if _, e = m.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if r.npmCalls != calls {
		t.Fatal("rollback called npm")
	}
	if s, _ := m.State(); s.Current != first.Current || s.Previous != second.Current {
		t.Fatal("rollback did not swap")
	}
	preview := Manager{m.Root, "next", r}
	if s, _ := preview.State(); s.Current != "" {
		t.Fatal("channels not isolated")
	}
}
func TestTamperedLockAndInvalidChannel(t *testing.T) {
	ctx := context.Background()
	r := &fakeRunner{dep: "0.6.9"}
	m := Manager{t.TempDir(), "latest", r}
	if _, e := m.Plan(ctx, "0.2.8-beta.1"); e == nil {
		t.Fatal("stable channel accepted beta")
	}
	c, e := m.Plan(ctx, "0.2.8")
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	f := filepath.Join(c.stage, "package-lock.json")
	b, _ := os.ReadFile(f)
	_ = os.WriteFile(f, append(b, ' '), 0600)
	if _, e = c.Commit(ctx); e == nil {
		t.Fatal("changed lock accepted")
	}
	if s, _ := m.State(); s.Current != "" {
		t.Fatal("invalid candidate activated")
	}
}
func TestChannelLock(t *testing.T) {
	dir := t.TempDir()
	unlock, e := lockChannel(dir)
	if e != nil {
		t.Fatal(e)
	}
	if other, e := lockChannel(dir); e == nil {
		other()
		unlock()
		t.Fatal("concurrent lock acquired")
	}
	unlock()
	again, e := lockChannel(dir)
	if e != nil {
		t.Fatal(e)
	}
	again()
}

func TestRunningReleaseRetainedAndUninstallProtected(t *testing.T) {
	ctx := context.Background()
	r := &fakeRunner{version: "0.2.8", dep: "0.6.9"}
	m := Manager{t.TempDir(), "latest", r}
	install := func(version string) {
		t.Helper()
		c, err := m.Plan(ctx, version)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := c.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	install("0.2.8")
	first, _, free, err := m.AcquireCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	install("0.2.9")
	install("0.2.10")
	if count, err := m.Cleanup(); err != nil || count != 0 {
		t.Fatalf("removed leased release: %d %v", count, err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	before, _ := m.State()
	if err := m.Uninstall(); err == nil {
		t.Fatal("uninstall removed a running release")
	}
	if after, _ := m.State(); after != before {
		t.Fatal("failed uninstall changed active state")
	}
	free()
	if count, err := m.Cleanup(); err != nil || count != 1 {
		t.Fatalf("cleanup: %d %v", count, err)
	}
	if _, err := os.Stat(first); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old release remained")
	}
	preview := Manager{m.Root, "next", r}
	c, err := preview.Plan(ctx, "0.3.0-beta.1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if s, _ := m.State(); s.Current != "" {
		t.Fatal("uninstall retained active channel")
	}
	if s, _ := preview.State(); s.Current == "" {
		t.Fatal("uninstall removed preview")
	}
}

func TestStageLeaseAndRepair(t *testing.T) {
	ctx := context.Background()
	r := &fakeRunner{version: "0.2.8", dep: "0.6.9"}
	m := Manager{t.TempDir(), "latest", r}
	c, err := m.Plan(ctx, r.version)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	dir, _ := m.dir()
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(c.stage, old, old); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(dir, "releases", ".stage-orphan")
	if err := os.Mkdir(orphan, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orphan, old, old); err != nil {
		t.Fatal(err)
	}
	if removed, err := m.Cleanup(); removed != 1 || err != nil {
		t.Fatalf("cleanup stage: %d %v", removed, err)
	}
	if _, err := c.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	current, _ := m.Current()
	if err := os.Remove(filepath.Join(current, "package-lock.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.VerifyCurrent(ctx); err == nil {
		t.Fatal("damaged install verified")
	}
	repair, err := m.Plan(ctx, r.version)
	if err != nil {
		t.Fatal(err)
	}
	defer repair.Close()
	if len(repair.Changes) == 0 {
		t.Fatal("repair did not resolve new lock")
	}
	if _, err := repair.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOutdatedHints(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want bool
	}{
		{`{}`, false},
		{`{"resource":{"current":"0.6.9","wanted":"0.6.9","latest":"0.7.0"}}`, false},
		{`{"resource":{"current":"0.6.9","wanted":"0.6.10","latest":"0.7.0"}}`, true},
		{`{"resource":[{"current":"0.6.9","wanted":"0.6.10"}]}`, true},
	} {
		if got, err := hasWantedUpdate([]byte(test.raw)); err != nil || got != test.want {
			t.Fatalf("%s: got %v %v", test.raw, got, err)
		}
	}
	if _, err := hasWantedUpdate([]byte(`{"error":{"code":"E404"}}`)); err == nil {
		t.Fatal("npm error treated as success")
	}
}
