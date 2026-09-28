// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored (handfixes.json: instance-url-env-override): pins the
// ZAMMAD_URL / ZAMMAD_BASE_URL resolution order Load applies, so a reprint
// that drops the ZAMMAD_URL block fails here instead of shipping a CLI that
// can only ever dial the placeholder host.

package config

import "testing"

func TestLoadInstanceURLOverrides(t *testing.T) {
	t.Setenv("ZAMMAD_CONFIG", t.TempDir()+"/config.toml")
	t.Setenv("ZAMMAD_API_TOKEN", "t")

	t.Setenv("ZAMMAD_BASE_URL", "")
	t.Setenv("ZAMMAD_URL", "https://support.example.com/")
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://support.example.com/api/v1" {
		t.Fatalf("ZAMMAD_URL root should gain /api/v1: got %q", cfg.BaseURL)
	}

	t.Setenv("ZAMMAD_URL", "https://support.example.com/api/v1")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://support.example.com/api/v1" {
		t.Fatalf("ZAMMAD_URL with /api/v1 must not double it: got %q", cfg.BaseURL)
	}

	t.Setenv("ZAMMAD_BASE_URL", "https://other.example.com/custom/api")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://other.example.com/custom/api" {
		t.Fatalf("ZAMMAD_BASE_URL must win over ZAMMAD_URL: got %q", cfg.BaseURL)
	}

	t.Setenv("ZAMMAD_BASE_URL", "${user_config.zammad_base_url}")
	t.Setenv("ZAMMAD_URL", "${user_config.zammad_url}")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://your-instance.zammad.com/api/v1" {
		t.Fatalf("unresolved bundle placeholders must count as unset: got %q", cfg.BaseURL)
	}
}
