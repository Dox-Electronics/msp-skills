// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import "testing"

func TestHsSyncPruneAllowed(t *testing.T) {
	cases := []struct {
		name                 string
		full, noPrune        bool
		param, resParam, glb []string
		want                 bool
	}{
		{"full unfiltered walk prunes", true, false, nil, nil, nil, true},
		{"incremental never prunes", false, false, nil, nil, nil, false},
		{"--no-prune wins", true, true, nil, nil, nil, false},
		{"--param narrows the walk", true, false, []string{"archived=true"}, nil, nil, false},
		{"--resource-param narrows the walk", true, false, nil, []string{"hubspot-contacts-crm:archived=true"}, nil, false},
		{"--global-param narrows the walk", true, false, nil, nil, []string{"archived=true"}, false},
	}
	for _, c := range cases {
		if got := hsSyncPruneAllowed(c.full, c.noPrune, c.param, c.resParam, c.glb); got != c.want {
			t.Errorf("%s: hsSyncPruneAllowed = %v, want %v", c.name, got, c.want)
		}
	}
}
