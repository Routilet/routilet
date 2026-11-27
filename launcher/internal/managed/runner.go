package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ProcessRunner reuses the existing managed Node runtime. Never installs a
// compiler or runs package lifecycle scripts.
type ProcessRunner struct{ Node, NPMEntry, Cache string }

func (r ProcessRunner) NPM(ctx context.Context, dir string, args ...string) ([]byte, error) {
	operation := strings.Join(args, " ")
	args = append([]string{r.NPMEntry}, args...)
	args = append(args, "--ignore-scripts", "--no-audit", "--no-fund", "--fetch-retries=0", "--fetch-timeout=15000")
	if r.Cache != "" {
		args = append(args, "--cache", r.Cache)
	}
	output, err := r.run(ctx, dir, 3*time.Minute, args...)
	if err != nil {
		return nil, fmt.Errorf("npm %s: %w", operation, err)
	}
	return output, nil
}
func (r ProcessRunner) CLI(ctx context.Context, dir string, args ...string) ([]byte, error) {
	entry := filepath.Join(dir, "node_modules", filepath.FromSlash(App), "dist", "accoding.mjs")
	return r.run(ctx, dir, 10*time.Second, append([]string{entry}, args...)...)
}
func (r ProcessRunner) run(ctx context.Context, dir string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.Node, args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		if failure, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(string(failure.Stderr)))
		}
		return nil, err
	}
	return output, nil
}

// Outdated is a best-effort background hint, not an approved dependency plan.
// npm returns status 1 for ordinary outdated packages as well as some errors.
func (r ProcessRunner) Outdated(ctx context.Context, dir string) (bool, error) {
	args := []string{r.NPMEntry, "outdated", "--all", "--json", "--prefer-online", "--ignore-scripts", "--no-audit", "--no-fund", "--fetch-retries=0", "--fetch-timeout=4000"}
	if r.Cache != "" {
		args = append(args, "--cache", r.Cache)
	}
	cmd := exec.CommandContext(ctx, r.Node, args...)
	cmd.Dir = dir
	raw, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || ctx.Err() != nil {
			return false, err
		}
	}
	return hasWantedUpdate(raw)
}

func hasWantedUpdate(raw []byte) (bool, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		return false, err
	}
	if data == nil || data["error"] != nil {
		return false, errors.New("npm outdated did not return dependency information")
	}
	type row struct{ Current, Wanted string }
	for _, value := range data {
		var rows []row
		if err := json.Unmarshal(value, &rows); err != nil {
			var one row
			if err = json.Unmarshal(value, &one); err != nil {
				continue
			}
			rows = []row{one}
		}
		for _, item := range rows {
			if item.Current != "" && item.Wanted != "" && item.Current != item.Wanted {
				return true, nil
			}
		}
	}
	return false, nil
}
