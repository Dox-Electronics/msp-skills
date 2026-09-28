// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

// Hand-fix explicit-config-credentials-one-path (skills/wordpress/handfixes.json).
// Under an explicit --config / WORDPRESS_CONFIG file, Load reads the sibling
// data/credentials.toml first. Saving and clearing must act on that same file,
// or `auth set-token` writes the global store while the next Load keeps the
// stale sibling token, and `auth logout` leaves the sibling token active.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"wordpress-pp-cli/internal/cliutil"
)

func TestExplicitConfigSavesAndClearsItsOwnCredentialsFile(t *testing.T) {
	isolateCredentials(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	sibling := filepath.Join(dir, "data", "credentials.toml")

	// A stale token already sits in the sibling store.
	if err := cliutil.SaveCredentialsTo(sibling, &cliutil.Credentials{WordpressBasicAuth: "stale-token"}); err != nil {
		t.Fatalf("seed sibling: %v", err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.WordpressBasicAuth != "stale-token" {
		t.Fatalf("precondition: want sibling token loaded, got %q", cfg.WordpressBasicAuth)
	}
	if got, _ := cfg.CredentialsFilePath(); got == "" || filepath.Base(filepath.Dir(got)) != "data" {
		t.Fatalf("CredentialsFilePath = %q, want the explicit config's sibling data/credentials.toml", got)
	}

	if err := cfg.SaveCredential("fresh-token"); err != nil {
		t.Fatalf("save: %v", err)
	}
	reloaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.WordpressBasicAuth != "fresh-token" {
		t.Fatalf("after set-token, Load returned %q; want fresh-token (save and load must use one store)", reloaded.WordpressBasicAuth)
	}
	global, err := cliutil.CredentialsFilePath()
	if err != nil {
		t.Fatalf("global path: %v", err)
	}
	if _, err := os.Stat(global); !os.IsNotExist(err) {
		t.Fatalf("explicit-config save leaked into the global credentials file %s (err=%v)", global, err)
	}

	if err := reloaded.ClearTokens(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := os.Stat(sibling); !os.IsNotExist(err) {
		t.Fatalf("logout left the sibling credentials file behind (err=%v)", err)
	}
	after, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load after logout: %v", err)
	}
	if after.WordpressBasicAuth != "" {
		t.Fatalf("after logout Load still returned a credential %q", after.WordpressBasicAuth)
	}
}

func TestDefaultConfigStillUsesTheGlobalCredentialsFile(t *testing.T) {
	isolateCredentials(t)
	t.Setenv("WORDPRESS_CONFIG", "")
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := cfg.CredentialsFilePath()
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	want, err := cliutil.CredentialsFilePath()
	if err != nil {
		t.Fatalf("global: %v", err)
	}
	if got != want {
		t.Fatalf("default config credentials path = %q, want global %q", got, want)
	}
}

func TestSymlinkedExplicitConfigKeepsOneCredentialsFileAcrossSave(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX symlink test")
	}
	isolateCredentials(t)
	t.Setenv("WORDPRESS_BASE_URL", "")
	linkDir := t.TempDir()
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "config.toml")
	if err := os.WriteFile(target, []byte("base_url = \"https://site.test/wp-json/wp/v2\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "config.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(link)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := cfg.SaveCredential("fresh-token"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("save replaced the config symlink with a regular file (err=%v)", err)
	}
	reloaded, err := Load(link)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.WordpressBasicAuth != "fresh-token" {
		t.Fatalf("after set-token through a symlinked config, Load returned %q; want fresh-token", reloaded.WordpressBasicAuth)
	}
	if reloaded.BaseURL != "https://site.test/wp-json/wp/v2" {
		t.Fatalf("config written through the link lost base_url: %q", reloaded.BaseURL)
	}
}
