package infra

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// envView is the sub-view state for the Environments list+form.
type envView int

const (
	envViewList envView = iota
	envViewForm
)

// ── InfraEditor ───────────────────────────────────────────────────────────────

// InfraEditor manages the INFRASTRUCTURE main-tab.
type InfraEditor struct {
	activeTab infraTabIdx

	networkingFields []core.Field
	netFormIdx       int
	netEnabled       bool

	cicdFields  []core.Field
	cicdFormIdx int
	cicdEnabled bool

	obsFields  []core.Field
	obsFormIdx int
	obsEnabled bool

	// Environments list+form (replaces the old flat env-topology form).
	envs       []manifest.ServerEnvironmentDef
	envIdx     int
	envView    envView
	envForm    []core.Field
	envFormIdx int

	internalMode core.Mode
	formInput    textinput.Model
	width        int

	// Vim motion state
	nav  core.VimNav
	cBuf bool

	dd core.DropdownState

	// backendLanguages mirrors the languages from the backend services/monolith
	// so that container_runtime options reflect what is actually being built.
	backendLanguages string // joined with "," for cheap equality checks

	// cloudProvider caches the last provider used to narrow networking/cicd/obs
	// option lists, to avoid redundant field-slice rebuilds.
	cloudProvider string

	// Per-subtab undo stack (structural add/delete only)
	envsUndo core.UndoStack[[]manifest.ServerEnvironmentDef]
}

func (ie InfraEditor) activeTabEnabled() bool {
	switch ie.activeTab {
	case infraTabNetworking:
		return ie.netEnabled
	case infraTabCICD:
		return ie.cicdEnabled
	case infraTabObservability:
		return ie.obsEnabled
	case infraTabEnvironments:
		return true // environments list is always accessible
	}
	return false
}

func (ie *InfraEditor) enableActiveTab() {
	switch ie.activeTab {
	case infraTabNetworking:
		ie.netEnabled = true
		ie.netFormIdx = 0
	case infraTabCICD:
		ie.cicdEnabled = true
		ie.cicdFormIdx = 0
	case infraTabObservability:
		ie.obsEnabled = true
		ie.obsFormIdx = 0
	}
}

func (ie *InfraEditor) disableActiveTab() {
	switch ie.activeTab {
	case infraTabNetworking:
		ie.netEnabled = false
		ie.networkingFields = defaultNetworkingFields()
		ie.netFormIdx = 0
	case infraTabCICD:
		ie.cicdEnabled = false
		ie.cicdFields = defaultInfraCICDFields()
		ie.cicdFormIdx = 0
	case infraTabObservability:
		ie.obsEnabled = false
		ie.obsFields = defaultObservabilityFields()
		ie.obsFormIdx = 0
	}
}

// NewEditor creates and returns the initial InfraEditor.
func NewEditor() InfraEditor {
	return InfraEditor{
		networkingFields: defaultNetworkingFields(),
		cicdFields:       defaultInfraCICDFields(),
		obsFields:        defaultObservabilityFields(),
		formInput:        core.NewFormInput(),
	}
}

// EnvironmentNames returns the names of all configured server environments.
// Used by backend and data editors to populate environment selector dropdowns.
func (ie InfraEditor) EnvironmentNames() []string {
	names := make([]string, 0, len(ie.envs))
	for _, e := range ie.envs {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names
}

// EnvironmentDefs returns lightweight records of each configured environment's
// name, orchestrator, and cloud provider. Used by the API Gateway editor to
// filter technology options based on the selected environment.
func (ie InfraEditor) EnvironmentDefs() []manifest.ServerEnvironmentDef {
	out := make([]manifest.ServerEnvironmentDef, 0, len(ie.envs))
	for _, e := range ie.envs {
		if e.Name != "" {
			out = append(out, manifest.ServerEnvironmentDef{
				Name:          e.Name,
				Orchestrator:  e.Orchestrator,
				CloudProvider: e.CloudProvider,
			})
		}
	}
	return out
}

// PrimaryOrchestrator returns the orchestrator of the first configured environment,
// or an empty string when no environments have been defined yet.
func (ie InfraEditor) PrimaryOrchestrator() string {
	for _, e := range ie.envs {
		if e.Orchestrator != "" {
			return e.Orchestrator
		}
	}
	return ""
}

// PrimaryCloudProvider returns the cloud_provider of the first environment,
// for use by other editors that need to narrow their options by cloud provider.
func (ie InfraEditor) PrimaryCloudProvider() string {
	return ie.primaryCloudProvider()
}

// primaryCloudProvider returns the cloud_provider of the first environment for
// narrowing networking/cicd/obs option lists.
func (ie InfraEditor) primaryCloudProvider() string {
	for _, e := range ie.envs {
		if e.CloudProvider != "" {
			return e.CloudProvider
		}
	}
	return ""
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (ie InfraEditor) ToManifestInfraPillar() manifest.InfraPillar {
	var p manifest.InfraPillar
	if ie.netEnabled {
		p.Networking = &manifest.NetworkingConfig{
			DNSProvider:     core.NoneToEmpty(core.FieldGet(ie.networkingFields, "dns_provider")),
			TLSSSL:          core.NoneToEmpty(core.FieldGet(ie.networkingFields, "tls_ssl")),
			ReverseProxy:    core.NoneToEmpty(core.FieldGet(ie.networkingFields, "reverse_proxy")),
			CDN:             core.NoneToEmpty(core.FieldGet(ie.networkingFields, "cdn")),
			PrimaryDomain:   core.FieldGet(ie.networkingFields, "primary_domain"),
			DomainStrategy:  core.NoneToEmpty(core.FieldGet(ie.networkingFields, "domain_strategy")),
			CORSEnforcement: core.NoneToEmpty(core.FieldGet(ie.networkingFields, "cors_infra")),
			CORSStrategy:    core.NoneToEmpty(core.FieldGet(ie.networkingFields, "cors_strategy")),
			CORSOrigins:     core.FieldGet(ie.networkingFields, "cors_origins"),
			SSLCertMgmt:     core.NoneToEmpty(core.FieldGet(ie.networkingFields, "ssl_cert")),
		}
	}
	if ie.cicdEnabled {
		p.CICD = &manifest.CICDConfig{
			Platform:          core.NoneToEmpty(core.FieldGet(ie.cicdFields, "platform")),
			ContainerRegistry: core.NoneToEmpty(core.FieldGet(ie.cicdFields, "registry")),
			DeployStrategy:    core.NoneToEmpty(core.FieldGet(ie.cicdFields, "deploy_strategy")),
			IaCTool:           core.NoneToEmpty(core.FieldGet(ie.cicdFields, "iac_tool")),
			SecretsMgmt:       core.NoneToEmpty(core.FieldGet(ie.cicdFields, "secrets_mgmt")),
			ContainerRuntime:  core.NoneToEmpty(core.FieldGet(ie.cicdFields, "container_runtime")),
			BackupDR:          core.NoneToEmpty(core.FieldGet(ie.cicdFields, "backup_dr")),
		}
	}
	if ie.obsEnabled {
		p.Observability = &manifest.ObservabilityConfig{
			Logging:       core.NoneToEmpty(core.FieldGet(ie.obsFields, "logging")),
			Metrics:       core.NoneToEmpty(core.FieldGet(ie.obsFields, "metrics")),
			Tracing:       core.NoneToEmpty(core.FieldGet(ie.obsFields, "tracing")),
			ErrorTracking: core.NoneToEmpty(core.FieldGet(ie.obsFields, "error_tracking")),
			HealthChecks:  core.FieldGet(ie.obsFields, "health_checks") == "true",
			Alerting:      core.NoneToEmpty(core.FieldGet(ie.obsFields, "alerting")),
			LogRetention:  core.FieldGet(ie.obsFields, "log_retention"),
		}
	}
	p.Environments = ie.envs
	return p
}

// FromInfraPillar populates the editor from a saved manifest InfraPillar,
// reversing the ToManifestInfraPillar() operation.
func (ie InfraEditor) FromInfraPillar(ip manifest.InfraPillar) InfraEditor {
	if ip.Networking != nil && (ip.Networking.DNSProvider != "" || ip.Networking.ReverseProxy != "" || ip.Networking.CDN != "") {
		ie.netEnabled = true
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "dns_provider", ip.Networking.DNSProvider)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "tls_ssl", ip.Networking.TLSSSL)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "reverse_proxy", ip.Networking.ReverseProxy)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "cdn", ip.Networking.CDN)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "primary_domain", ip.Networking.PrimaryDomain)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "domain_strategy", ip.Networking.DomainStrategy)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "cors_infra", ip.Networking.CORSEnforcement)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "cors_strategy", ip.Networking.CORSStrategy)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "cors_origins", ip.Networking.CORSOrigins)
		ie.networkingFields = core.SetFieldValue(ie.networkingFields, "ssl_cert", ip.Networking.SSLCertMgmt)
	}

	if ip.CICD != nil && ip.CICD.Platform != "" {
		ie.cicdEnabled = true
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "platform", ip.CICD.Platform)
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "registry", ip.CICD.ContainerRegistry)
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "deploy_strategy", ip.CICD.DeployStrategy)
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "iac_tool", ip.CICD.IaCTool)
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "secrets_mgmt", ip.CICD.SecretsMgmt)
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "container_runtime", ip.CICD.ContainerRuntime)
		ie.cicdFields = core.SetFieldValue(ie.cicdFields, "backup_dr", ip.CICD.BackupDR)
	}

	if o := ip.Observability; o != nil && (o.Logging != "" || o.Metrics != "") {
		ie.obsEnabled = true
		ie.obsFields = core.SetFieldValue(ie.obsFields, "logging", o.Logging)
		ie.obsFields = core.SetFieldValue(ie.obsFields, "metrics", o.Metrics)
		// Narrow alerting/tracing options to those compatible with the saved metrics
		// backend before restoring their values, so SelIdx is computed from the
		// correct (narrowed) option list.
		ie.obsFields = applyMetricsToObsFields(ie.obsFields)
		ie.obsFields = core.SetFieldValue(ie.obsFields, "tracing", o.Tracing)
		ie.obsFields = core.SetFieldValue(ie.obsFields, "error_tracking", o.ErrorTracking)
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}
		ie.obsFields = core.SetFieldValue(ie.obsFields, "health_checks", boolStr(o.HealthChecks))
		ie.obsFields = core.SetFieldValue(ie.obsFields, "alerting", o.Alerting)
		ie.obsFields = core.SetFieldValue(ie.obsFields, "log_retention", o.LogRetention)
	}

	if len(ip.Environments) > 0 {
		ie.envs = ip.Environments
		ie.envIdx = 0
		ie.envView = envViewList
	}

	return ie
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (ie InfraEditor) Mode() core.Mode {
	if ie.internalMode == core.ModeInsert {
		return core.ModeInsert
	}
	return core.ModeNormal
}

func (ie InfraEditor) HintLine() string {
	if ie.internalMode == core.ModeInsert {
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	if ie.dd.Open {
		return core.HintBar("j/k", "navigate", "Enter/Space", "select", "Esc", "cancel")
	}
	if ie.activeTab == infraTabEnvironments {
		switch ie.envView {
		case envViewForm:
			return core.HintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "H", "cycle back", "b/Esc", "save & back", "h/l", "sub-tab")
		default:
			return core.HintBar("j/k", "navigate", "a", "add env", "d", "delete", "u", "undo", "Enter", "edit", "h/l", "sub-tab")
		}
	}
	if !ie.activeTabEnabled() {
		return core.HintBar("a", "configure", "h/l", "sub-tab")
	}
	return core.HintBar("j/k", "navigate", "gg/G", "top/bottom", "[n]j/k", "jump n lines", "Enter/Space", "dropdown", "H", "cycle back", "D", "delete config", "a/i", "edit text", "h/l", "sub-tab")
}

// ── Update ────────────────────────────────────────────────────────────────────

func (ie InfraEditor) Update(msg tea.Msg) (InfraEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		ie.width = wsz.Width
		ie.formInput.Width = wsz.Width - 22
		return ie, nil
	}
	if ie.internalMode == core.ModeInsert {
		return ie.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return ie, nil
	}

	if ie.dd.Open && ie.activeTab != infraTabEnvironments {
		return ie.updateInfraDropdown(key)
	}

	switch key.String() {
	case "h", "left", "l", "right":
		ie.activeTab = infraTabIdx(core.NavigateTab(key.String(), int(ie.activeTab), len(infraTabLabels)))
		return ie, nil
	}

	// cc detection: clear field and enter insert mode
	if !ie.dd.Open {
		if key.String() == "c" {
			if ie.cBuf {
				ie.cBuf = false
				return ie.clearAndEnterInsert()
			}
			ie.cBuf = true
			return ie, nil
		}
		ie.cBuf = false
	}

	switch ie.activeTab {
	case infraTabNetworking:
		return ie.updateFields(key)
	case infraTabCICD:
		return ie.updateFields(key)
	case infraTabObservability:
		return ie.updateFields(key)
	case infraTabEnvironments:
		return ie.updateEnvTab(key)
	}
	return ie, nil
}

func (ie InfraEditor) updateFields(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	if !ie.activeTabEnabled() {
		if key.String() == "a" {
			ie.enableActiveTab()
		}
		return ie, nil
	}
	var fields []core.Field
	var idx int
	switch ie.activeTab {
	case infraTabNetworking:
		fields, idx = ie.networkingFields, ie.netFormIdx
	case infraTabCICD:
		fields, idx = ie.cicdFields, ie.cicdFormIdx
	case infraTabObservability:
		fields, idx = ie.obsFields, ie.obsFormIdx
	default:
		return ie, nil
	}
	n := len(fields)
	k := key.String()

	metricsChanged := false
	if newIdx, consumed := ie.nav.Handle(k, idx, n); consumed {
		idx = newIdx
	} else {
		ie.nav.Reset()
		switch k {
		case "enter", " ":
			if idx < n {
				f := &fields[idx]
				if f.Kind == core.KindSelect && len(f.Options) > 0 {
					ie.dd.Open = true
					ie.dd.OptIdx = f.SelIdx
				} else {
					switch ie.activeTab {
					case infraTabNetworking:
						ie.networkingFields = fields
						ie.netFormIdx = idx
					case infraTabCICD:
						ie.cicdFields = fields
						ie.cicdFormIdx = idx
					case infraTabObservability:
						ie.obsFields = fields
						ie.obsFormIdx = idx
					}
					return ie.tryEnterInsert()
				}
			}
		case "H", "shift+left":
			if idx < n {
				f := &fields[idx]
				if f.Kind == core.KindSelect {
					prev := f.Value
					f.CyclePrev()
					if ie.activeTab == infraTabObservability && f.Key == "metrics" && f.Value != prev {
						metricsChanged = true
					}
				}
			}
		case "D":
			ie.disableActiveTab()
			return ie, nil
		case "i", "a":
			switch ie.activeTab {
			case infraTabNetworking:
				ie.networkingFields = fields
				ie.netFormIdx = idx
			case infraTabCICD:
				ie.cicdFields = fields
				ie.cicdFormIdx = idx
			case infraTabObservability:
				ie.obsFields = fields
				ie.obsFormIdx = idx
			}
			return ie.tryEnterInsert()
		}
	}
	// Write back updated fields and index
	switch ie.activeTab {
	case infraTabNetworking:
		ie.networkingFields = fields
		ie.netFormIdx = idx
	case infraTabCICD:
		ie.cicdFields = fields
		ie.cicdFormIdx = idx
	case infraTabObservability:
		ie.obsFields = fields
		ie.obsFormIdx = idx
		if metricsChanged {
			ie.obsFields = applyMetricsToObsFields(ie.obsFields)
		}
	}
	return ie, nil
}

func (ie InfraEditor) updateInfraDropdown(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	var f *core.Field
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netFormIdx < len(ie.networkingFields) {
			f = &ie.networkingFields[ie.netFormIdx]
		}
	case infraTabCICD:
		if ie.cicdFormIdx < len(ie.cicdFields) {
			f = &ie.cicdFields[ie.cicdFormIdx]
		}
	case infraTabObservability:
		if ie.obsFormIdx < len(ie.obsFields) {
			f = &ie.obsFields[ie.obsFormIdx]
		}
	}
	if f == nil {
		ie.dd.Open = false
		return ie, nil
	}
	ie.dd.OptIdx = core.NavigateDropdown(key.String(), ie.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = ie.dd.OptIdx
		if ie.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[ie.dd.OptIdx]
		}
		ie.dd.Open = false
		if ie.activeTab == infraTabObservability && f.Key == "metrics" {
			ie.obsFields = applyMetricsToObsFields(ie.obsFields)
		}
		if f.PrepareCustomEntry() {
			return ie.tryEnterInsert()
		}
	case "esc", "b":
		ie.dd.Open = false
	}
	return ie, nil
}

func (ie InfraEditor) updateInsert(msg tea.Msg) (InfraEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			ie.saveInput()
			ie.internalMode = core.ModeNormal
			ie.formInput.Blur()
			return ie, nil
		case "tab":
			ie.saveInput()
			ie.advanceField(1)
			return ie.tryEnterInsert()
		case "shift+tab":
			ie.saveInput()
			ie.advanceField(-1)
			return ie.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	ie.formInput, cmd = ie.formInput.Update(msg)
	return ie, cmd
}

func (ie *InfraEditor) advanceField(delta int) {
	switch ie.activeTab {
	case infraTabNetworking:
		n := len(ie.networkingFields)
		if n > 0 {
			ie.netFormIdx = (ie.netFormIdx + delta + n) % n
		}
	case infraTabCICD:
		n := len(ie.cicdFields)
		if n > 0 {
			ie.cicdFormIdx = (ie.cicdFormIdx + delta + n) % n
		}
	case infraTabObservability:
		n := len(ie.obsFields)
		if n > 0 {
			ie.obsFormIdx = (ie.obsFormIdx + delta + n) % n
		}
	case infraTabEnvironments:
		n := len(ie.envForm)
		if n > 0 {
			ie.envFormIdx = (ie.envFormIdx + delta + n) % n
		}
	}
}

func (ie *InfraEditor) saveInput() {
	val := ie.formInput.Value()
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netFormIdx < len(ie.networkingFields) && ie.networkingFields[ie.netFormIdx].CanEditAsText() {
			ie.networkingFields[ie.netFormIdx].SaveTextInput(val)
		}
	case infraTabCICD:
		if ie.cicdFormIdx < len(ie.cicdFields) && ie.cicdFields[ie.cicdFormIdx].CanEditAsText() {
			ie.cicdFields[ie.cicdFormIdx].SaveTextInput(val)
		}
	case infraTabObservability:
		if ie.obsFormIdx < len(ie.obsFields) && ie.obsFields[ie.obsFormIdx].CanEditAsText() {
			ie.obsFields[ie.obsFormIdx].SaveTextInput(val)
		}
	case infraTabEnvironments:
		if ie.envFormIdx < len(ie.envForm) && ie.envForm[ie.envFormIdx].CanEditAsText() {
			ie.envForm[ie.envFormIdx].SaveTextInput(val)
		}
	}
}

func (ie InfraEditor) clearAndEnterInsert() (InfraEditor, tea.Cmd) {
	ie, cmd := ie.tryEnterInsert()
	if ie.internalMode == core.ModeInsert {
		ie.formInput.SetValue("")
	}
	return ie, cmd
}

func (ie InfraEditor) tryEnterInsert() (InfraEditor, tea.Cmd) {
	n := 0
	switch ie.activeTab {
	case infraTabNetworking:
		n = len(ie.networkingFields)
	case infraTabCICD:
		n = len(ie.cicdFields)
	case infraTabObservability:
		n = len(ie.obsFields)
	case infraTabEnvironments:
		n = len(ie.envForm)
	}
	for range n {
		var f *core.Field
		switch ie.activeTab {
		case infraTabNetworking:
			if ie.netFormIdx < len(ie.networkingFields) {
				f = &ie.networkingFields[ie.netFormIdx]
			}
		case infraTabCICD:
			if ie.cicdFormIdx < len(ie.cicdFields) {
				f = &ie.cicdFields[ie.cicdFormIdx]
			}
		case infraTabObservability:
			if ie.obsFormIdx < len(ie.obsFields) {
				f = &ie.obsFields[ie.obsFormIdx]
			}
		case infraTabEnvironments:
			if ie.envFormIdx < len(ie.envForm) {
				f = &ie.envForm[ie.envFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			ie.internalMode = core.ModeInsert
			ie.formInput.SetValue(f.TextInputValue())
			ie.formInput.Width = ie.width - 22
			ie.formInput.CursorEnd()
			return ie, ie.formInput.Focus()
		}
		ie.advanceField(1)
	}
	return ie, nil
}
