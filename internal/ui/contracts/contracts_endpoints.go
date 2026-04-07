package contracts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// epRateLimitDefault returns the smart default for a new endpoint's rate_limit
// field based on the backend WAF rate-limit strategy. When no strategy is
// configured ("None" or empty), rate limiting makes no sense at the endpoint
// level, so "None" is returned. Otherwise "Default (global)" defers to the
// WAF policy.
func (ce ContractsEditor) epRateLimitDefault() string {
	if ce.wafRateLimitStrategy == "" || ce.wafRateLimitStrategy == "None" {
		return "None"
	}
	return "Default (global)"
}

// ── Endpoint updates ──────────────────────────────────────────────────────────

func (ce ContractsEditor) updateEndpoints(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.epSubView {
	case core.ViewList:
		return ce.updateEPList(key)
	case core.ViewForm:
		return ce.updateEPForm(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateEPList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.endpoints)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.epIdx < n-1 {
			ce.epIdx++
		}
	case "k", "up":
		if ce.epIdx > 0 {
			ce.epIdx--
		}
	case "u":
		if snap, ok := ce.epsUndo.Pop(); ok {
			ce.endpoints = snap
			if ce.epIdx >= len(ce.endpoints) && ce.epIdx > 0 {
				ce.epIdx = len(ce.endpoints) - 1
			}
		}
	case "a":
		ce.epsUndo.Push(core.CopySlice(ce.endpoints))
		ce.endpoints = append(ce.endpoints, manifest.EndpointDef{})
		ce.epIdx = len(ce.endpoints) - 1
		ce.epForm = defaultEndpointFormFields(ce.availableServices, ce.dtoNames(), ce.availableAuthRoles)
		existing := make([]string, 0, len(ce.endpoints)-1)
		for i, ep := range ce.endpoints {
			if i != ce.epIdx {
				existing = append(existing, ep.NamePath)
			}
		}
		ce.epForm = core.SetFieldValue(ce.epForm, "name_path", core.UniqueName("endpoint", existing))
		ce.epForm = core.SetFieldValue(ce.epForm, "rate_limit", ce.epRateLimitDefault())
		ce.epFormIdx = 0
		ce.epSubView = core.ViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.epsUndo.Push(core.CopySlice(ce.endpoints))
			ce.endpoints = append(ce.endpoints[:ce.epIdx], ce.endpoints[ce.epIdx+1:]...)
			if ce.epIdx > 0 && ce.epIdx >= len(ce.endpoints) {
				ce.epIdx = len(ce.endpoints) - 1
			}
		}
	case "enter":
		if n > 0 {
			ep := ce.endpoints[ce.epIdx]
			ce.epForm = defaultEndpointFormFields(ce.availableServices, ce.dtoNames(), ce.availableAuthRoles)
			ce.epForm = core.SetFieldValue(ce.epForm, "service_unit", ep.ServiceUnit)
			ce.epForm = core.SetFieldValue(ce.epForm, "name_path", ep.NamePath)
			if ep.Protocol != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "protocol", ep.Protocol)
			}
			ce.epForm = core.SetFieldValue(ce.epForm, "auth_required", ep.AuthRequired)
			if ep.AuthRoles != "" {
				for i := range ce.epForm {
					if ce.epForm[i].Key != "auth_roles" {
						continue
					}
					for _, sel := range strings.Split(ep.AuthRoles, ", ") {
						for j, opt := range ce.epForm[i].Options {
							if opt == strings.TrimSpace(sel) {
								ce.epForm[i].SelectedIdxs = append(ce.epForm[i].SelectedIdxs, j)
							}
						}
					}
					break
				}
			}
			ce.epForm = core.SetFieldValue(ce.epForm, "request_dto", ep.RequestDTO)
			ce.epForm = core.SetFieldValue(ce.epForm, "response_dto", ep.ResponseDTO)
			if ep.HTTPMethod != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "http_method", ep.HTTPMethod)
			}
			if ep.GraphQLOpType != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "graphql_op_type", ep.GraphQLOpType)
			}
			if ep.GRPCStreamType != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "grpc_stream_type", ep.GRPCStreamType)
			}
			if ep.WSDirection != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "ws_direction", ep.WSDirection)
			}
			if ep.PaginationStrategy != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "pagination", ep.PaginationStrategy)
			}
			if ep.RateLimit != "" {
				ce.epForm = core.SetFieldValue(ce.epForm, "rate_limit", ep.RateLimit)
			} else {
				ce.epForm = core.SetFieldValue(ce.epForm, "rate_limit", ce.epRateLimitDefault())
			}
			ce.epForm = core.SetFieldValue(ce.epForm, "description", ep.Description)
			ce.epFormIdx = 0
			ce.epSubView = core.ViewForm
		}
	}
	return ce, nil
}

func (ce ContractsEditor) updateEPForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleEPFields()
	n := len(visible)
	switch key.String() {
	case "j", "down":
		if ce.epFormIdx < n-1 {
			ce.epFormIdx++
		}
	case "k", "up":
		if ce.epFormIdx > 0 {
			ce.epFormIdx--
		}
	case "enter", " ":
		if ce.epFormIdx < n {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
				ce.dd.Open = true
				if f.Kind == core.KindMultiSelect {
					ce.dd.OptIdx = f.DDCursor
				} else {
					ce.dd.OptIdx = f.SelIdx
				}
			} else {
				return ce.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ce.epFormIdx < n {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.Kind == core.KindSelect {
				f.CyclePrev()
				ce.updateEPDependentFields()
			}
		}
	case "i", "a":
		if ce.epFormIdx < n {
			f := ce.epFieldByKey(visible[ce.epFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				return ce.tryEnterInsert()
			}
		}
	case "b", "esc":
		ce.saveEPForm()
		ce.epSubView = core.ViewList
	}
	ce.saveEPForm()
	return ce, nil
}

func (ce *ContractsEditor) saveEPForm() {
	if ce.epIdx >= len(ce.endpoints) {
		return
	}
	ep := &ce.endpoints[ce.epIdx]
	ep.ServiceUnit = core.FieldGet(ce.epForm, "service_unit")
	ep.NamePath = core.FieldGet(ce.epForm, "name_path")
	ep.Protocol = core.FieldGet(ce.epForm, "protocol")
	ep.AuthRequired = core.FieldGet(ce.epForm, "auth_required")
	ep.AuthRoles = core.FieldGetMulti(ce.epForm, "auth_roles")
	ep.RequestDTO = core.FieldGet(ce.epForm, "request_dto")
	ep.ResponseDTO = core.FieldGet(ce.epForm, "response_dto")
	ep.HTTPMethod = core.FieldGet(ce.epForm, "http_method")
	ep.GraphQLOpType = core.FieldGet(ce.epForm, "graphql_op_type")
	ep.GRPCStreamType = core.FieldGet(ce.epForm, "grpc_stream_type")
	ep.WSDirection = core.FieldGet(ce.epForm, "ws_direction")
	proto := core.FieldGet(ce.epForm, "protocol")
	if proto == "WebSocket message" || proto == "gRPC" || proto == "Event" {
		ep.PaginationStrategy = ""
	} else {
		ep.PaginationStrategy = core.FieldGet(ce.epForm, "pagination")
	}
	ep.RateLimit = core.FieldGet(ce.epForm, "rate_limit")
	ep.Description = core.FieldGet(ce.epForm, "description")
}

// ── Versioning update ─────────────────────────────────────────────────────────

func (ce ContractsEditor) updateVersioning(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	if !ce.versioningEnabled {
		if key.String() == "a" {
			ce.versioningEnabled = true
			ce.verFormIdx = 0
			ce.rebuildVersioningFields()
		}
		return ce, nil
	}
	switch key.String() {
	case "j", "down":
		if ce.verFormIdx < len(ce.versioningFields)-1 {
			ce.verFormIdx++
		}
	case "k", "up":
		if ce.verFormIdx > 0 {
			ce.verFormIdx--
		}
	case "enter", " ":
		f := &ce.versioningFields[ce.verFormIdx]
		if f.Kind == core.KindSelect && len(f.Options) > 0 {
			ce.dd.Open = true
			ce.dd.OptIdx = f.SelIdx
		} else {
			return ce.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &ce.versioningFields[ce.verFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "D":
		ce.versioningEnabled = false
		ce.versioningFields = defaultVersioningFields()
		ce.verFormIdx = 0
	case "i", "a":
		if ce.versioningFields[ce.verFormIdx].CanEditAsText() {
			return ce.tryEnterInsert()
		}
	}
	return ce, nil
}

// ── External APIs updates ─────────────────────────────────────────────────────

func (ce ContractsEditor) updateExternal(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	switch ce.extSubView {
	case core.ViewList:
		return ce.updateExtList(key)
	case core.ViewForm:
		return ce.updateExtForm(key)
	case core.ViewSubList:
		return ce.updateExtSubList(key)
	case core.ViewSubForm:
		return ce.updateExtSubForm(key)
	}
	return ce, nil
}

func (ce ContractsEditor) updateExtList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	n := len(ce.externalAPIs)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.extIdx < n-1 {
			ce.extIdx++
		}
	case "k", "up":
		if ce.extIdx > 0 {
			ce.extIdx--
		}
	case "u":
		if snap, ok := ce.extUndo.Pop(); ok {
			ce.externalAPIs = snap
			if ce.extIdx >= len(ce.externalAPIs) && ce.extIdx > 0 {
				ce.extIdx = len(ce.externalAPIs) - 1
			}
		}
	case "a":
		ce.extUndo.Push(core.CopySlice(ce.externalAPIs))
		ce.externalAPIs = append(ce.externalAPIs, manifest.ExternalAPIDef{})
		ce.extIdx = len(ce.externalAPIs) - 1
		ce.extForm = defaultExternalAPIFormFields(ce.availableServices)
		existing := make([]string, 0, len(ce.externalAPIs)-1)
		for i, api := range ce.externalAPIs {
			if i != ce.extIdx {
				existing = append(existing, api.Provider)
			}
		}
		ce.extForm = core.SetFieldValue(ce.extForm, "provider", core.UniqueName("api", existing))
		ce.extFormIdx = 0
		ce.extSubView = core.ViewForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.extUndo.Push(core.CopySlice(ce.externalAPIs))
			ce.externalAPIs = append(ce.externalAPIs[:ce.extIdx], ce.externalAPIs[ce.extIdx+1:]...)
			if ce.extIdx > 0 && ce.extIdx >= len(ce.externalAPIs) {
				ce.extIdx = len(ce.externalAPIs) - 1
			}
		}
	case "enter":
		if n > 0 {
			api := ce.externalAPIs[ce.extIdx]
			ce.extForm = defaultExternalAPIFormFields(ce.availableServices)
			ce.extForm = core.SetFieldValue(ce.extForm, "provider", api.Provider)
			if api.CalledByService != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "called_by_service", api.CalledByService)
			}
			ce.extForm = core.SetFieldValue(ce.extForm, "responsibility", api.Responsibility)
			if api.Protocol != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "protocol", api.Protocol)
			}
			ce.extForm = core.SetFieldValue(ce.extForm, "auth_mechanism", api.AuthMechanism)
			ce.extForm = core.SetFieldValue(ce.extForm, "failure_strategy", api.FailureStrategy)
			// REST / shared
			ce.extForm = core.SetFieldValue(ce.extForm, "base_url", api.BaseURL)
			ce.extForm = core.SetFieldValue(ce.extForm, "rate_limit", api.RateLimit)
			ce.extForm = core.SetFieldValue(ce.extForm, "webhook_endpoint", api.WebhookEndpoint)
			// gRPC
			if api.TLSMode != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "tls_mode", api.TLSMode)
			}
			// WebSocket
			ce.extForm = core.SetFieldValue(ce.extForm, "ws_subprotocol", api.WSSubprotocol)
			if api.MessageFormat != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "message_format", api.MessageFormat)
			}
			// Webhook
			if api.HMACHeader != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "hmac_header", api.HMACHeader)
			}
			if api.RetryPolicy != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "retry_policy", api.RetryPolicy)
			}
			// SOAP
			if api.SOAPVersion != "" {
				ce.extForm = core.SetFieldValue(ce.extForm, "soap_version", api.SOAPVersion)
			}
			// Filter auth_mechanism and failure_strategy options for the loaded protocol.
			ce.updateExtDependentFields()
			ce.extFormIdx = 0
			ce.extSubView = core.ViewForm
		}
	}
	return ce, nil
}

func (ce ContractsEditor) updateExtForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleExtFormFields()
	n := len(visible)
	switch key.String() {
	case "j", "down":
		if ce.extFormIdx < n-1 {
			ce.extFormIdx++
		}
	case "k", "up":
		if ce.extFormIdx > 0 {
			ce.extFormIdx--
		}
	case "enter", " ":
		if ce.extFormIdx < n {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
				ce.dd.Open = true
				if f.Kind == core.KindMultiSelect {
					ce.dd.OptIdx = f.DDCursor
				} else {
					ce.dd.OptIdx = f.SelIdx
				}
			} else {
				return ce.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ce.extFormIdx < n {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && f.Kind == core.KindSelect {
				f.CyclePrev()
				ce.updateExtDependentFields()
			}
		}
	case "i", "a":
		if ce.extFormIdx < n {
			f := ce.extFormFieldByKey(visible[ce.extFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				return ce.tryEnterInsert()
			}
		}
	case "I":
		ce.saveExtForm()
		ce.extIntIdx = 0
		ce.extSubView = core.ViewSubList
		return ce, nil
	case "b", "esc":
		ce.saveExtForm()
		ce.extSubView = core.ViewList
	}
	ce.saveExtForm()
	return ce, nil
}

func (ce *ContractsEditor) saveExtForm() {
	if ce.extIdx >= len(ce.externalAPIs) {
		return
	}
	api := &ce.externalAPIs[ce.extIdx]
	api.Provider = core.FieldGet(ce.extForm, "provider")
	v := core.FieldGet(ce.extForm, "called_by_service")
	if v == "(any / unspecified)" {
		v = ""
	}
	api.CalledByService = v
	api.Responsibility = core.FieldGet(ce.extForm, "responsibility")
	api.Protocol = core.FieldGet(ce.extForm, "protocol")
	api.AuthMechanism = core.FieldGet(ce.extForm, "auth_mechanism")
	api.FailureStrategy = core.FieldGet(ce.extForm, "failure_strategy")
	// REST / shared
	api.BaseURL = core.FieldGet(ce.extForm, "base_url")
	api.RateLimit = core.FieldGet(ce.extForm, "rate_limit")
	api.WebhookEndpoint = core.FieldGet(ce.extForm, "webhook_endpoint")
	// gRPC
	api.TLSMode = core.FieldGet(ce.extForm, "tls_mode")
	// WebSocket
	api.WSSubprotocol = core.FieldGet(ce.extForm, "ws_subprotocol")
	api.MessageFormat = core.FieldGet(ce.extForm, "message_format")
	// Webhook
	api.HMACHeader = core.FieldGet(ce.extForm, "hmac_header")
	api.RetryPolicy = core.FieldGet(ce.extForm, "retry_policy")
	// SOAP
	api.SOAPVersion = core.FieldGet(ce.extForm, "soap_version")
}

func (ce *ContractsEditor) saveExtIntForm() {
	if ce.extIdx >= len(ce.externalAPIs) {
		return
	}
	api := &ce.externalAPIs[ce.extIdx]
	if ce.extIntIdx >= len(api.Interactions) {
		return
	}
	it := &api.Interactions[ce.extIntIdx]
	it.Name = core.FieldGet(ce.extIntForm, "name")
	it.Path = core.FieldGet(ce.extIntForm, "path")
	it.RequestDTO = core.FieldGet(ce.extIntForm, "request_dto")
	it.ResponseDTO = core.FieldGet(ce.extIntForm, "response_dto")
	it.HTTPMethod = core.FieldGet(ce.extIntForm, "http_method")
	it.GQLOperation = core.FieldGet(ce.extIntForm, "gql_operation")
	it.GRPCStreamType = core.FieldGet(ce.extIntForm, "grpc_stream_type")
	it.WSDirection = core.FieldGet(ce.extIntForm, "ws_direction")
}

func (ce ContractsEditor) updateExtSubList(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	if ce.extIdx >= len(ce.externalAPIs) {
		return ce, nil
	}
	interactions := ce.externalAPIs[ce.extIdx].Interactions
	n := len(interactions)
	switch key.String() {
	case "j", "down":
		if n > 0 && ce.extIntIdx < n-1 {
			ce.extIntIdx++
		}
	case "k", "up":
		if ce.extIntIdx > 0 {
			ce.extIntIdx--
		}
	case "a":
		proto := ce.externalAPIs[ce.extIdx].Protocol
		if proto == "" {
			proto = "REST"
		}
		opts := ce.dtoNamesForProtocol(proto)
		ce.externalAPIs[ce.extIdx].Interactions = append(
			ce.externalAPIs[ce.extIdx].Interactions,
			manifest.ExternalAPIInteraction{},
		)
		ce.extIntIdx = len(ce.externalAPIs[ce.extIdx].Interactions) - 1
		ce.extIntForm = defaultExtInteractionFormFields(opts)
		ce.refreshExtIntDTOOptions()
		ce.extIntFormIdx = 0
		ce.extSubView = core.ViewSubForm
		return ce.tryEnterInsert()
	case "d":
		if n > 0 {
			ce.externalAPIs[ce.extIdx].Interactions = append(
				ce.externalAPIs[ce.extIdx].Interactions[:ce.extIntIdx],
				ce.externalAPIs[ce.extIdx].Interactions[ce.extIntIdx+1:]...,
			)
			if ce.extIntIdx > 0 && ce.extIntIdx >= len(ce.externalAPIs[ce.extIdx].Interactions) {
				ce.extIntIdx = len(ce.externalAPIs[ce.extIdx].Interactions) - 1
			}
		}
	case "enter", "i":
		if n > 0 {
			it := interactions[ce.extIntIdx]
			proto := ce.externalAPIs[ce.extIdx].Protocol
			if proto == "" {
				proto = "REST"
			}
			opts := ce.dtoNamesForProtocol(proto)
			ce.extIntForm = defaultExtInteractionFormFields(opts)
			ce.extIntForm = core.SetFieldValue(ce.extIntForm, "name", it.Name)
			ce.extIntForm = core.SetFieldValue(ce.extIntForm, "path", it.Path)
			ce.extIntForm = core.SetFieldValue(ce.extIntForm, "request_dto", it.RequestDTO)
			ce.extIntForm = core.SetFieldValue(ce.extIntForm, "response_dto", it.ResponseDTO)
			if it.HTTPMethod != "" {
				ce.extIntForm = core.SetFieldValue(ce.extIntForm, "http_method", it.HTTPMethod)
			}
			if it.GQLOperation != "" {
				ce.extIntForm = core.SetFieldValue(ce.extIntForm, "gql_operation", it.GQLOperation)
			}
			if it.GRPCStreamType != "" {
				ce.extIntForm = core.SetFieldValue(ce.extIntForm, "grpc_stream_type", it.GRPCStreamType)
			}
			if it.WSDirection != "" {
				ce.extIntForm = core.SetFieldValue(ce.extIntForm, "ws_direction", it.WSDirection)
			}
			ce.refreshExtIntDTOOptions()
			ce.extIntFormIdx = 0
			ce.extSubView = core.ViewSubForm
		}
	case "b", "esc":
		ce.extSubView = core.ViewForm
	}
	return ce, nil
}

func (ce ContractsEditor) updateExtSubForm(key tea.KeyMsg) (ContractsEditor, tea.Cmd) {
	visible := ce.visibleExtIntFormFields()
	n := len(visible)
	switch key.String() {
	case "j", "down":
		if ce.extIntFormIdx < n-1 {
			ce.extIntFormIdx++
		}
	case "k", "up":
		if ce.extIntFormIdx > 0 {
			ce.extIntFormIdx--
		}
	case "enter", " ":
		if ce.extIntFormIdx < n {
			f := ce.extIntFormFieldByKey(visible[ce.extIntFormIdx].Key)
			if f != nil && (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
				ce.dd.Open = true
				if f.Kind == core.KindMultiSelect {
					ce.dd.OptIdx = f.DDCursor
				} else {
					ce.dd.OptIdx = f.SelIdx
				}
			} else {
				return ce.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ce.extIntFormIdx < n {
			f := ce.extIntFormFieldByKey(visible[ce.extIntFormIdx].Key)
			if f != nil && f.Kind == core.KindSelect {
				f.CyclePrev()
			}
		}
	case "i", "a":
		if ce.extIntFormIdx < n {
			f := ce.extIntFormFieldByKey(visible[ce.extIntFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				return ce.tryEnterInsert()
			}
		}
	case "b", "esc":
		ce.saveExtIntForm()
		ce.extSubView = core.ViewSubList
	}
	ce.saveExtIntForm()
	return ce, nil
}

func (ce ContractsEditor) viewExternal(w int) []string {
	switch ce.extSubView {
	case core.ViewList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # External APIs — a: add  d: delete  Enter: edit"), "")
		if len(ce.externalAPIs) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no external APIs yet — press 'a' to add one)"))
		} else {
			for i, api := range ce.externalAPIs {
				name := api.Provider
				if name == "" {
					name = fmt.Sprintf("(api #%d)", i+1)
				}
				subtitle := api.Protocol
				if subtitle == "" {
					subtitle = "REST"
				}
				if api.CalledByService != "" {
					subtitle += " · svc:" + api.CalledByService
				}
				if api.AuthMechanism != "" {
					subtitle += " · " + api.AuthMechanism
				}
				if len(api.Interactions) > 0 {
					subtitle += fmt.Sprintf(" · %d interaction(s)", len(api.Interactions))
				}
				lines = append(lines, core.RenderListItem(w, i == ce.extIdx, "  ▶ ", name, subtitle))
			}
		}
		return lines

	case core.ViewForm:
		provider := core.FieldGet(ce.extForm, "provider")
		if provider == "" {
			provider = "(new external API)"
		}
		proto := core.FieldGet(ce.extForm, "protocol")
		if proto == "" {
			proto = "REST"
		}
		intCount := 0
		if ce.extIdx < len(ce.externalAPIs) {
			intCount = len(ce.externalAPIs[ce.extIdx].Interactions)
		}
		var lines []string
		intHint := fmt.Sprintf("  I: interactions (%d)", intCount)
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(provider)+" "+core.StyleSectionDesc.Render("["+proto+"]")+core.StyleSectionDesc.Render(intHint), "")
		visible := ce.visibleExtFormFields()
		lines = append(lines, core.RenderFormFields(w, visible, ce.extFormIdx, ce.internalMode == core.ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines

	case core.ViewSubList:
		provider := ""
		if ce.extIdx < len(ce.externalAPIs) {
			provider = ce.externalAPIs[ce.extIdx].Provider
		}
		proto := ""
		if ce.extIdx < len(ce.externalAPIs) {
			proto = ce.externalAPIs[ce.extIdx].Protocol
		}
		if proto == "" {
			proto = "REST"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← "+provider+" ["+proto+"] — Interactions — a: add  d: delete  Enter: edit"), "")
		if ce.extIdx >= len(ce.externalAPIs) {
			return lines
		}
		interactions := ce.externalAPIs[ce.extIdx].Interactions
		if len(interactions) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no interactions yet — press 'a' to add one)"))
		} else {
			for i, it := range interactions {
				name := it.Name
				if name == "" {
					name = fmt.Sprintf("(interaction #%d)", i+1)
				}
				subtitle := it.Path
				if it.HTTPMethod != "" && it.Path != "" {
					subtitle = it.HTTPMethod + " " + it.Path
				} else if it.HTTPMethod != "" {
					subtitle = it.HTTPMethod
				} else if it.GQLOperation != "" {
					subtitle = it.GQLOperation
				} else if it.GRPCStreamType != "" {
					subtitle = it.GRPCStreamType
				}
				lines = append(lines, core.RenderListItem(w, i == ce.extIntIdx, "  ▷ ", name, subtitle))
			}
		}
		return lines

	case core.ViewSubForm:
		provider := ""
		if ce.extIdx < len(ce.externalAPIs) {
			provider = ce.externalAPIs[ce.extIdx].Provider
		}
		name := core.FieldGet(ce.extIntForm, "name")
		if name == "" {
			name = "(new interaction)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← "+provider+" ← ")+core.StyleFieldKey.Render(name), "")
		visible := ce.visibleExtIntFormFields()
		lines = append(lines, core.RenderFormFields(w, visible, ce.extIntFormIdx, ce.internalMode == core.ModeInsert, ce.formInput, ce.dd.Open, ce.dd.OptIdx)...)
		return lines
	}
	return nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (ce ContractsEditor) View(w, h int) string {
	ce.width = w
	ce.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		core.StyleSectionDesc.Render("  # Contracts — DTOs, endpoints, and API versioning"),
		"",
		core.RenderSubTabBar(contractsTabLabels, int(ce.activeTab), w),
		"",
	)
	const ceHeaderH = 4

	switch ce.activeTab {
	case contractsTabDTOs:
		dtoLines := ce.viewDTOs(w)
		switch ce.dtoSubView {
		case core.ViewList:
			dtoLines = core.AppendViewport(dtoLines, 2, ce.dtoIdx, h-ceHeaderH)
		case core.ViewForm:
			dtoLines = core.AppendViewport(dtoLines, 2, ce.dtoFormIdx, h-ceHeaderH)
		case core.ViewSubList:
			dtoLines = core.AppendViewport(dtoLines, 2, ce.dtoFieldIdx, h-ceHeaderH)
		case core.ViewSubForm:
			dtoLines = core.AppendViewport(dtoLines, 2, ce.dtoFieldFormIdx, h-ceHeaderH)
		}
		lines = append(lines, dtoLines...)
	case contractsTabEndpoints:
		epLines := ce.viewEndpoints(w)
		switch ce.epSubView {
		case core.ViewList:
			epLines = core.AppendViewport(epLines, 2, ce.epIdx, h-ceHeaderH)
		case core.ViewForm:
			epLines = core.AppendViewport(epLines, 2, ce.epFormIdx, h-ceHeaderH)
		}
		lines = append(lines, epLines...)
	case contractsTabVersioning:
		verLines := ce.viewVersioning(w)
		if ce.versioningEnabled {
			verLines = core.AppendViewport(verLines, 2, ce.verFormIdx, h-ceHeaderH)
		}
		lines = append(lines, verLines...)
	case contractsTabExternal:
		extLines := ce.viewExternal(w)
		switch ce.extSubView {
		case core.ViewList:
			extLines = core.AppendViewport(extLines, 2, ce.extIdx, h-ceHeaderH)
		case core.ViewForm:
			extLines = core.AppendViewport(extLines, 2, ce.extFormIdx, h-ceHeaderH)
		case core.ViewSubList:
			extLines = core.AppendViewport(extLines, 2, ce.extIntIdx, h-ceHeaderH)
		case core.ViewSubForm:
			extLines = core.AppendViewport(extLines, 2, ce.extIntFormIdx, h-ceHeaderH)
		}
		lines = append(lines, extLines...)
	}

	return core.FillTildes(lines, h)
}
