package infra

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// visibleNetworkingFields hides cors_origins unless cors_strategy is "Strict allowlist".
func (ie InfraEditor) visibleNetworkingFields() []core.Field {
	corsStrategy := core.FieldGet(ie.networkingFields, "cors_strategy")
	var out []core.Field
	for _, f := range ie.networkingFields {
		if f.Key == "cors_origins" && corsStrategy != "Strict allowlist" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// ── View ──────────────────────────────────────────────────────────────────────

func (ie InfraEditor) View(w, h int) string {
	ie.width = w
	ie.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		core.StyleSectionDesc.Render("  # Infrastructure — networking, CI/CD, and observability"),
		"",
		core.RenderSubTabBar(infraTabLabels, int(ie.activeTab), w),
		"",
	)

	const infraHeaderH = 4
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netEnabled {
			fl := core.RenderFormFields(w, ie.visibleNetworkingFields(), ie.netFormIdx, ie.internalMode == core.ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, ie.netFormIdx, h-infraHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabCICD:
		if ie.cicdEnabled {
			fl := core.RenderFormFields(w, ie.cicdFields, ie.cicdFormIdx, ie.internalMode == core.ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, ie.cicdFormIdx, h-infraHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabObservability:
		if ie.obsEnabled {
			fl := core.RenderFormFields(w, ie.obsFields, ie.obsFormIdx, ie.internalMode == core.ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, ie.obsFormIdx, h-infraHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case infraTabEnvironments:
		lines = append(lines, ie.viewEnvTab(w)...)
	}

	return core.FillTildes(lines, h)
}

// ── Environments list+form ────────────────────────────────────────────────────

func (ie InfraEditor) updateEnvTab(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	switch ie.envView {
	case envViewList:
		return ie.updateEnvList(key)
	case envViewForm:
		return ie.updateEnvForm(key)
	}
	return ie, nil
}

func (ie InfraEditor) updateEnvList(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	n := len(ie.envs)
	switch key.String() {
	case "j", "down":
		if n > 0 && ie.envIdx < n-1 {
			ie.envIdx++
		}
	case "k", "up":
		if ie.envIdx > 0 {
			ie.envIdx--
		}
	case "u":
		if snap, ok := ie.envsUndo.Pop(); ok {
			ie.envs = snap
			if ie.envIdx >= len(ie.envs) && ie.envIdx > 0 {
				ie.envIdx = len(ie.envs) - 1
			}
		}
	case "a":
		ie.envsUndo.Push(core.CopySlice(ie.envs))
		existing := make([]string, 0, len(ie.envs))
		for _, e := range ie.envs {
			existing = append(existing, e.Name)
		}
		newDef := manifest.ServerEnvironmentDef{
			Name:          core.UniqueName("environment", existing),
			ComputeEnv:    "Containers (Docker)",
			CloudProvider: "AWS",
			Orchestrator:  "Docker Compose",
		}
		ie.envs = append(ie.envs, newDef)
		ie.envIdx = len(ie.envs) - 1
		ie.envForm = serverEnvFormFromDef(ie.envs[ie.envIdx])
		ie.envFormIdx = 0
		ie.envView = envViewForm
		// apply compute_env narrowing to orchestrator options
		ie.applyEnvOrchestratorOptions()
		// propagate first env's cloud_provider to infra networking/cicd/obs
		ie.SetCloudProvider(ie.primaryCloudProvider())
	case "d":
		if n > 0 {
			ie.envsUndo.Push(core.CopySlice(ie.envs))
			ie.envs = append(ie.envs[:ie.envIdx], ie.envs[ie.envIdx+1:]...)
			if ie.envIdx > 0 && ie.envIdx >= len(ie.envs) {
				ie.envIdx = len(ie.envs) - 1
			}
		}
	case "enter", "l", "right":
		if n > 0 {
			ie.envForm = serverEnvFormFromDef(ie.envs[ie.envIdx])
			ie.envFormIdx = 0
			ie.envView = envViewForm
			ie.applyEnvOrchestratorOptions()
		}
	}
	return ie, nil
}

func (ie InfraEditor) updateEnvForm(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	if ie.dd.Open {
		return ie.updateEnvFormDropdown(key)
	}
	n := len(ie.envForm)
	switch key.String() {
	case "j", "down":
		if n > 0 {
			ie.envFormIdx = (ie.envFormIdx + 1) % n
		}
	case "k", "up":
		if n > 0 {
			ie.envFormIdx = (ie.envFormIdx - 1 + n) % n
		}
	case "enter", " ":
		if ie.envFormIdx < n {
			f := &ie.envForm[ie.envFormIdx]
			if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
				ie.dd.Open = true
				ie.dd.OptIdx = f.SelIdx
			} else {
				return ie.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if ie.envFormIdx < n {
			f := &ie.envForm[ie.envFormIdx]
			if f.Kind == core.KindSelect {
				f.CyclePrev()
				ie.onEnvFormFieldChanged(f.Key)
			}
		}
	case "i", "a":
		if ie.envFormIdx < n && ie.envForm[ie.envFormIdx].CanEditAsText() {
			return ie.tryEnterInsert()
		}
	case "b", "esc":
		ie.saveEnvForm()
		ie.envView = envViewList
	}
	ie.saveEnvForm()
	return ie, nil
}

func (ie InfraEditor) updateEnvFormDropdown(key tea.KeyMsg) (InfraEditor, tea.Cmd) {
	if ie.envFormIdx >= len(ie.envForm) {
		ie.dd.Open = false
		return ie, nil
	}
	f := &ie.envForm[ie.envFormIdx]
	ie.dd.OptIdx = core.NavigateDropdown(key.String(), ie.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(ie.dd.OptIdx)
			f.DDCursor = ie.dd.OptIdx
		} else {
			f.SelIdx = ie.dd.OptIdx
			if ie.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ie.dd.OptIdx]
			}
			ie.dd.Open = false
			ie.onEnvFormFieldChanged(f.Key)
			if f.PrepareCustomEntry() {
				return ie.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(ie.dd.OptIdx)
			f.DDCursor = ie.dd.OptIdx
		} else {
			f.SelIdx = ie.dd.OptIdx
			if ie.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[ie.dd.OptIdx]
			}
			ie.dd.Open = false
			ie.onEnvFormFieldChanged(f.Key)
			if f.PrepareCustomEntry() {
				return ie.tryEnterInsert()
			}
		}
	case "esc", "b":
		ie.dd.Open = false
	}
	ie.saveEnvForm()
	return ie, nil
}

// onEnvFormFieldChanged reacts to a field value change inside the env form.
func (ie *InfraEditor) onEnvFormFieldChanged(key string) {
	switch key {
	case "compute_env":
		ie.applyEnvOrchestratorOptions()
	case "cloud_provider":
		ie.SetCloudProvider(ie.primaryCloudProvider())
	}
}

// applyEnvOrchestratorOptions narrows orchestrator options in envForm
// based on the current compute_env selection.
func (ie *InfraEditor) applyEnvOrchestratorOptions() {
	computeEnv := core.FieldGet(ie.envForm, "compute_env")
	opts := narrowOrchestratorOptions(computeEnv)
	for i := range ie.envForm {
		if ie.envForm[i].Key != "orchestrator" {
			continue
		}
		ie.envForm[i].Options = opts
		found := false
		for j, o := range opts {
			if o == ie.envForm[i].Value {
				ie.envForm[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			ie.envForm[i].Value = opts[0]
			ie.envForm[i].SelIdx = 0
		}
		break
	}
}

// saveEnvForm writes the current envForm back to ie.envs[ie.envIdx].
func (ie *InfraEditor) saveEnvForm() {
	if ie.envIdx >= len(ie.envs) {
		return
	}
	ie.envs[ie.envIdx] = serverEnvDefFromForm(ie.envForm)
	// Keep networking/cicd/obs options in sync with primary env's settings.
	ie.SetCloudProvider(ie.primaryCloudProvider())
	ie.applyOrchestratorToCICD(ie.PrimaryOrchestrator())
	ie.applyOrchestratorToNetworking(ie.PrimaryOrchestrator())
}

func (ie InfraEditor) viewEnvTab(w int) []string {
	switch ie.envView {
	case envViewForm:
		return ie.viewEnvForm(w)
	default:
		return ie.viewEnvList(w)
	}
}

func (ie InfraEditor) viewEnvList(w int) []string {
	var lines []string
	lines = append(lines, core.StyleSectionDesc.Render("  server environments — define deployment targets (dev, staging, prod)"))
	lines = append(lines, "")
	if len(ie.envs) == 0 {
		lines = append(lines, core.StyleSectionDesc.Render("  (no environments — press 'a' to add one)"))
		return lines
	}
	for i, e := range ie.envs {
		cursor := "  "
		style := core.StyleFieldKey
		if i == ie.envIdx {
			cursor = core.StyleCursor.Render("▶ ")
			style = core.StyleFieldKeyActive
		}
		summary := e.Name
		if e.CloudProvider != "" {
			summary += "  " + core.StyleHelpDesc.Render(e.CloudProvider)
		}
		if e.ComputeEnv != "" {
			summary += "  " + core.StyleHelpDesc.Render(e.ComputeEnv)
		}
		if e.Orchestrator != "" {
			summary += "  " + core.StyleHelpDesc.Render(e.Orchestrator)
		}
		lines = append(lines, cursor+style.Render(summary))
	}
	return lines
}

func (ie InfraEditor) viewEnvForm(w int) []string {
	if ie.envIdx >= len(ie.envs) {
		return nil
	}
	header := core.StyleSectionDesc.Render("  editing environment: " + ie.envs[ie.envIdx].Name)
	lines := []string{header, ""}
	lines = append(lines, core.RenderFormFields(w, ie.envForm, ie.envFormIdx, ie.internalMode == core.ModeInsert, ie.formInput, ie.dd.Open, ie.dd.OptIdx)...)
	return lines
}

// CurrentField returns the currently highlighted form field for the description panel.
// Returns nil when in list view or when no field can be resolved.
func (ie *InfraEditor) CurrentField() *core.Field {
	switch ie.activeTab {
	case infraTabNetworking:
		if ie.netEnabled && ie.netFormIdx >= 0 && ie.netFormIdx < len(ie.networkingFields) {
			return &ie.networkingFields[ie.netFormIdx]
		}
	case infraTabCICD:
		if ie.cicdEnabled && ie.cicdFormIdx >= 0 && ie.cicdFormIdx < len(ie.cicdFields) {
			return &ie.cicdFields[ie.cicdFormIdx]
		}
	case infraTabObservability:
		if ie.obsEnabled && ie.obsFormIdx >= 0 && ie.obsFormIdx < len(ie.obsFields) {
			return &ie.obsFields[ie.obsFormIdx]
		}
	case infraTabEnvironments:
		if ie.envView == envViewForm && ie.envFormIdx >= 0 && ie.envFormIdx < len(ie.envForm) {
			return &ie.envForm[ie.envFormIdx]
		}
	}
	return nil
}
