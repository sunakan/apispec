// Copyright 2026 Ehab Terra
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spec

import "testing"

func TestSplitMethodFromPath(t *testing.T) {
	cases := []struct {
		in         string
		wantMethod string
		wantPath   string
	}{
		// Go 1.22 ServeMux method-aware patterns.
		{`"GET /users/{id}"`, "GET", "/users/{id}"},
		{`"POST /users"`, "POST", "/users"},
		{`"get /health"`, "GET", "/health"}, // verb is upper-cased
		{`"DELETE /users/{id}"`, "DELETE", "/users/{id}"},
		// Plain net/http patterns pass through unchanged.
		{`"/health"`, "", "/health"},
		{`"/users/{id}"`, "", "/users/{id}"},
		// A leading non-verb token is not treated as a method.
		{`"FOO /bar"`, "", "FOO /bar"},
		{"", "", ""},
	}
	for _, c := range cases {
		gotMethod, gotPath := splitMethodFromPath(c.in)
		if gotMethod != c.wantMethod || gotPath != c.wantPath {
			t.Errorf("splitMethodFromPath(%q) = (%q, %q); want (%q, %q)",
				c.in, gotMethod, gotPath, c.wantMethod, c.wantPath)
		}
	}
}

func TestNormalizeServeMuxPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/users/{id}", "/users/{id}"},
		{"/files/{path...}", "/files/{path}"}, // trailing wildcard
		{"/items/{$}", "/items/"},             // end-of-path anchor dropped
		{"/static/{dir...}/{$}", "/static/{dir}/"},
	}
	for _, c := range cases {
		if got := normalizeServeMuxPath(c.in); got != c.want {
			t.Errorf("normalizeServeMuxPath(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestRouteMethodAndPathSyntax(t *testing.T) {
	cases := []struct {
		name     string
		cfg      *APISpecConfig
		pattern  int
		path     string
		handler  string
		method   string
		wantPath string
		explicit bool
	}{
		{"chi explicit verb", DefaultChiConfig(), 2, "POST /items", "getItems", "POST", "/items", true},
		{"chi end parameter", DefaultChiConfig(), 2, "POST /tail/{$}", "handler", "POST", "/tail/{$}", true},
		{"chi dots parameter", DefaultChiConfig(), 2, "GET /files/{rest...}", "handler", "GET", "/files/{rest...}", true},
		{"chi plain parameter", DefaultChiConfig(), 2, "/tail/{$}", "deleteItems", "DELETE", "/tail/{$}", true},
		{"chi fallback", DefaultChiConfig(), 2, "/items", "handler", "GET", "/items", false},
		{"chi host retained", DefaultChiConfig(), 2, "GET api.example.com/items", "handler", "GET", "api.example.com/items", true},
		{"http wildcard", DefaultHTTPConfig(), 0, "GET /files/{rest...}", "handler", "GET", "/files/{rest}", true},
		{"http anchor", DefaultHTTPConfig(), 0, "POST /tail/{$}", "handler", "POST", "/tail/", true},
		{"http host", DefaultHTTPConfig(), 0, "GET api.example.com/items", "handler", "GET", "/items", true},
		{"secondary http wildcard", HTTPSecondaryConfig(), 0, "GET /files/{rest...}", "handler", "GET", "/files/{rest}", true},
		{"secondary http anchor", HTTPSecondaryConfig(), 1, "POST /tail/{$}", "handler", "POST", "/tail/", true},
		{"secondary http host", HTTPSecondaryConfig(), 1, "GET api.example.com/items", "handler", "GET", "/items", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg.Hosts = []string{"api.example.com"}
			meta := exSweepMeta()
			matcher := NewRoutePatternMatcher(tc.cfg.Framework.RoutePatterns[tc.pattern], tc.cfg, NewContextProvider(meta))
			edge := sweepEdge(meta, "main", "app", "HandleFunc", "app", "", "",
				sweepLit(meta, tc.path), sweepIdent(meta, tc.handler))
			route := NewRouteInfo()
			if !matcher.extractRouteDetails(sweepNode(edge), route) {
				t.Fatal("route details missing")
			}
			if route.Method != tc.method || route.Path != tc.wantPath || route.MethodExplicit != tc.explicit {
				t.Errorf("got %s %s (explicit=%v), want %s %s (explicit=%v)",
					route.Method, route.Path, route.MethodExplicit, tc.method, tc.wantPath, tc.explicit)
			}
		})
	}
}
