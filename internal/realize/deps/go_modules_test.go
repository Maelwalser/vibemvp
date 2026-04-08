package deps

import (
	"strings"
	"testing"
)

func TestStubGoMod_IncludesUUID(t *testing.T) {
	stub := StubGoMod("monolith", "1.23", nil, nil)
	if !strings.Contains(stub, "github.com/google/uuid") {
		t.Error("stub go.mod should include uuid")
	}
	if !strings.Contains(stub, "module monolith") {
		t.Error("stub should have correct module path")
	}
	if !strings.Contains(stub, "go 1.23") {
		t.Error("stub should have correct go version")
	}
}

func TestStubGoMod_IncludesDBDriver(t *testing.T) {
	stub := StubGoMod("monolith", "1.23", []string{"PostgreSQL"}, nil)
	if !strings.Contains(stub, "pgx") {
		t.Error("stub should include pgx for PostgreSQL")
	}
	if !strings.Contains(stub, "github.com/google/uuid") {
		t.Error("stub should still include uuid")
	}
}

func TestStubGoMod_DefaultGoVersion(t *testing.T) {
	stub := StubGoMod("myapp", "", nil, nil)
	if !strings.Contains(stub, "go 1.23") {
		t.Error("stub should default to go 1.23 when goVersion is empty")
	}
}

func TestStubGoMod_UsesResolvedVersions(t *testing.T) {
	resolved := map[string]ModuleInfo{
		"uuid": {Module: "github.com/google/uuid", Version: "v1.7.0"},
	}
	stub := StubGoMod("myapp", "1.23", nil, resolved)
	if !strings.Contains(stub, "v1.7.0") {
		t.Error("stub should use resolved version v1.7.0, not static fallback")
	}
}

func TestStubGoMod_MultipleTechnologies(t *testing.T) {
	stub := StubGoMod("myapp", "1.23", []string{"PostgreSQL", "Redis"}, nil)
	if !strings.Contains(stub, "pgx") {
		t.Error("stub should include pgx for PostgreSQL")
	}
	if !strings.Contains(stub, "go-redis") {
		t.Error("stub should include go-redis for Redis")
	}
}

func TestStubGoMod_FallbackModulePath(t *testing.T) {
	stub := StubGoMod("stub", "1.23", nil, nil)
	if !strings.Contains(stub, "module stub") {
		t.Error("stub should use provided module path")
	}
}
