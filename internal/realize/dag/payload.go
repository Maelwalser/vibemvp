package dag

import "github.com/vibe-mvp/internal/manifest"

// TaskPayload is the scoped manifest slice fed to the agent for one task.
// Only the fields relevant to the task's Kind are populated — others are nil/zero.
// This keeps agent prompts tight and prevents cross-contamination between pillars.
// omitempty tags ensure nil/zero fields are omitted from JSON to reduce token usage.
type TaskPayload struct {
	// ModulePath is the Go module path for this service (e.g. "core-api").
	// Derived deterministically from the service name and shared across all
	// sub-tasks so every layer uses identical import paths.
	ModulePath  string               `json:"module_path,omitempty"`
	ArchPattern manifest.ArchPattern `json:"arch_pattern,omitempty"`
	EnvConfig   manifest.EnvConfig   `json:"env_config,omitempty"`

	// Data pillar
	Domains      []manifest.DomainDef      `json:"domains,omitempty"`
	Databases    []manifest.DBSourceDef    `json:"databases,omitempty"`
	Cachings     []manifest.CachingConfig  `json:"cachings,omitempty"`
	FileStorages []manifest.FileStorageDef `json:"file_storages,omitempty"`

	// Backend pillar — per-service tasks set Service; others see AllServices.
	Service     *manifest.ServiceDef     `json:"service,omitempty"`
	AllServices []manifest.ServiceDef    `json:"all_services,omitempty"`
	CommLinks   []manifest.CommLink      `json:"comm_links,omitempty"`
	Messaging   *manifest.MessagingConfig `json:"messaging,omitempty"`
	APIGateway  *manifest.APIGatewayConfig `json:"api_gateway,omitempty"`
	Auth        *manifest.AuthConfig     `json:"auth,omitempty"`

	// Contracts pillar
	DTOs       []manifest.DTODef      `json:"dtos,omitempty"`
	Endpoints  []manifest.EndpointDef `json:"endpoints,omitempty"`
	Versioning manifest.APIVersioning `json:"versioning,omitempty"`

	// Frontend pillar
	Frontend *manifest.FrontendPillar `json:"frontend,omitempty"`

	// Infrastructure pillar
	Infra *manifest.InfraPillar `json:"infra,omitempty"`

	// Cross-cutting pillar
	CrossCut *manifest.CrossCutPillar `json:"cross_cut,omitempty"`
}
