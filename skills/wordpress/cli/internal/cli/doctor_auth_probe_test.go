// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

// Hand-fix doctor-probe-requires-auth (skills/wordpress/handfixes.json).
package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wordpress-pp-cli/internal/client"
	"wordpress-pp-cli/internal/config"
)

// A WordPress site whose web server strips the Authorization header: public
// routes answer 200 anonymously, /users/me answers 401 rest_not_logged_in.
func strippingWordPress(t *testing.T, honourAuth bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authed := honourAuth && r.Header.Get("Authorization") != ""
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/me"):
			if !authed {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":"rest_not_logged_in"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
}

func probeVerdict(t *testing.T, srvURL string) string {
	t.Helper()
	cfg := &config.Config{BaseURL: srvURL, WordpressBasicAuth: "operator:app-password"}
	c := client.New(cfg, 5*time.Second, 0)
	report := map[string]any{}
	doctorProbeCredentials(context.Background(), c, RootCmd(), "wordpress-cli", report)
	v, _ := report["credentials"].(string)
	return v
}

func TestDoctorDoesNotCallAnAnonymousAnswerValid(t *testing.T) {
	srv := strippingWordPress(t, false)
	defer srv.Close()
	got := probeVerdict(t, srv.URL)
	if strings.HasPrefix(got, "valid") {
		t.Fatalf("Authorization stripped by the server, yet doctor said %q", got)
	}
	if !strings.Contains(got, "/users/me") {
		t.Fatalf("doctor should probe the auth-required /users/me route, got %q", got)
	}
}

func TestDoctorCallsAnAuthenticatedAnswerValid(t *testing.T) {
	srv := strippingWordPress(t, true)
	defer srv.Close()
	if got := probeVerdict(t, srv.URL); !strings.HasPrefix(got, "valid") {
		t.Fatalf("credential honoured by the server, yet doctor said %q", got)
	}
}
