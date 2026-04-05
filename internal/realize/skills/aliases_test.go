package skills

import (
	"strings"
	"testing"

	"github.com/vibe-menu/internal/realize/dag"
)

func TestAliasMap_NoEmptyKeys(t *testing.T) {
	for k := range aliasMap {
		if k == "" {
			t.Error("aliasMap contains empty key")
		}
	}
}

func TestAliasMap_NoEmptyValues(t *testing.T) {
	for k, v := range aliasMap {
		if v == "" {
			t.Errorf("aliasMap[%q] has empty skill name", k)
		}
	}
}

func TestAliasMap_ValuesAreLowercase(t *testing.T) {
	for k, v := range aliasMap {
		if v != strings.ToLower(v) {
			t.Errorf("aliasMap[%q] = %q: skill file names should be lowercase", k, v)
		}
	}
}

func TestAliasMap_KnownLanguagesPresent(t *testing.T) {
	knownLanguages := []string{"Go", "TypeScript", "Python", "Rust", "Java", "Kotlin"}
	for _, lang := range knownLanguages {
		if _, ok := aliasMap[lang]; !ok {
			t.Errorf("aliasMap missing entry for language %q", lang)
		}
	}
}

func TestAliasMap_KnownFrameworksPresent(t *testing.T) {
	knownFrameworks := []string{"Gin", "Echo", "FastAPI", "Django", "Flask", "Spring Boot"}
	for _, fw := range knownFrameworks {
		if _, ok := aliasMap[fw]; !ok {
			t.Errorf("aliasMap missing entry for framework %q", fw)
		}
	}
}

func TestUniversalSkillsForKind_CoreTaskKindsCovered(t *testing.T) {
	// These task kinds run LLM agents and must have at least one universal skill.
	coreKinds := []dag.TaskKind{
		dag.TaskKindServicePlan,
		dag.TaskKindServiceRepository,
		dag.TaskKindServiceLogic,
		dag.TaskKindServiceHandler,
		dag.TaskKindServiceBootstrap,
		dag.TaskKindAuth,
		dag.TaskKindDataSchemas,
		dag.TaskKindDataMigrations,
		dag.TaskKindContracts,
		dag.TaskKindFrontend,
		dag.TaskKindInfraDocker,
		dag.TaskKindInfraTerraform,
		dag.TaskKindInfraCI,
		dag.TaskKindCrossCutTesting,
		dag.TaskKindCrossCutDocs,
	}
	for _, kind := range coreKinds {
		skills, ok := universalSkillsForKind[kind]
		if !ok {
			t.Errorf("universalSkillsForKind missing entry for kind %q", kind)
			continue
		}
		if len(skills) == 0 {
			t.Errorf("universalSkillsForKind[%q] is empty; expected at least one universal skill", kind)
		}
	}
}

func TestUniversalSkillsForKind_NoEmptySkillNames(t *testing.T) {
	for kind, skills := range universalSkillsForKind {
		for _, s := range skills {
			if s == "" {
				t.Errorf("universalSkillsForKind[%q] contains an empty skill name", kind)
			}
		}
	}
}

func TestAliasMap_DatabasesPresent(t *testing.T) {
	knownDBs := []string{"PostgreSQL", "MySQL", "MongoDB", "Redis"}
	for _, db := range knownDBs {
		if _, ok := aliasMap[db]; !ok {
			t.Errorf("aliasMap missing entry for database %q", db)
		}
	}
}
