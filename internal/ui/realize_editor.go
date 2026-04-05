package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// RealizeMsg is emitted when the user presses R to start realization.
type RealizeMsg struct{}

// ── Provider → tier model data ────────────────────────────────────────────────

// providerOrder is the canonical display order for providers.
var providerOrder = []string{"Claude", "ChatGPT", "Gemini", "Mistral", "Llama", "Custom"}

// providerTierModels maps provider label → [fast, medium, slow] model ID lists.
// Index 0 = TierFast, 1 = TierMedium, 2 = TierSlow.
var providerTierModels = map[string][3][]string{
	"Claude": {
		{"claude-haiku-4-5-20251001"},
		{"claude-sonnet-4-6"},
		{"claude-opus-4-6"},
	},
	"ChatGPT": {
		{"gpt-4o-mini", "o3-mini"},
		{"gpt-4o", "gpt-4o-2024-11-20"},
		{"o1", "o1-preview"},
	},
	"Gemini": {
		{"gemini-2.0-flash", "gemini-1.5-flash"},
		{"gemini-2.0-pro-exp", "gemini-1.5-pro"},
		{"gemini-ultra"},
	},
	"Mistral": {
		{"open-mistral-nemo"},
		{"mistral-small-2409", "mistral-small-2402"},
		{"mistral-large-2411", "mistral-large-2407"},
	},
	"Llama": {
		{"llama-3.2-8b-preview", "llama-3.1-8b-instant"},
		{"llama-3.3-70b-versatile", "llama-3.1-70b-versatile"},
		{"llama-3.1-405b-reasoning"},
	},
}

// ── Field helpers ─────────────────────────────────────────────────────────────

const unset = "—"

func defaultRealizeFields() []Field {
	return []Field{
		{
			Key: "app_name", Label: "app_name      ", Kind: KindText,
			Value: "my-app",
		},
		{
			Key: "output_dir", Label: "output_dir    ", Kind: KindText,
			Value: ".",
		},
		{
			Key: "concurrency", Label: "concurrency   ", Kind: KindSelect,
			Options: []string{"1", "2", "4", "8"},
			Value:   "4", SelIdx: 2,
		},
		{
			Key: "verify", Label: "verify        ", Kind: KindSelect,
			Options: OptionsOnOff,
			Value:   "true",
		},
		{
			Key: "dry_run", Label: "dry_run       ", Kind: KindSelect,
			Options: OptionsOffOn,
			Value:   "false",
		},
		// Provider selector — options are rebuilt from configured providers.
		{
			Key: "provider", Label: "provider      ", Kind: KindSelect,
			Options: []string{unset},
			Value:   unset,
		},
		// Tier model selectors — options are rebuilt when provider changes.
		{
			Key: "tier_fast", Label: "tier_fast     ", Kind: KindSelect,
			Options: []string{unset}, Value: unset,
		},
		{
			Key: "tier_medium", Label: "tier_medium   ", Kind: KindSelect,
			Options: []string{unset}, Value: unset,
		},
		{
			Key: "tier_slow", Label: "tier_slow     ", Kind: KindSelect,
			Options: []string{unset}, Value: unset,
		},
	}
}

// ── RealizeEditor ─────────────────────────────────────────────────────────────

// RealizeEditor manages the REALIZE main-tab.
type RealizeEditor struct {
	fields    []Field
	activeIdx int
	mode      Mode
	formInput textinput.Model
	width     int
	dd        DropdownState
	cBuf      bool
}

func newRealizeEditor() RealizeEditor {
	return RealizeEditor{
		fields:    defaultRealizeFields(),
		formInput: newFormInput(),
	}
}

// ── Provider / tier options sync ──────────────────────────────────────────────

// UpdateProviderOptions rebuilds the provider selector options from the
// currently configured providers and syncs the tier fields accordingly.
func (r RealizeEditor) UpdateProviderOptions(configured map[string]ProviderSelection) RealizeEditor {
	// Build the list of configured providers — no unset sentinel in the dropdown.
	options := []string{}
	for _, label := range providerOrder {
		sel, ok := configured[label]
		if !ok || !sel.IsSet() {
			continue
		}
		options = append(options, label)
	}

	for i := range r.fields {
		if r.fields[i].Key != "provider" {
			continue
		}
		r.fields[i].Options = options
		// Keep current value if still in the list; otherwise reset to unset.
		found := false
		for j, o := range options {
			if o == r.fields[i].Value {
				r.fields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found {
			r.fields[i].Value = unset
			r.fields[i].SelIdx = 0
		}
		break
	}
	return r.syncTierFields()
}

// syncTierFields updates the tier_fast / tier_medium / tier_slow option lists
// to match the currently selected provider. When no provider is selected (unset),
// all tier fields are reset to the unset sentinel.
func (r RealizeEditor) syncTierFields() RealizeEditor {
	provider := fieldGet(r.fields, "provider")
	tiers, hasProvider := providerTierModels[provider]

	tierKeys := []string{"tier_fast", "tier_medium", "tier_slow"}
	for ti, key := range tierKeys {
		var options []string
		if hasProvider {
			options = tiers[ti]
		}
		// No unset sentinel in options — — is only the placeholder Value.

		for i := range r.fields {
			if r.fields[i].Key != key {
				continue
			}
			r.fields[i].Options = options
			found := false
			for j, o := range options {
				if o == r.fields[i].Value {
					r.fields[i].SelIdx = j
					found = true
					break
				}
			}
			if !found {
				r.fields[i].Value = unset
				r.fields[i].SelIdx = 0
			}
			break
		}
	}
	return r
}

// ── ToManifest / FromRealizeOptions ──────────────────────────────────────────

func (r RealizeEditor) ToManifestRealizeOptions() manifest.RealizeOptions {
	concurrency := 4
	switch fieldGet(r.fields, "concurrency") {
	case "1":
		concurrency = 1
	case "2":
		concurrency = 2
	case "4":
		concurrency = 4
	case "8":
		concurrency = 8
	}

	// Treat the unset sentinel as an empty string so the orchestrator falls back
	// to its default (Claude via ANTHROPIC_API_KEY env var).
	emptyIfUnset := func(v string) string {
		if v == unset {
			return ""
		}
		return v
	}

	return manifest.RealizeOptions{
		AppName:     fieldGet(r.fields, "app_name"),
		OutputDir:   fieldGet(r.fields, "output_dir"),
		Concurrency: concurrency,
		Verify:      fieldGet(r.fields, "verify") == "true",
		DryRun:      fieldGet(r.fields, "dry_run") == "true",
		Provider:    emptyIfUnset(fieldGet(r.fields, "provider")),
		TierFast:    emptyIfUnset(fieldGet(r.fields, "tier_fast")),
		TierMedium:  emptyIfUnset(fieldGet(r.fields, "tier_medium")),
		TierSlow:    emptyIfUnset(fieldGet(r.fields, "tier_slow")),
	}
}

func (r RealizeEditor) FromRealizeOptions(ro manifest.RealizeOptions) RealizeEditor {
	r.fields = setFieldValue(r.fields, "app_name", ro.AppName)
	r.fields = setFieldValue(r.fields, "output_dir", ro.OutputDir)
	switch ro.Concurrency {
	case 1, 2, 4, 8:
		r.fields = setFieldValue(r.fields, "concurrency", fmt.Sprintf("%d", ro.Concurrency))
	}
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	r.fields = setFieldValue(r.fields, "verify", boolStr(ro.Verify))
	r.fields = setFieldValue(r.fields, "dry_run", boolStr(ro.DryRun))

	if ro.Provider != "" {
		r.fields = setFieldValue(r.fields, "provider", ro.Provider)
		r = r.syncTierFields()
	}
	if ro.TierFast != "" {
		r.fields = setFieldValue(r.fields, "tier_fast", ro.TierFast)
	}
	if ro.TierMedium != "" {
		r.fields = setFieldValue(r.fields, "tier_medium", ro.TierMedium)
	}
	if ro.TierSlow != "" {
		r.fields = setFieldValue(r.fields, "tier_slow", ro.TierSlow)
	}
	return r
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (r RealizeEditor) Mode() Mode {
	if r.mode == ModeInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (r RealizeEditor) HintLine() string {
	if r.mode == ModeInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	return hintBar("j/k", "navigate", "Space/Enter", "cycle", "i", "edit text", "R", "start realization")
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r RealizeEditor) Update(msg tea.Msg) (RealizeEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		r.width = wsz.Width
		r.formInput.Width = wsz.Width - 22
		return r, nil
	}
	if r.mode == ModeInsert {
		return r.updateInsert(msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}
	if r.dd.Open {
		return r.updateDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if r.activeIdx < len(r.fields)-1 {
			r.activeIdx++
		}
	case "k", "up":
		if r.activeIdx > 0 {
			r.activeIdx--
		}
	case "g":
		r.activeIdx = 0
	case "G":
		r.activeIdx = len(r.fields) - 1
	case "enter", " ":
		if r.activeIdx < len(r.fields) {
			f := &r.fields[r.activeIdx]
			if f.Kind == KindSelect && len(f.Options) > 0 {
				r.dd.Open = true
				r.dd.OptIdx = f.SelIdx
				if r.dd.OptIdx >= len(f.Options) {
					r.dd.OptIdx = 0
				}
			} else if f.Kind != KindSelect {
				return r.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if r.activeIdx < len(r.fields) {
			f := &r.fields[r.activeIdx]
			if f.Kind == KindSelect && len(f.Options) > 0 {
				f.CyclePrev()
				if f.Key == "provider" {
					r = r.syncTierFields()
				}
			}
		}
	case "i":
		return r.tryEnterInsert()
	case "c":
		if r.cBuf {
			r.cBuf = false
			return r.clearAndEnterInsert()
		}
		r.cBuf = true
		return r, nil
	case "R":
		return r, func() tea.Msg { return RealizeMsg{} }
	default:
		r.cBuf = false
	}
	return r, nil
}

func (r RealizeEditor) updateDropdown(key tea.KeyMsg) (RealizeEditor, tea.Cmd) {
	if r.activeIdx >= len(r.fields) {
		r.dd.Open = false
		return r, nil
	}
	f := &r.fields[r.activeIdx]
	r.dd.OptIdx = NavigateDropdown(key.String(), r.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = r.dd.OptIdx
		if r.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[r.dd.OptIdx]
		}
		r.dd.Open = false
		// When provider changes, rebuild tier options.
		if f.Key == "provider" {
			r = r.syncTierFields()
		}
		if f.PrepareCustomEntry() {
			return r.tryEnterInsert()
		}
	case "esc", "b":
		r.dd.Open = false
	}
	return r, nil
}

func (r RealizeEditor) updateInsert(msg tea.Msg) (RealizeEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			r.saveInput()
			r.mode = ModeNormal
			r.formInput.Blur()
			return r, nil
		case "tab":
			r.saveInput()
			if next := nextEditableIdx(r.fields, r.activeIdx); next >= 0 {
				r.activeIdx = next
			}
			return r.tryEnterInsert()
		case "shift+tab":
			r.saveInput()
			if prev := prevEditableIdx(r.fields, r.activeIdx); prev >= 0 {
				r.activeIdx = prev
			}
			return r.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	r.formInput, cmd = r.formInput.Update(msg)
	return r, cmd
}

func (r *RealizeEditor) saveInput() {
	if r.activeIdx < len(r.fields) && r.fields[r.activeIdx].CanEditAsText() {
		r.fields[r.activeIdx].SaveTextInput(r.formInput.Value())
	}
}

func (r RealizeEditor) clearAndEnterInsert() (RealizeEditor, tea.Cmd) {
	r, cmd := r.tryEnterInsert()
	if r.mode == ModeInsert {
		r.formInput.SetValue("")
	}
	return r, cmd
}

func (r RealizeEditor) tryEnterInsert() (RealizeEditor, tea.Cmd) {
	if r.activeIdx < len(r.fields) && r.fields[r.activeIdx].CanEditAsText() {
		r.mode = ModeInsert
		r.formInput.SetValue(r.fields[r.activeIdx].TextInputValue())
		r.formInput.Width = r.width - 22
		r.formInput.CursorEnd()
		return r, r.formInput.Focus()
	}
	return r, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

var (
	styleRealizeBtn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrBg)).
			Background(lipgloss.Color(clrGreen)).
			Bold(true).
			Padding(0, 2)

	styleRealizeBtnHint = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrComment))
)

// splitIdx is the index at which the field list splits into the two display groups.
const splitIdx = 5

func (r RealizeEditor) View(w, h int) string {
	r.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Realization — configure output directory, app name, and agent options"),
		"",
	)

	appFields := r.fields[:splitIdx]
	tierFields := r.fields[splitIdx:]

	var content []string
	activeLine := r.activeIdx

	content = append(content, renderFormFields(w, appFields, r.activeIdx, r.mode == ModeInsert, r.formInput, r.dd.Open, r.dd.OptIdx)...)
	content = append(content, "")
	content = append(content, StyleSectionDesc.Render("  # Provider — select a configured provider and assign a model to each complexity tier"))
	content = append(content, "")

	if r.activeIdx >= splitIdx {
		activeLine = len(appFields) + 1 + 2 + (r.activeIdx - splitIdx)
	}

	content = append(content, renderFormFields(w, tierFields, r.activeIdx-splitIdx, r.mode == ModeInsert, r.formInput, r.dd.Open, r.dd.OptIdx)...)
	content = append(content, "")

	btn := styleRealizeBtn.Render(" R  Start Realization ")
	hint := styleRealizeBtnHint.Render("  saves manifest then launches the realize agent")
	btnLine := "  " + btn + hint
	pad := w - lipgloss.Width(btnLine)
	if pad > 0 {
		btnLine += strings.Repeat(" ", pad)
	}
	content = append(content, btnLine)

	const realizeHeaderH = 2
	lines = append(lines, appendViewport(content, 0, activeLine, h-realizeHeaderH)...)

	return fillTildes(lines, h)
}

// CurrentField returns the currently highlighted form field for the description panel.
func (r *RealizeEditor) CurrentField() *Field {
	if r.activeIdx >= 0 && r.activeIdx < len(r.fields) {
		return &r.fields[r.activeIdx]
	}
	return nil
}
