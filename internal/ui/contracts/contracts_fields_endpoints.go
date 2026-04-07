package contracts

import "github.com/vibe-menu/internal/ui/core"

// ── endpoint / versioning / external API field definitions ───────────────────

func defaultEndpointFormFields(serviceOptions, dtoOptions, roleOptions []string) []core.Field {
	// Ensure at least empty slice so KindSelect works
	if serviceOptions == nil {
		serviceOptions = []string{}
	}
	if dtoOptions == nil {
		dtoOptions = []string{}
	}
	if roleOptions == nil {
		roleOptions = []string{}
	}
	fields := []core.Field{
		{Key: "service_unit", Label: "service_unit  ", Kind: core.KindSelect,
			Options: serviceOptions,
			Value:   core.PlaceholderFor(serviceOptions, "(no services configured)"),
		},
		{Key: "name_path", Label: "name_path     ", Kind: core.KindText},
		{
			Key: "protocol", Label: "protocol      ", Kind: core.KindSelect,
			Options: []string{"REST", "GraphQL", "gRPC", "WebSocket message", "Event"},
			Value:   "REST",
		},
		{
			Key: "auth_required", Label: "auth_required ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{Key: "auth_roles", Label: "auth_roles    ", Kind: core.KindMultiSelect,
			Options: roleOptions,
			Value:   core.PlaceholderFor(roleOptions, "(no roles configured)"),
		},
		{Key: "request_dto", Label: "request_dto   ", Kind: core.KindSelect,
			Options: dtoOptions,
			Value:   core.PlaceholderFor(dtoOptions, "(no DTOs configured)"),
		},
		{Key: "response_dto", Label: "response_dto  ", Kind: core.KindSelect,
			Options: dtoOptions,
			Value:   core.PlaceholderFor(dtoOptions, "(no DTOs configured)"),
		},
		{
			Key: "http_method", Label: "http_method   ", Kind: core.KindSelect,
			Options: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Value:   "GET",
		},
		{
			Key: "graphql_op_type", Label: "Operation     ", Kind: core.KindSelect,
			Options: []string{"Query", "Mutation", "Subscription"},
			Value:   "Query",
		},
		{
			Key: "grpc_stream_type", Label: "Stream Type   ", Kind: core.KindSelect,
			Options: []string{"Unary", "Server stream", "Client stream", "Bidirectional"},
			Value:   "Unary",
		},
		{
			Key: "ws_direction", Label: "WS Direction  ", Kind: core.KindSelect,
			Options: []string{"Client→Server", "Server→Client", "Bidirectional"},
			Value:   "Bidirectional", SelIdx: 2,
		},
		{
			Key: "pagination", Label: "Pagination    ", Kind: core.KindSelect,
			Options: []string{"Cursor-based", "Offset/limit", "Keyset", "Page number", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "rate_limit", Label: "Rate Limit    ", Kind: core.KindSelect,
			Options: []string{"Default (global)", "Strict", "Relaxed", "None"},
			Value:   "Default (global)",
		},
		{Key: "description", Label: "description   ", Kind: core.KindText},
	}
	return fields
}

// versioningByProtocol maps each versionable endpoint protocol to its valid
// versioning strategies.
var versioningByProtocol = map[string][]string{
	"REST":    {"URL path (/v1/)", "Header (Accept-Version)", "Query param", "None"},
	"GraphQL": {"Schema evolution", "None"},
	"gRPC":    {"Package versioning", "None"},
}

// versioningStrategyFieldKey returns the field key used to store the versioning
// strategy for a given protocol.
func versioningStrategyFieldKey(proto string) string {
	return "strategy_" + proto
}

// versioningStrategyLabel returns the label for a per-protocol strategy field.
func versioningStrategyLabel(proto string) string {
	switch proto {
	case "REST":
		return "REST strategy  "
	case "GraphQL":
		return "GraphQL strat  "
	case "gRPC":
		return "gRPC strategy  "
	default:
		return proto + " strategy "
	}
}

// defaultVersioningTailFields returns the non-strategy versioning fields
// (current version + deprecation policy) with their defaults.
func defaultVersioningTailFields() []core.Field {
	return []core.Field{
		{Key: "current_version", Label: "current_ver   ", Kind: core.KindText, Value: "v1"},
		{
			Key: "deprecation", Label: "deprecation   ", Kind: core.KindSelect,
			Options: []string{
				"None", "Sunset header", "Versioned removal notice", "Changelog entry", "Custom",
			},
			Value: "None",
		},
	}
}

// defaultVersioningFields returns an initial versioning field list assuming REST
// as the only protocol. rebuildVersioningFields() replaces this with the
// correct per-protocol set once endpoints are known.
func defaultVersioningFields() []core.Field {
	restOpts := versioningByProtocol["REST"]
	stratField := core.Field{
		Key: versioningStrategyFieldKey("REST"), Label: versioningStrategyLabel("REST"),
		Kind: core.KindSelect, Options: restOpts, Value: restOpts[0],
	}
	return append([]core.Field{stratField}, defaultVersioningTailFields()...)
}

// activeEndpointProtocols returns the distinct versionable protocols (REST,
// GraphQL, gRPC) that have at least one endpoint defined, in a stable order.
func (ce ContractsEditor) activeEndpointProtocols() []string {
	seen := make(map[string]bool)
	order := []string{"REST", "GraphQL", "gRPC"}
	for _, ep := range ce.endpoints {
		switch ep.Protocol {
		case "REST", "GraphQL", "gRPC":
			seen[ep.Protocol] = true
		}
	}
	var result []string
	for _, p := range order {
		if seen[p] {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"REST"} // sensible default when no endpoints exist
	}
	return result
}

// rebuildVersioningFields rebuilds the versioning field slice to show one
// strategy selector per active endpoint protocol, followed by the shared fields
// (current_version, deprecation). Existing values are preserved by key.
func (ce *ContractsEditor) rebuildVersioningFields() {
	// Snapshot current values so we can restore them after rebuild.
	saved := make(map[string]string, len(ce.versioningFields))
	for _, f := range ce.versioningFields {
		saved[f.Key] = f.DisplayValue()
	}

	protos := ce.activeEndpointProtocols()
	var fields []core.Field
	for _, proto := range protos {
		opts := versioningByProtocol[proto]
		key := versioningStrategyFieldKey(proto)
		cur, hasSaved := saved[key]
		f := core.Field{
			Key: key, Label: versioningStrategyLabel(proto),
			Kind: core.KindSelect, Options: opts, Value: opts[0],
		}
		if hasSaved {
			for j, opt := range opts {
				if opt == cur {
					f.SelIdx = j
					f.Value = opt
					break
				}
			}
		}
		fields = append(fields, f)
	}

	tail := defaultVersioningTailFields()
	// Preserve current values for tail fields.
	for i := range tail {
		if v, ok := saved[tail[i].Key]; ok && v != "" {
			tail[i].Value = v
		}
	}
	ce.versioningFields = append(fields, tail...)

	// Clamp cursor.
	if ce.verFormIdx >= len(ce.versioningFields) {
		ce.verFormIdx = len(ce.versioningFields) - 1
	}
}

func defaultExternalAPIFormFields(serviceOptions []string) []core.Field {
	return []core.Field{
		// ── Common ──────────────────────────────────────────────────────────────
		{Key: "provider", Label: "provider      ", Kind: core.KindText},
		{
			Key: "called_by_service", Label: "called_by     ", Kind: core.KindSelect,
			Options: append([]string{"(any / unspecified)"}, serviceOptions...),
			Value:   "(any / unspecified)",
		},
		{Key: "responsibility", Label: "responsibility", Kind: core.KindText},
		{
			Key: "protocol", Label: "protocol      ", Kind: core.KindSelect,
			Options: []string{"REST", "GraphQL", "gRPC", "WebSocket", "Webhook", "SOAP"},
			Value:   "REST",
		},
		{
			Key: "auth_mechanism", Label: "auth_mechanism", Kind: core.KindSelect,
			Options: []string{"API Key", "OAuth2 Client Credentials", "OAuth2 PKCE", "Bearer Token", "Basic Auth", "mTLS", "None"},
			Value:   "API Key",
		},
		{
			Key: "failure_strategy", Label: "failure_strat ", Kind: core.KindSelect,
			Options: []string{"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
			Value:   "Circuit Breaker",
		},

		// ── REST / GraphQL / gRPC / WebSocket / SOAP ────────────────────────────
		{Key: "base_url", Label: "base_url      ", Kind: core.KindText},
		{Key: "rate_limit", Label: "rate_limit    ", Kind: core.KindText},
		{Key: "webhook_endpoint", Label: "webhook_path  ", Kind: core.KindText},

		// ── gRPC ────────────────────────────────────────────────────────────────
		{
			Key: "tls_mode", Label: "tls_mode      ", Kind: core.KindSelect,
			Options: []string{"TLS", "mTLS", "Insecure"},
			Value:   "TLS",
		},

		// ── WebSocket ───────────────────────────────────────────────────────────
		{Key: "ws_subprotocol", Label: "subprotocol   ", Kind: core.KindText},
		{
			Key: "message_format", Label: "message_format", Kind: core.KindSelect,
			Options: []string{"JSON", "MessagePack", "Binary", "Text"},
			Value:   "JSON",
		},

		// ── Webhook (inbound) ───────────────────────────────────────────────────
		{Key: "hmac_header", Label: "hmac_header   ", Kind: core.KindText, Value: "X-Hub-Signature-256"},
		{
			Key: "retry_policy", Label: "retry_policy  ", Kind: core.KindSelect,
			Options: []string{"Retry 3x", "Retry 5x", "Immediate fail", "None"},
			Value:   "Retry 3x",
		},

		// ── SOAP ────────────────────────────────────────────────────────────────
		{
			Key: "soap_version", Label: "soap_version  ", Kind: core.KindSelect,
			Options: []string{"1.1", "1.2"},
			Value:   "1.1",
		},
	}
}

// defaultExtInteractionFormFields returns the form fields for a single
// interaction/call on an external API. Protocol-specific fields are filtered
// by visibleExtIntFormFields before rendering.
func defaultExtInteractionFormFields(dtoOptions []string) []core.Field {
	if dtoOptions == nil {
		dtoOptions = []string{}
	}
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "path", Label: "path          ", Kind: core.KindText},
		{Key: "request_dto", Label: "request_dto   ", Kind: core.KindSelect,
			Options: dtoOptions,
			Value:   core.PlaceholderFor(dtoOptions, "(no DTOs configured)"),
		},
		{Key: "response_dto", Label: "response_dto  ", Kind: core.KindSelect,
			Options: dtoOptions,
			Value:   core.PlaceholderFor(dtoOptions, "(no DTOs configured)"),
		},
		// ── REST ────────────────────────────────────────────────────────────────
		{
			Key: "http_method", Label: "http_method   ", Kind: core.KindSelect,
			Options: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Value:   "GET",
		},
		// ── GraphQL ─────────────────────────────────────────────────────────────
		{
			Key: "gql_operation", Label: "gql_operation ", Kind: core.KindSelect,
			Options: []string{"Query", "Mutation", "Subscription"},
			Value:   "Query",
		},
		// ── gRPC ────────────────────────────────────────────────────────────────
		{
			Key: "grpc_stream_type", Label: "stream_type   ", Kind: core.KindSelect,
			Options: []string{"Unary", "Server streaming", "Client streaming", "Bidirectional"},
			Value:   "Unary",
		},
		// ── WebSocket ───────────────────────────────────────────────────────────
		{
			Key: "ws_direction", Label: "ws_direction  ", Kind: core.KindSelect,
			Options: []string{"Send", "Receive", "Bidirectional"},
			Value:   "Send",
		},
	}
}

// visibleExtFormFields returns only the fields relevant to the currently
// selected protocol, hiding all others.
func (ce ContractsEditor) visibleExtFormFields() []core.Field {
	if len(ce.extForm) == 0 {
		return nil
	}
	proto := core.FieldGet(ce.extForm, "protocol")
	var visible []core.Field
	for _, f := range ce.extForm {
		switch f.Key {
		case "rate_limit":
			if proto != "REST" && proto != "GraphQL" {
				continue
			}
		case "webhook_endpoint":
			if proto != "REST" && proto != "Webhook" {
				continue
			}
		// gRPC-only
		case "tls_mode":
			if proto != "gRPC" {
				continue
			}
		// WebSocket-only
		case "ws_subprotocol", "message_format":
			if proto != "WebSocket" {
				continue
			}
		// Webhook-only
		case "hmac_header", "retry_policy":
			if proto != "Webhook" {
				continue
			}
		// SOAP-only
		case "soap_version":
			if proto != "SOAP" {
				continue
			}
		// base_url: shown for REST, GraphQL, gRPC, WebSocket, SOAP (not Webhook)
		case "base_url":
			if proto == "Webhook" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// visibleExtIntFormFields returns only the interaction form fields relevant to
// the parent external API's protocol.
func (ce ContractsEditor) visibleExtIntFormFields() []core.Field {
	if len(ce.extIntForm) == 0 {
		return nil
	}
	proto := ""
	if ce.extIdx < len(ce.externalAPIs) {
		proto = ce.externalAPIs[ce.extIdx].Protocol
	}
	if proto == "" {
		proto = "REST"
	}
	var visible []core.Field
	for _, f := range ce.extIntForm {
		switch f.Key {
		case "path":
			if proto == "Webhook" {
				continue
			}
		case "http_method":
			if proto != "REST" {
				continue
			}
		case "gql_operation":
			if proto != "GraphQL" {
				continue
			}
		case "grpc_stream_type":
			if proto != "gRPC" {
				continue
			}
		case "ws_direction":
			if proto != "WebSocket" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// extIntFormFieldByKey returns a pointer to the interaction form field with the given key.
func (ce *ContractsEditor) extIntFormFieldByKey(key string) *core.Field {
	for i := range ce.extIntForm {
		if ce.extIntForm[i].Key == key {
			return &ce.extIntForm[i]
		}
	}
	return nil
}

// refreshExtIntDTOOptions updates the request_dto and response_dto option lists
// in the interaction form to match the parent API's protocol.
func (ce *ContractsEditor) refreshExtIntDTOOptions() {
	if ce.extIdx >= len(ce.externalAPIs) {
		return
	}
	proto := ce.externalAPIs[ce.extIdx].Protocol
	if proto == "" {
		proto = "REST"
	}
	opts := ce.dtoNamesForProtocol(proto)
	placeholder := core.PlaceholderFor(opts, "(no matching DTOs)")
	for i := range ce.extIntForm {
		key := ce.extIntForm[i].Key
		if key != "request_dto" && key != "response_dto" {
			continue
		}
		f := &ce.extIntForm[i]
		prev := f.Value
		f.Options = opts
		found := false
		for j, o := range opts {
			if o == prev {
				f.SelIdx = j
				f.Value = o
				found = true
				break
			}
		}
		if !found {
			f.SelIdx = 0
			if len(opts) > 0 {
				f.Value = opts[0]
			} else {
				f.Value = placeholder
			}
		}
	}
}

// extFormFieldByKey returns a pointer to the ext form field with the given key.
func (ce *ContractsEditor) extFormFieldByKey(key string) *core.Field {
	for i := range ce.extForm {
		if ce.extForm[i].Key == key {
			return &ce.extForm[i]
		}
	}
	return nil
}

// failureStrategyByProtocol maps each external API protocol to its valid
// failure strategies. Circuit breaker is only relevant for synchronous
// protocols; async protocols (Webhook) use retry/DLQ semantics instead.
var failureStrategyByProtocol = map[string][]string{
	"REST":      {"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
	"GraphQL":   {"Circuit Breaker", "Retry with backoff", "Fallback value", "Timeout + fail", "None"},
	"gRPC":      {"Circuit Breaker", "Retry with backoff", "Timeout + fail", "None"},
	"WebSocket": {"Reconnect with backoff", "Fallback value", "None"},
	"Webhook":   {"Retry with backoff", "DLQ", "None"},
	"SOAP":      {"Circuit Breaker", "Retry with backoff", "Timeout + fail", "None"},
}

// authMechanismByProtocol maps each external API protocol to the auth
// mechanisms that make sense for it.
var authMechanismByProtocol = map[string][]string{
	"REST":      {"API Key", "OAuth2 Client Credentials", "OAuth2 PKCE", "Bearer Token", "Basic Auth", "mTLS", "None"},
	"GraphQL":   {"API Key", "OAuth2 Client Credentials", "OAuth2 PKCE", "Bearer Token", "Basic Auth", "mTLS", "None"},
	"gRPC":      {"mTLS", "API Key", "Bearer Token", "None"},
	"WebSocket": {"Bearer Token", "API Key", "None"},
	"Webhook":   {"HMAC signature", "API Key", "None"},
	"SOAP":      {"API Key", "OAuth2 Client Credentials", "Bearer Token", "Basic Auth", "mTLS", "None"},
}

// updateExtDependentFields filters failure_strategy and auth_mechanism options
// by protocol, then clamps extFormIdx to the visible field range.
func (ce *ContractsEditor) updateExtDependentFields() {
	protocol := core.FieldGet(ce.extForm, "protocol")

	updateSelectField := func(key string, optsByProto map[string][]string) {
		opts, ok := optsByProto[protocol]
		if !ok {
			return
		}
		for i := range ce.extForm {
			if ce.extForm[i].Key != key {
				continue
			}
			ce.extForm[i].Options = opts
			current := ce.extForm[i].Value
			valid := false
			for j, o := range opts {
				if o == current {
					ce.extForm[i].SelIdx = j
					valid = true
					break
				}
			}
			if !valid {
				ce.extForm[i].Value = opts[0]
				ce.extForm[i].SelIdx = 0
			}
			break
		}
	}

	updateSelectField("failure_strategy", failureStrategyByProtocol)
	updateSelectField("auth_mechanism", authMechanismByProtocol)

	visible := ce.visibleExtFormFields()
	if len(visible) > 0 && ce.extFormIdx >= len(visible) {
		ce.extFormIdx = len(visible) - 1
	}
}

// ── DTO and Endpoint field visibility ────────────────────────────────────────

// protocolsForService returns the endpoint protocols supported by a service
// based on its technology selections in the Backend pillar.
func (ce ContractsEditor) protocolsForService(serviceName string) []string {
	techToProto := map[string]string{
		"REST":           "REST",
		"GraphQL":        "GraphQL",
		"gRPC":           "gRPC",
		"WebSocket":      "WebSocket message",
		"SSE":            "REST",
		"tRPC":           "REST",
		"MQTT":           "Event",
		"Kafka consumer": "Event",
	}
	for _, svc := range ce.availableServiceDefs {
		if svc.Name != serviceName {
			continue
		}
		if len(svc.Technologies) == 0 {
			return nil
		}
		seen := make(map[string]bool)
		var protos []string
		for _, tech := range svc.Technologies {
			if proto, ok := techToProto[tech]; ok && !seen[proto] {
				seen[proto] = true
				protos = append(protos, proto)
			}
		}
		if len(protos) == 0 {
			return nil
		}
		return protos
	}
	return nil
}

// updateEPDependentFields refreshes the protocol options based on the selected
// service unit and clamps epFormIdx to the visible field range.
func (ce *ContractsEditor) updateEPDependentFields() {
	if ce.activeTab != contractsTabEndpoints || ce.epSubView != core.ViewForm {
		return
	}
	svcName := core.FieldGet(ce.epForm, "service_unit")
	protos := ce.protocolsForService(svcName)
	if protos == nil {
		protos = []string{"REST", "GraphQL", "gRPC", "WebSocket message", "Event"}
	}
	for i := range ce.epForm {
		if ce.epForm[i].Key != "protocol" {
			continue
		}
		current := ce.epForm[i].Value
		ce.epForm[i].Options = protos
		found := false
		for j, opt := range protos {
			if opt == current {
				ce.epForm[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			ce.epForm[i].SelIdx = 0
			ce.epForm[i].Value = protos[0]
		}
		break
	}
	// Clamp cursor to visible range
	visible := ce.visibleEPFields()
	if len(visible) > 0 && ce.epFormIdx >= len(visible) {
		ce.epFormIdx = len(visible) - 1
	}
}

// visibleDTOFields returns only the DTO form fields relevant to the selected
// serialisation protocol, hiding the other protocol-specific fields.
func (ce ContractsEditor) visibleDTOFields() []core.Field {
	if len(ce.dtoForm) == 0 {
		return nil
	}
	proto := core.FieldGet(ce.dtoForm, "protocol")
	var visible []core.Field
	for _, f := range ce.dtoForm {
		switch f.Key {
		case "proto_package", "proto_syntax", "proto_options":
			if proto != "Protobuf" {
				continue
			}
		case "avro_namespace", "schema_registry":
			if proto != "Avro" {
				continue
			}
		case "thrift_namespace", "thrift_language":
			if proto != "Thrift" {
				continue
			}
		case "namespace":
			if proto != "FlatBuffers" && proto != "Cap'n Proto" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// dtoFormFieldByKey returns a pointer to the DTO form field with the given key.
func (ce *ContractsEditor) dtoFormFieldByKey(key string) *core.Field {
	for i := range ce.dtoForm {
		if ce.dtoForm[i].Key == key {
			return &ce.dtoForm[i]
		}
	}
	return nil
}

// updateDTODependentFields clamps dtoFormIdx to the visible field range after a
// protocol change so the cursor never lands on a hidden field.
func (ce *ContractsEditor) updateDTODependentFields() {
	if ce.activeTab != contractsTabDTOs || ce.dtoSubView != core.ViewForm {
		return
	}
	visible := ce.visibleDTOFields()
	if len(visible) > 0 && ce.dtoFormIdx >= len(visible) {
		ce.dtoFormIdx = len(visible) - 1
	}
}

// visibleEPFields returns only the endpoint form fields relevant to the
// currently selected protocol and auth setting.
func (ce ContractsEditor) visibleEPFields() []core.Field {
	if len(ce.epForm) == 0 {
		return nil
	}
	proto := core.FieldGet(ce.epForm, "protocol")
	authRequired := core.FieldGet(ce.epForm, "auth_required")
	var visible []core.Field
	for _, f := range ce.epForm {
		switch f.Key {
		case "auth_roles":
			if authRequired != "true" {
				continue
			}
		case "http_method":
			if proto != "REST" {
				continue
			}
		case "graphql_op_type":
			if proto != "GraphQL" {
				continue
			}
		case "grpc_stream_type":
			if proto != "gRPC" {
				continue
			}
		case "ws_direction":
			if proto != "WebSocket message" {
				continue
			}
		case "pagination":
			if proto == "WebSocket message" || proto == "gRPC" || proto == "Event" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// epFieldByKey returns a pointer to the endpoint form field with the given key.
func (ce *ContractsEditor) epFieldByKey(key string) *core.Field {
	for i := range ce.epForm {
		if ce.epForm[i].Key == key {
			return &ce.epForm[i]
		}
	}
	return nil
}
