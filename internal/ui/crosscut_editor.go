package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-mvp/internal/manifest"
)

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type ccTabIdx int

const (
	ccTabTesting ccTabIdx = iota
	ccTabDocs
)

var ccTabLabels = []string{"TESTING", "DOCS"}

// ── mode ──────────────────────────────────────────────────────────────────────

type ccMode int

const (
	ccNormal ccMode = iota
	ccInsert
)

// ── field definitions ─────────────────────────────────────────────────────────

func defaultTestingFields() []Field {
	return []Field{
		{
			Key: "unit", Label: "unit          ", Kind: KindSelect,
			Options: []string{
				"Jest", "Vitest", "pytest", "Go testing",
				"JUnit", "xUnit", "Other",
			},
			Value: "Go testing", SelIdx: 3,
		},
		{
			Key: "integration", Label: "integration   ", Kind: KindSelect,
			Options: []string{
				"Testcontainers", "Docker Compose", "In-memory fakes", "None",
			},
			Value: "Testcontainers",
		},
		{
			Key: "e2e", Label: "e2e           ", Kind: KindSelect,
			Options: []string{"Playwright", "Cypress", "Selenium", "None"},
			Value:   "Playwright",
		},
		{
			Key: "api", Label: "api           ", Kind: KindSelect,
			Options: []string{"Bruno", "Hurl", "Postman/Newman", "REST Client", "None"},
			Value:   "Hurl", SelIdx: 1,
		},
		{
			Key: "load", Label: "load          ", Kind: KindSelect,
			Options: []string{"k6", "Locust", "Artillery", "JMeter", "None"},
			Value:   "k6",
		},
		{
			Key: "contract", Label: "contract      ", Kind: KindSelect,
			Options: []string{"Pact", "Schemathesis", "Dredd", "None"},
			Value:   "None", SelIdx: 3,
		},
	}
}

func defaultDocsFields() []Field {
	return []Field{
		{
			Key: "api_docs", Label: "api_docs      ", Kind: KindSelect,
			Options: []string{
				"OpenAPI/Swagger", "GraphQL Playground",
				"gRPC reflection", "None",
			},
			Value: "OpenAPI/Swagger",
		},
		{
			Key: "auto_generate", Label: "auto_generate ", Kind: KindSelect,
			Options: []string{"false", "true"}, Value: "true", SelIdx: 1,
		},
		{
			Key: "changelog", Label: "changelog     ", Kind: KindSelect,
			Options: []string{"Conventional Commits", "Manual", "None"},
			Value:   "Conventional Commits",
		},
	}
}

// ── CrossCutEditor ────────────────────────────────────────────────────────────

// CrossCutEditor manages the CROSS-CUTTING CONCERNS main-tab.
type CrossCutEditor struct {
	activeTab ccTabIdx

	testingFields []Field
	testFormIdx   int

	docsFields  []Field
	docsFormIdx int

	internalMode ccMode
	formInput    textinput.Model
	width        int
}

func newCrossCutEditor() CrossCutEditor {
	return CrossCutEditor{
		testingFields: defaultTestingFields(),
		docsFields:    defaultDocsFields(),
		formInput:     newFormInput(),
	}
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (cc CrossCutEditor) ToManifestCrossCutPillar() manifest.CrossCutPillar {
	return manifest.CrossCutPillar{
		Testing: manifest.TestingConfig{
			Unit:        fieldGet(cc.testingFields, "unit"),
			Integration: fieldGet(cc.testingFields, "integration"),
			E2E:         fieldGet(cc.testingFields, "e2e"),
			API:         fieldGet(cc.testingFields, "api"),
			Load:        fieldGet(cc.testingFields, "load"),
			Contract:    fieldGet(cc.testingFields, "contract"),
		},
		Docs: manifest.DocsConfig{
			APIDocs:      fieldGet(cc.docsFields, "api_docs"),
			AutoGenerate: fieldGet(cc.docsFields, "auto_generate") == "true",
			Changelog:    fieldGet(cc.docsFields, "changelog"),
		},
	}
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (cc CrossCutEditor) Mode() Mode {
	if cc.internalMode == ccInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (cc CrossCutEditor) HintLine() string {
	if cc.internalMode == ccInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	return hintBar("j/k", "navigate", "Space/Enter", "cycle", "H", "cycle back", "i", "edit text", "h/l", "sub-tab")
}

// ── Update ────────────────────────────────────────────────────────────────────

func (cc CrossCutEditor) Update(msg tea.Msg) (CrossCutEditor, tea.Cmd) {
	if cc.internalMode == ccInsert {
		return cc.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return cc, nil
	}

	switch key.String() {
	case "h", "left":
		if cc.activeTab > 0 {
			cc.activeTab--
		}
		return cc, nil
	case "l", "right":
		if int(cc.activeTab) < len(ccTabLabels)-1 {
			cc.activeTab++
		}
		return cc, nil
	}

	switch cc.activeTab {
	case ccTabTesting:
		return cc.updateFields(&cc.testingFields, &cc.testFormIdx, key)
	case ccTabDocs:
		return cc.updateFields(&cc.docsFields, &cc.docsFormIdx, key)
	}
	return cc, nil
}

func (cc CrossCutEditor) updateFields(fields *[]Field, idx *int, key tea.KeyMsg) (CrossCutEditor, tea.Cmd) {
	n := len(*fields)
	switch key.String() {
	case "j", "down":
		if *idx < n-1 {
			*idx++
		}
	case "k", "up":
		if *idx > 0 {
			*idx--
		}
	case "enter", " ":
		f := &(*fields)[*idx]
		if f.Kind == KindSelect {
			f.CycleNext()
		} else {
			return cc.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &(*fields)[*idx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		return cc.tryEnterInsert()
	}
	return cc, nil
}

func (cc CrossCutEditor) updateInsert(msg tea.Msg) (CrossCutEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			cc.saveInput()
			cc.internalMode = ccNormal
			cc.formInput.Blur()
			return cc, nil
		case "tab":
			cc.saveInput()
			cc.advanceField(1)
			return cc.tryEnterInsert()
		case "shift+tab":
			cc.saveInput()
			cc.advanceField(-1)
			return cc.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	cc.formInput, cmd = cc.formInput.Update(msg)
	return cc, cmd
}

func (cc *CrossCutEditor) advanceField(delta int) {
	switch cc.activeTab {
	case ccTabTesting:
		n := len(cc.testingFields)
		if n > 0 {
			cc.testFormIdx = (cc.testFormIdx + delta + n) % n
		}
	case ccTabDocs:
		n := len(cc.docsFields)
		if n > 0 {
			cc.docsFormIdx = (cc.docsFormIdx + delta + n) % n
		}
	}
}

func (cc *CrossCutEditor) saveInput() {
	val := cc.formInput.Value()
	switch cc.activeTab {
	case ccTabTesting:
		if cc.testFormIdx < len(cc.testingFields) && cc.testingFields[cc.testFormIdx].Kind == KindText {
			cc.testingFields[cc.testFormIdx].Value = val
		}
	case ccTabDocs:
		if cc.docsFormIdx < len(cc.docsFields) && cc.docsFields[cc.docsFormIdx].Kind == KindText {
			cc.docsFields[cc.docsFormIdx].Value = val
		}
	}
}

func (cc CrossCutEditor) tryEnterInsert() (CrossCutEditor, tea.Cmd) {
	var f *Field
	switch cc.activeTab {
	case ccTabTesting:
		if cc.testFormIdx < len(cc.testingFields) {
			f = &cc.testingFields[cc.testFormIdx]
		}
	case ccTabDocs:
		if cc.docsFormIdx < len(cc.docsFields) {
			f = &cc.docsFields[cc.docsFormIdx]
		}
	}
	if f == nil || f.Kind != KindText {
		return cc, nil
	}
	cc.internalMode = ccInsert
	cc.formInput.SetValue(f.Value)
	cc.formInput.Width = cc.width - 22
	cc.formInput.CursorEnd()
	return cc, cc.formInput.Focus()
}

// ── View ──────────────────────────────────────────────────────────────────────

func (cc CrossCutEditor) View(w, h int) string {
	cc.width = w
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Cross-Cutting Concerns — testing strategy and documentation"),
		"",
		renderSubTabBar(ccTabLabels, int(cc.activeTab)),
		"",
	)

	switch cc.activeTab {
	case ccTabTesting:
		lines = append(lines, renderFormFields(w, cc.testingFields, cc.testFormIdx, cc.internalMode == ccInsert, cc.formInput)...)
	case ccTabDocs:
		lines = append(lines, renderFormFields(w, cc.docsFields, cc.docsFormIdx, cc.internalMode == ccInsert, cc.formInput)...)
	}

	return fillTildes(lines, h)
}
