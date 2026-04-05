package dag

import (
	"testing"

	"github.com/vibe-menu/internal/manifest"
)

// minimalMonolith returns a manifest with a single monolith service.
func minimalMonolith() *manifest.Manifest {
	return &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMonolith,
			Services: []manifest.ServiceDef{
				{Name: "monolith", Language: "Go", Framework: "Gin"},
			},
		},
	}
}

func buildDAG(t *testing.T, m *manifest.Manifest) *DAG {
	t.Helper()
	d, err := (&Builder{}).Build(m)
	if err != nil {
		t.Fatalf("Builder.Build() unexpected error: %v", err)
	}
	return d
}

func assertTaskPresent(t *testing.T, d *DAG, id string) {
	t.Helper()
	if _, ok := d.Tasks[id]; !ok {
		t.Errorf("expected task %q in DAG, not found; tasks=%v", id, taskIDs(d))
	}
}

func assertTaskAbsent(t *testing.T, d *DAG, id string) {
	t.Helper()
	if _, ok := d.Tasks[id]; ok {
		t.Errorf("unexpected task %q found in DAG", id)
	}
}

func taskIDs(d *DAG) []string {
	ids := make([]string, 0, len(d.Tasks))
	for id := range d.Tasks {
		ids = append(ids, id)
	}
	return ids
}

func TestBuild_DataTasksAlwaysPresent(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskPresent(t, d, "data.schemas")
	assertTaskPresent(t, d, "data.migrations")
}

func TestBuild_Monolith_ServiceChain(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	for _, id := range []string{
		"svc.monolith.plan",
		"svc.monolith.deps",
		"svc.monolith.repository",
		"svc.monolith.service",
		"svc.monolith.handler",
		"svc.monolith.bootstrap",
	} {
		assertTaskPresent(t, d, id)
	}
}

func TestBuild_Monolith_ServiceChain_DependencyOrder(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	order := d.Order()
	pos := make(map[string]int, len(order))
	for i, id := range order {
		pos[id] = i
	}
	chain := []string{
		"svc.monolith.plan",
		"svc.monolith.deps",
		"svc.monolith.repository",
		"svc.monolith.service",
		"svc.monolith.handler",
		"svc.monolith.bootstrap",
	}
	for i := 1; i < len(chain); i++ {
		if pos[chain[i-1]] >= pos[chain[i]] {
			t.Errorf("%s must precede %s in topological order", chain[i-1], chain[i])
		}
	}
}

func TestBuild_Microservices_TwoServices(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "api", Language: "Go", Framework: "Gin"},
				{Name: "worker", Language: "Go", Framework: "Gin"},
			},
		},
	}
	d := buildDAG(t, m)

	for _, svc := range []string{"api", "worker"} {
		assertTaskPresent(t, d, "svc."+svc+".bootstrap")
		assertTaskPresent(t, d, "svc."+svc+".plan")
	}
}

func TestBuild_WithAuth_AddsAuthTask(t *testing.T) {
	m := minimalMonolith()
	m.Backend.Auth = &manifest.AuthConfig{Strategy: "JWT"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "backend.auth")
}

func TestBuild_WithoutAuth_NoAuthTask(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "backend.auth")
}

func TestBuild_WithMessaging_AddsMessagingTask(t *testing.T) {
	m := minimalMonolith()
	m.Backend.Messaging = &manifest.MessagingConfig{BrokerTech: "Kafka"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "backend.messaging")
}

func TestBuild_WithoutMessaging_NoMessagingTask(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "backend.messaging")
}

func TestBuild_NoContractsWhenEmpty(t *testing.T) {
	// No DTOs and no endpoints → no contracts task
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "contracts")
}

func TestBuild_ContractsPresent_WhenDTOsDefined(t *testing.T) {
	m := minimalMonolith()
	m.Contracts.DTOs = []manifest.DTODef{{Name: "UserDTO"}}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "contracts")
}

func TestBuild_ContractsDependsOnServices(t *testing.T) {
	m := minimalMonolith()
	m.Contracts.DTOs = []manifest.DTODef{{Name: "UserDTO"}}
	d := buildDAG(t, m)

	contracts := d.Tasks["contracts"]
	if contracts == nil {
		t.Fatal("contracts task not found")
	}

	found := false
	for _, dep := range contracts.Dependencies {
		if dep == "svc.monolith.bootstrap" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("contracts should depend on svc.monolith.bootstrap, deps=%v", contracts.Dependencies)
	}
}

func TestBuild_NoFrontend_WhenTechEmpty(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "frontend")
}

func TestBuild_FrontendPresent_WhenFrameworkSet(t *testing.T) {
	m := minimalMonolith()
	m.Frontend.Tech = &manifest.FrontendTechConfig{Framework: "Next.js", Language: "TypeScript"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "frontend")
}

func TestBuild_InfraDockerAlwaysPresent(t *testing.T) {
	// infra.docker is always added regardless of manifest content
	d := buildDAG(t, minimalMonolith())
	assertTaskPresent(t, d, "infra.docker")
}

func TestBuild_InfraTerraform_OnlyWhenIaCToolSet(t *testing.T) {
	m := minimalMonolith()
	d := buildDAG(t, m)
	assertTaskAbsent(t, d, "infra.terraform")

	m.Infra.CICD = &manifest.CICDConfig{IaCTool: "Terraform"}
	d = buildDAG(t, m)
	assertTaskPresent(t, d, "infra.terraform")
}

func TestBuild_InfraCI_OnlyWhenPlatformSet(t *testing.T) {
	m := minimalMonolith()
	d := buildDAG(t, m)
	assertTaskAbsent(t, d, "infra.cicd")

	m.Infra.CICD = &manifest.CICDConfig{Platform: "GitHub Actions"}
	d = buildDAG(t, m)
	assertTaskPresent(t, d, "infra.cicd")
}

func TestBuild_CrossCutTesting_WhenUnitSet(t *testing.T) {
	m := minimalMonolith()
	m.CrossCut.Testing = &manifest.TestingConfig{Unit: "go test"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "crosscut.testing")
}

func TestBuild_CrossCutDocs_WhenAPIDocsSet(t *testing.T) {
	m := minimalMonolith()
	m.CrossCut.Docs = &manifest.DocsConfig{APIDocs: "OpenAPI"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "crosscut.docs")
}

func TestBuild_ValidDAG_NoErrors(t *testing.T) {
	// Full-featured manifest should produce a valid DAG with no build errors
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "users", Language: "Go", Framework: "Gin"},
				{Name: "orders", Language: "Go", Framework: "Gin"},
			},
			Auth:      &manifest.AuthConfig{Strategy: "JWT"},
			Messaging: &manifest.MessagingConfig{BrokerTech: "Kafka"},
		},
		Contracts: manifest.ContractsPillar{
			DTOs:      []manifest.DTODef{{Name: "UserDTO"}},
			Endpoints: []manifest.EndpointDef{{NamePath: "/users/:id", Protocol: "REST"}},
		},
		Frontend: manifest.FrontendPillar{
			Tech: &manifest.FrontendTechConfig{Framework: "Next.js", Language: "TypeScript"},
		},
	}
	_, err := (&Builder{}).Build(m)
	if err != nil {
		t.Errorf("Build() returned error for valid full manifest: %v", err)
	}
}
