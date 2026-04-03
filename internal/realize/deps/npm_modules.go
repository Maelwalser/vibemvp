package deps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var WellKnownNpmPackages = map[string]string{
	"next":                  "15.3.0",
	"react":                 "19.1.0",
	"react-dom":             "19.1.0",
	"typescript":            "5.7.2",
	"@types/react":          "19.1.0",
	"@types/react-dom":      "19.1.0",
	"@types/node":           "22.10.0",
	"tailwindcss":           "3.4.17",
	"postcss":               "8.5.1",
	"autoprefixer":          "10.4.20",
	"eslint":                "9.17.0",
	"eslint-config-next":    "15.3.0",
	"axios":                 "1.7.9",
	"@tanstack/react-query": "5.62.3",
	"zustand":               "5.0.2",
	"zod":                   "3.24.1",
	"react-hook-form":       "7.54.2",
	"@hookform/resolvers":   "3.9.1",
	"lucide-react":          "0.468.0",
	"clsx":                  "2.1.1",
	"tailwind-merge":        "2.5.5",
}

// LibraryAPIDocs holds the exported API surface of commonly-misused libraries.
// Injected into agent prompts to prevent hallucinated types/functions.
//
// Each entry is keyed by a lowercase technology name that matches against
// the task's technology stack.
func resolveNpmVersion(ctx context.Context, pkg, fallback string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rawURL := "https://registry.npmjs.org/" + url.PathEscape(pkg) + "/latest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fallback
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback
	}
	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Version == "" {
		return fallback
	}
	return result.Version
}

// resolveAllNpmVersions fetches the latest versions for all packages in the fallback
// map concurrently, returning a new map with resolved (or fallback) versions.
func resolveAllNpmVersions(ctx context.Context) map[string]string {
	type result struct {
		pkg, version string
	}
	ch := make(chan result, len(WellKnownNpmPackages))
	var wg sync.WaitGroup
	for pkg, fallback := range WellKnownNpmPackages {
		wg.Add(1)
		go func(p, fb string) {
			defer wg.Done()
			ch <- result{p, resolveNpmVersion(ctx, p, fb)}
		}(pkg, fallback)
	}
	wg.Wait()
	close(ch)
	resolved := make(map[string]string, len(WellKnownNpmPackages))
	for r := range ch {
		resolved[r.pkg] = r.version
	}
	return resolved
}

// resolveGoDevToolVersion fetches the latest version of a Go dev tool from the Go
// module proxy, then reads its go.mod to find the minimum Go version it requires.
// Both version and MinGoVersion are updated from live registry data.
// Falls back to t on any error.
