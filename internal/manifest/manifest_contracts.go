package manifest

// ── Contracts tab types ───────────────────────────────────────────────────────

// DTOField describes a single field within a DTO.
type DTOField struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Nullable   bool   `json:"nullable"`
	Validation string `json:"validation,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// DTODef describes a Data Transfer Object.
type DTODef struct {
	Name          string     `json:"name"`
	Category      string     `json:"category"`
	SourceDomains string     `json:"source_domains,omitempty"`
	Description   string     `json:"description,omitempty"`
	Fields        []DTOField `json:"fields,omitempty"`
}

// EndpointDef describes an API endpoint or operation.
type EndpointDef struct {
	ServiceUnit        string `json:"service_unit"`
	NamePath           string `json:"name_path"`
	Protocol           string `json:"protocol"`
	AuthRequired       string `json:"auth_required"`
	RequestDTO         string `json:"request_dto,omitempty"`
	ResponseDTO        string `json:"response_dto,omitempty"`
	HTTPMethod         string `json:"http_method,omitempty"`
	Description        string `json:"description,omitempty"`
	GraphQLOpType      string `json:"graphql_op_type,omitempty"`
	GRPCStreamType     string `json:"grpc_stream_type,omitempty"`
	WSDirection        string `json:"ws_direction,omitempty"`
	PaginationStrategy string `json:"pagination_strategy,omitempty"`
	RateLimit          string `json:"rate_limit,omitempty"`
}

// APIVersioning describes how the API handles versioning.
type APIVersioning struct {
	Strategy           string `json:"strategy"`
	CurrentVersion     string `json:"current_version,omitempty"`
	DeprecationPolicy  string `json:"deprecation_policy,omitempty"`
	PaginationStrategy string `json:"pagination_strategy,omitempty"`
}

// ExternalAPIDef describes a third-party API that the system consumes.
type ExternalAPIDef struct {
	Provider        string `json:"provider"`
	AuthMechanism   string `json:"auth_mechanism"`
	RateLimit       string `json:"rate_limit,omitempty"`
	WebhookEndpoint string `json:"webhook_endpoint,omitempty"`
	FailureStrategy string `json:"failure_strategy"`
	BaseURL         string `json:"base_url,omitempty"`
}

// ContractsPillar groups all contract-related configuration.
type ContractsPillar struct {
	DTOs         []DTODef         `json:"dtos,omitempty"`
	Endpoints    []EndpointDef    `json:"endpoints,omitempty"`
	Versioning   APIVersioning    `json:"versioning"`
	ExternalAPIs []ExternalAPIDef `json:"external_apis,omitempty"`
}
