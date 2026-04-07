package backend

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── modes ─────────────────────────────────────────────────────────────────────

// ── arch options ──────────────────────────────────────────────────────────────

type archOption struct {
	value string
	label string
	desc  string
}

var beArchOptions = []archOption{
	{"monolith", "Monolith", "Single deployable unit — all features in one codebase"},
	{"modular-monolith", "Modular Monolith", "Clear domain boundaries, single deployment"},
	{"microservices", "Microservices", "Independent services communicating over a network"},
	{"event-driven", "Event-Driven", "Services communicate asynchronously via events"},
	{"hybrid", "Hybrid", "Mix of patterns — each service unit tagged with its own pattern"},
}

// ── sub-tab IDs per arch ──────────────────────────────────────────────────────

// backendSubTab enumerates the logical sub-tabs in the backend section.
type backendSubTab int

const (
	beTabEnv backendSubTab = iota
	beTabServices
	beTabComm
	beTabMessaging
	beTabAPIGW
	beTabJobs
	beTabSecurity
	beTabAuth
)

// subTabsForArch returns the ordered list of sub-tabs for the given arch value.
func subTabsForArch(arch string) []backendSubTab {
	switch arch {
	case "monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabJobs, beTabSecurity, beTabAuth}
	case "modular-monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabJobs, beTabSecurity, beTabAuth}
	case "microservices":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabAPIGW, beTabJobs, beTabSecurity, beTabAuth}
	case "event-driven":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabJobs, beTabSecurity, beTabAuth}
	case "hybrid":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabAPIGW, beTabJobs, beTabSecurity, beTabAuth}
	default:
		return []backendSubTab{beTabEnv, beTabServices, beTabJobs, beTabSecurity, beTabAuth}
	}
}

func subTabLabel(t backendSubTab) string {
	switch t {
	case beTabEnv:
		return "CONFIG"
	case beTabServices:
		return "SERVICES"
	case beTabComm:
		return "COMM"
	case beTabMessaging:
		return "MESSAGING"
	case beTabAPIGW:
		return "API GW"
	case beTabJobs:
		return "JOBS"
	case beTabSecurity:
		return "SECURITY"
	case beTabAuth:
		return "AUTH"
	}
	return "?"
}

// ── beSubView / beAuthView types ──────────────────────────────────────────────

type beSubView int

const (
	beViewList beSubView = iota
	beViewForm
)

type beAuthView int

const (
	beAuthViewConfig   beAuthView = iota // flat config fields (strategy, provider, etc.)
	beAuthViewRoleList                   // roles list
	beAuthViewRoleForm                   // single role edit form
	beAuthViewPermList                   // permissions list
	beAuthViewPermForm                   // single permission edit form
)

// beRepoSubView tracks the drill-down state within a service's Data Access panel.
type beRepoSubView int

const (
	beRepoSubViewNone   beRepoSubView = iota // not in repo editing
	beRepoSubViewList                        // listing repos for the current service
	beRepoSubViewForm                        // editing a repo's basic fields
	beRepoSubViewOpList                      // listing ops for the current repo
	beRepoSubViewOpForm                      // editing an op
)

// ── list + form sub-editor for services and comm links ───────────────────────

type beListView int

const (
	beListViewList beListView = iota
	beListViewForm
)

type beListEditor struct {
	items    [][]core.Field // each item is a slice of fields
	itemView beListView
	itemIdx  int
	formIdx  int
	form     []core.Field
}

func newBeListEditor() beListEditor {
	return beListEditor{itemView: beListViewList}
}

// ── BackendEditor ─────────────────────────────────────────────────────────────

// BackendEditor manages the BACKEND section.
type BackendEditor struct {
	// Arch selection
	ArchIdx       int
	ArchConfirmed bool
	dropdownOpen  bool
	dropdownIdx   int

	// Sub-tab state
	activeTabIdx int // index into subTabsForArch(arch)
	activeField  int

	// core.Field stores
	EnvFields       []core.Field
	envEnabled      bool
	MessagingFields []core.Field
	APIGWFields     []core.Field
	apiGWEnabled    bool
	AuthFields      []core.Field
	authEnabled     bool

	// Security/WAF tab
	securityFields []core.Field
	secEnabled     bool

	// Jobs tab
	jobQueues   []manifest.JobQueueDef
	jobsSubView beSubView
	jobsIdx     int
	jobsForm    []core.Field
	jobsFormIdx int

	// Auth permissions + roles sub-editors
	authSubView     beAuthView
	authPerms       []manifest.PermissionDef
	authPermsIdx    int
	authPermForm    []core.Field
	authPermFormIdx int
	authRoles       []manifest.RoleDef
	authRolesIdx    int
	authRoleForm    []core.Field
	authRoleFormIdx int

	// List editors
	serviceEditor     beListEditor
	commEditor        beListEditor
	eventEditor       beListEditor // event catalog within messaging
	stackConfigEditor beListEditor // CONFIG tab stack configs (non-monolith)

	// Data Access (Repository) sub-editor — drill-down within a service form
	repoSubView beRepoSubView
	repoEditor  beListEditor
	opEditor    beListEditor

	// Stack configs for non-monolith arches (serialized to manifest).
	StackConfigs []manifest.StackConfig

	// Internal mode
	internalMode core.Mode
	formInput    textinput.Model
	width        int

	// Dropdown state (shared across all sub-contexts; only one can be open)
	dd core.DropdownState

	// Services and comm links for manifest export
	Services  []manifest.ServiceDef
	CommLinks []manifest.CommLink
	Events    []manifest.EventDef

	// Cross-tab references (injected from model.go)
	DomainNames        []string
	domainAttributes   map[string][]string // domain name → attribute names (from data tab)
	domainsByDB        map[string][]string // DB alias → domain names linked to that DB
	dbSourceTypes      map[string]string   // DB alias → DB type (from data tab)
	availableDTOs      []string
	availableEndpoints []string
	cacheAliases       []string                        // IsCache DB aliases from the Data pillar
	dbSourceAliases    []string                        // All DB source aliases from the Data pillar (for health_deps)
	dtoProtocols       []string                        // unique DTO serialisation protocols from ContractsEditor
	environmentNames   []string                        // InfraPillar environment names for service env dropdowns
	environmentDefs    []manifest.ServerEnvironmentDef // InfraPillar full env defs for API GW tech filtering
	orchestrator       string                          // Primary orchestrator from InfraPillar for service discovery
	cloudProvider      string                          // Primary cloud provider from InfraPillar for messaging deployment options

	// Vim motion state
	vim  core.VimNav
	cBuf bool

	// Per-subtab undo stacks (structural add/delete only)
	svcsUndo   core.UndoStack[core.SvcSnapshot]
	commsUndo  core.UndoStack[core.CommSnapshot]
	eventsUndo core.UndoStack[core.EventSnapshot]
	stacksUndo core.UndoStack[[][]core.Field]
	jobsUndo   core.UndoStack[[]manifest.JobQueueDef]
	rolesUndo  core.UndoStack[[]manifest.RoleDef]
	permsUndo  core.UndoStack[[]manifest.PermissionDef]
}

func NewEditor() BackendEditor {
	return BackendEditor{
		EnvFields:         defaultEnvFields(),
		MessagingFields:   defaultMessagingFields(),
		APIGWFields:       defaultAPIGWFields(),
		AuthFields:        defaultAuthFields(),
		securityFields:    defaultSecurityFields(),
		serviceEditor:     newBeListEditor(),
		commEditor:        newBeListEditor(),
		eventEditor:       newBeListEditor(),
		stackConfigEditor: newBeListEditor(),
		repoEditor:        newBeListEditor(),
		opEditor:          newBeListEditor(),
		formInput:         core.NewFormInput(),
		dropdownOpen:      true,
	}
}

// stackConfigNames returns the names of all defined stack configs.
func (be BackendEditor) stackConfigNames() []string {
	var names []string
	for _, item := range be.stackConfigEditor.items {
		if n := core.FieldGet(item, "name"); n != "" {
			names = append(names, n)
		}
	}
	return names
}

// langForConfig returns the language of the stack config with the given name,
// or "" if the name is empty, "(any)", or not found.
func (be BackendEditor) langForConfig(configName string) string {
	if configName == "" || configName == "(any)" {
		return ""
	}
	for _, item := range be.stackConfigEditor.items {
		if core.FieldGet(item, "name") == configName {
			return core.FieldGet(item, "language")
		}
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (be BackendEditor) currentArch() string {
	if be.ArchIdx >= 0 && be.ArchIdx < len(beArchOptions) {
		return beArchOptions[be.ArchIdx].value
	}
	return beArchOptions[0].value
}

func (be BackendEditor) activeTabs() []backendSubTab {
	return subTabsForArch(be.currentArch())
}

func (be BackendEditor) activeTab() backendSubTab {
	tabs := be.activeTabs()
	if be.activeTabIdx >= 0 && be.activeTabIdx < len(tabs) {
		return tabs[be.activeTabIdx]
	}
	return beTabEnv
}

func (be BackendEditor) tabLabels() []string {
	tabs := be.activeTabs()
	labels := make([]string, len(tabs))
	for i, t := range tabs {
		labels[i] = subTabLabel(t)
	}
	return labels
}

// ── core.Mode / HintLine ───────────────────────────────────────────────────────────

func (be BackendEditor) Mode() core.Mode {
	if be.internalMode == core.ModeInsert {
		return core.ModeInsert
	}
	return core.ModeNormal
}

// ── Update ────────────────────────────────────────────────────────────────────

func (be BackendEditor) Update(msg tea.Msg) (BackendEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		be.width = wsz.Width
		be.formInput.Width = wsz.Width - 22
		return be, nil
	}
	if !be.ArchConfirmed {
		return be.updateArchSelect(msg)
	}
	if be.internalMode == core.ModeInsert {
		return be.updateInsert(msg)
	}
	if be.dd.Open {
		key, ok := msg.(tea.KeyMsg)
		if ok {
			return be.updateDropdown(key)
		}
		return be, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "c" {
			if be.cBuf {
				be.cBuf = false
				return be.clearAndEnterInsert()
			}
			be.cBuf = true
			return be, nil
		}
		be.cBuf = false
	}
	return be.updateNormal(msg)
}

func (be BackendEditor) updateArchSelect(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}
	if be.dropdownOpen {
		switch key.String() {
		case "j", "down":
			if be.dropdownIdx < len(beArchOptions)-1 {
				be.dropdownIdx++
			}
		case "k", "up":
			if be.dropdownIdx > 0 {
				be.dropdownIdx--
			}
		case "g":
			be.dropdownIdx = 0
		case "G":
			be.dropdownIdx = len(beArchOptions) - 1
		case "enter", " ":
			oldArch := beArchOptions[be.ArchIdx].value
			be.ArchIdx = be.dropdownIdx
			be.dropdownOpen = false
			be.ArchConfirmed = true
			be.activeTabIdx = 0
			be.activeField = 0
			newArch := beArchOptions[be.ArchIdx].value
			if oldArch != newArch {
				be = be.resetForArchChange(newArch)
			}
		case "esc":
			be.dropdownOpen = false
		}
		return be, nil
	}
	switch key.String() {
	case "enter", " ":
		be.dropdownOpen = true
		be.dropdownIdx = be.ArchIdx
	}
	return be, nil
}

// resetForArchChange clears architecture-specific state that is not valid for
// the new architecture pattern. This ensures stale data from a previous arch
// (e.g. stack configs from microservices) does not leak into the manifest.
func (be BackendEditor) resetForArchChange(newArch string) BackendEditor {
	newTabs := subTabsForArch(newArch)
	hasTab := func(t backendSubTab) bool {
		for _, tab := range newTabs {
			if tab == t {
				return true
			}
		}
		return false
	}

	// Monolith uses pillar-level lang/fw; clear stack configs.
	// Non-monolith uses stack configs; reset monolith-level env fields.
	if newArch == "monolith" {
		be.stackConfigEditor = newBeListEditor()
		be.StackConfigs = nil
	} else {
		be.EnvFields = defaultEnvFields()
		be.envEnabled = false
	}

	// Clear comm links if COMM tab is no longer available.
	if !hasTab(beTabComm) {
		be.commEditor = newBeListEditor()
		be.CommLinks = nil
	}

	// Clear messaging + events if MESSAGING tab is no longer available.
	if !hasTab(beTabMessaging) {
		be.MessagingFields = defaultMessagingFields()
		be.eventEditor = newBeListEditor()
		be.Events = nil
	}

	// Clear API gateway if API GW tab is no longer available.
	if !hasTab(beTabAPIGW) {
		be.APIGWFields = defaultAPIGWFields()
		be.apiGWEnabled = false
	}

	// Clear service fields that are architecture-specific.
	for i := range be.Services {
		if newArch == "monolith" {
			be.Services[i].ConfigRef = ""
			be.Services[i].ServiceDiscovery = ""
			be.Services[i].Environment = ""
		}
		if newArch != "hybrid" {
			be.Services[i].PatternTag = ""
		}
	}
	for i := range be.serviceEditor.items {
		if newArch == "monolith" {
			be.serviceEditor.items[i] = core.SetFieldValue(be.serviceEditor.items[i], "config_ref", "")
			be.serviceEditor.items[i] = core.SetFieldValue(be.serviceEditor.items[i], "service_discovery", "")
			be.serviceEditor.items[i] = core.SetFieldValue(be.serviceEditor.items[i], "environment", "")
		}
		if newArch != "hybrid" {
			be.serviceEditor.items[i] = core.SetFieldValue(be.serviceEditor.items[i], "pattern_tag", "")
		}
	}

	// Monolith doesn't need inter-service mTLS.
	if newArch == "monolith" {
		be.securityFields = core.SetFieldValue(be.securityFields, "internal_mtls", "Disabled")
	}

	return be
}
