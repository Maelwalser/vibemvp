package backend

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// isAuthFieldHidden returns true when an auth config field should be hidden
// given the currently selected strategy and provider.
func (be BackendEditor) isAuthFieldHidden(key string) bool {
	switch key {
	case "session_mgmt":
		// Hide when no session-bearing strategy is active.
		for _, f := range be.AuthFields {
			if f.Key != "strategy" {
				continue
			}
			for _, idx := range f.SelectedIdxs {
				if idx >= 0 && idx < len(f.Options) && f.Options[idx] == "Session-based" {
					return false
				}
			}
			return true
		}
		return false
	case "token_storage":
		// Hide when no token-bearing strategy is selected (API Key, mTLS, None only).
		tokenBearing := map[string]bool{
			"JWT (stateless)":  true,
			"Session-based":    true,
			"OAuth 2.0 / OIDC": true,
		}
		for _, f := range be.AuthFields {
			if f.Key != "strategy" {
				continue
			}
			for _, idx := range f.SelectedIdxs {
				if idx >= 0 && idx < len(f.Options) && tokenBearing[f.Options[idx]] {
					return false
				}
			}
			return true
		}
		return false
	}
	return false
}

// updateAuthTokenStorageOptions recomputes token_storage options as the union
// of options applicable to the currently selected auth strategies.
func (be *BackendEditor) updateAuthTokenStorageOptions() {
	// Per-strategy option sets (in canonical display order).
	canShowCookie := false
	canShowBearer := false
	canShowOther := false

	for _, f := range be.AuthFields {
		if f.Key != "strategy" {
			continue
		}
		for _, idx := range f.SelectedIdxs {
			if idx < 0 || idx >= len(f.Options) {
				continue
			}
			switch f.Options[idx] {
			case "JWT (stateless)":
				canShowCookie = true
				canShowBearer = true
				canShowOther = true
			case "Session-based":
				canShowCookie = true
			case "OAuth 2.0 / OIDC":
				canShowCookie = true
				canShowBearer = true
			}
		}
		break
	}

	var opts []string
	if canShowCookie {
		opts = append(opts, "HttpOnly cookie")
	}
	if canShowBearer {
		opts = append(opts, "Authorization header (Bearer)")
	}
	if canShowOther {
		opts = append(opts, "Other")
	}
	if len(opts) == 0 {
		// No token-bearing strategy — field is hidden anyway; keep a safe default.
		opts = []string{"HttpOnly cookie", "Authorization header (Bearer)", "Other"}
	}

	for i := range be.AuthFields {
		if be.AuthFields[i].Key != "token_storage" {
			continue
		}
		// Preserve currently-selected values that still exist in new option set.
		optSet := make(map[string]int, len(opts))
		for j, o := range opts {
			optSet[o] = j
		}
		var kept []int
		for _, sel := range be.AuthFields[i].SelectedIdxs {
			if sel >= 0 && sel < len(be.AuthFields[i].Options) {
				if j, ok := optSet[be.AuthFields[i].Options[sel]]; ok {
					kept = append(kept, j)
				}
			}
		}
		be.AuthFields[i].Options = opts
		be.AuthFields[i].SelectedIdxs = kept
		break
	}
}

// mfaOptionsForProvider returns the MFA options appropriate for a given auth provider.
func mfaOptionsForProvider(provider string) []string {
	switch provider {
	case "Self-managed":
		return []string{"None", "TOTP", "Email"}
	case "Auth0", "Clerk", "Firebase Auth":
		return []string{"None", "TOTP", "SMS", "Email", "Passkeys/WebAuthn"}
	case "Keycloak":
		return []string{"None", "TOTP", "WebAuthn"}
	case "Supabase Auth":
		return []string{"None", "TOTP", "Phone (Twilio)"}
	case "AWS Cognito":
		return []string{"None", "TOTP", "SMS", "Email"}
	default:
		return []string{"None", "TOTP", "SMS", "Email", "Passkeys/WebAuthn"}
	}
}

// updateAuthMFAOptions recomputes the mfa field options based on the selected provider.
func (be *BackendEditor) updateAuthMFAOptions() {
	provider := core.FieldGet(be.AuthFields, "provider")
	opts := mfaOptionsForProvider(provider)
	cur := core.FieldGet(be.AuthFields, "mfa")
	for i := range be.AuthFields {
		if be.AuthFields[i].Key != "mfa" {
			continue
		}
		be.AuthFields[i].Options = opts
		// Keep current value when still valid; otherwise reset to "None".
		valid := false
		for j, o := range opts {
			if o == cur {
				be.AuthFields[i].SelIdx = j
				valid = true
				break
			}
		}
		if !valid {
			be.AuthFields[i].SelIdx = 0
			be.AuthFields[i].Value = opts[0]
		}
		break
	}
}

// nextAuthFieldIdx advances activeField by delta, skipping hidden auth fields.
func (be BackendEditor) nextAuthFieldIdx(delta int) int {
	n := len(be.AuthFields)
	if n == 0 {
		return 0
	}
	idx := be.activeField
	for i := 0; i < n; i++ {
		idx = (idx + delta + n) % n
		if !be.isAuthFieldHidden(be.AuthFields[idx].Key) {
			return idx
		}
	}
	return be.activeField
}

// ── Auth updates ──────────────────────────────────────────────────────────────

func (be BackendEditor) updateAuth(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	if !be.authEnabled {
		switch k {
		case "a":
			be.authEnabled = true
			be.activeField = 0
		case "h", "left":
			if be.activeTabIdx > 0 {
				be.activeTabIdx--
			}
		case "l", "right":
			if be.activeTabIdx < len(be.activeTabs())-1 {
				be.activeTabIdx++
			}
		case "b":
			be.ArchConfirmed = false
			be.dropdownOpen = false
			be.dropdownIdx = be.ArchIdx
			be.activeTabIdx = 0
			be.activeField = 0
		}
		return be, nil
	}
	switch be.authSubView {
	case beAuthViewConfig:
		return be.updateAuthConfig(key)
	case beAuthViewRoleList:
		return be.updateAuthRoleList(key)
	case beAuthViewRoleForm:
		return be.updateAuthRoleForm(key)
	case beAuthViewPermList:
		return be.updateAuthPermList(key)
	case beAuthViewPermForm:
		return be.updateAuthPermForm(key)
	}
	return be, nil
}

func (be BackendEditor) updateAuthConfig(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	n := len(be.AuthFields)

	// Intercept j/k before core.VimNav — auth fields use custom stepping
	// that skips hidden fields.
	switch k {
	case "j", "down":
		count := core.ParseVimCount(be.vim.CountBuf)
		be.vim.Reset()
		for i := 0; i < count; i++ {
			be.activeField = be.nextAuthFieldIdx(+1)
		}
		return be, nil
	case "k", "up":
		count := core.ParseVimCount(be.vim.CountBuf)
		be.vim.Reset()
		for i := 0; i < count; i++ {
			be.activeField = be.nextAuthFieldIdx(-1)
		}
		return be, nil
	}

	// Let core.VimNav handle digits, gg, G.
	if newIdx, consumed := be.vim.Handle(k, be.activeField, n); consumed {
		be.activeField = newIdx
		return be, nil
	}
	be.vim.Reset()

	switch k {
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	case "b":
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "r":
		be.authSubView = beAuthViewRoleList
		be.activeField = 0
	case "p":
		be.authSubView = beAuthViewPermList
		be.activeField = 0
	case "D":
		be.authEnabled = false
		be.AuthFields = defaultAuthFields()
		be.authPerms = nil
		be.authPermsIdx = 0
		be.authRoles = nil
		be.authRolesIdx = 0
		be.authSubView = beAuthViewConfig
		be.activeField = 0
	case "enter", " ":
		if be.activeField < n {
			f := &be.AuthFields[be.activeField]
			if f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect {
				if f.Key == "service_unit" {
					be.refreshAuthServiceUnitOptions(f)
				}
				if len(f.Options) > 0 {
					be.dd.Open = true
					if f.Kind == core.KindSelect {
						be.dd.OptIdx = f.SelIdx
					} else {
						be.dd.OptIdx = f.DDCursor
					}
				}
			} else {
				return be.tryEnterInsert()
			}
		}
	case "H", "shift+left":
		if be.activeField < n {
			f := &be.AuthFields[be.activeField]
			if f.Kind == core.KindSelect {
				if f.Key == "service_unit" {
					be.refreshAuthServiceUnitOptions(f)
				}
				f.CyclePrev()
				if f.Key == "provider" {
					be.updateAuthMFAOptions()
				}
			}
		}
	case "i", "a":
		return be.tryEnterInsert()
	}
	return be, nil
}

func (be BackendEditor) updateAuthRoleList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authRoles)
	switch key.String() {
	case "j", "down":
		if n > 0 && be.authRolesIdx < n-1 {
			be.authRolesIdx++
		}
	case "k", "up":
		if be.authRolesIdx > 0 {
			be.authRolesIdx--
		}
	case "u":
		if snap, ok := be.rolesUndo.Pop(); ok {
			be.authRoles = snap
			if be.authRolesIdx >= len(be.authRoles) && be.authRolesIdx > 0 {
				be.authRolesIdx = len(be.authRoles) - 1
			}
		}
	case "a":
		be.rolesUndo.Push(core.CopySlice(be.authRoles))
		be.authRoles = append(be.authRoles, manifest.RoleDef{})
		be.authRolesIdx = len(be.authRoles) - 1
		be.authRoleForm = defaultRoleFormFields(be.permissionNames(), be.roleNamesExcept(be.authRolesIdx))
		existing := make([]string, 0, len(be.authRoles)-1)
		for i, r := range be.authRoles {
			if i != be.authRolesIdx {
				existing = append(existing, r.Name)
			}
		}
		be.authRoleForm = core.SetFieldValue(be.authRoleForm, "name", core.UniqueName("role", existing))
		be.authRoleFormIdx = 0
		be.authSubView = beAuthViewRoleForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.rolesUndo.Push(core.CopySlice(be.authRoles))
			be.authRoles = append(be.authRoles[:be.authRolesIdx], be.authRoles[be.authRolesIdx+1:]...)
			if be.authRolesIdx > 0 && be.authRolesIdx >= len(be.authRoles) {
				be.authRolesIdx = len(be.authRoles) - 1
			}
		}
	case "enter", "i":
		if n > 0 {
			r := be.authRoles[be.authRolesIdx]
			be.authRoleForm = defaultRoleFormFields(be.permissionNames(), be.roleNamesExcept(be.authRolesIdx))
			be.authRoleForm = core.SetFieldValue(be.authRoleForm, "name", r.Name)
			be.authRoleForm = core.SetFieldValue(be.authRoleForm, "description", r.Description)
			be.authRoleForm = core.RestoreMultiSelectValue(be.authRoleForm, "permissions", strings.Join(r.Permissions, ", "))
			be.authRoleForm = core.RestoreMultiSelectValue(be.authRoleForm, "inherits", strings.Join(r.Inherits, ", "))
			be.authRoleFormIdx = 0
			be.authSubView = beAuthViewRoleForm
			be.activeField = 0
		}
	case "b", "esc":
		be.authSubView = beAuthViewConfig
		be.activeField = 0
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	}
	return be, nil
}

func (be BackendEditor) updateAuthRoleForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authRoleForm)
	switch key.String() {
	case "j", "down":
		if be.authRoleFormIdx < n-1 {
			be.authRoleFormIdx++
		}
		be.activeField = be.authRoleFormIdx
	case "k", "up":
		if be.authRoleFormIdx > 0 {
			be.authRoleFormIdx--
		}
		be.activeField = be.authRoleFormIdx
	case "enter", " ":
		if be.authRoleFormIdx < n {
			f := &be.authRoleForm[be.authRoleFormIdx]
			if f.Kind == core.KindMultiSelect && len(f.Options) > 0 {
				be.dd.Open = true
				be.dd.OptIdx = f.DDCursor
			} else if f.Kind == core.KindText {
				return be.enterAuthRoleFormInsert()
			}
		}
	case "i", "a":
		if be.authRoleFormIdx < n && be.authRoleForm[be.authRoleFormIdx].Kind == core.KindText {
			return be.enterAuthRoleFormInsert()
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "b", "esc":
		be.saveAuthRoleForm()
		be.authSubView = beAuthViewRoleList
	}
	be.saveAuthRoleForm()
	return be, nil
}

func (be BackendEditor) enterAuthRoleFormInsert() (BackendEditor, tea.Cmd) {
	n := len(be.authRoleForm)
	for i := 0; i < n; i++ {
		f := be.authRoleForm[be.authRoleFormIdx]
		if f.Kind == core.KindText {
			be.internalMode = core.ModeInsert
			be.formInput.SetValue(f.Value)
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
		be.authRoleFormIdx = (be.authRoleFormIdx + 1) % n
		be.activeField = be.authRoleFormIdx
	}
	return be, nil
}

func (be *BackendEditor) saveAuthRoleForm() {
	if be.authRolesIdx >= len(be.authRoles) {
		return
	}
	r := &be.authRoles[be.authRolesIdx]
	r.Name = core.FieldGet(be.authRoleForm, "name")
	r.Description = core.FieldGet(be.authRoleForm, "description")
	r.Permissions = core.SplitCSV(core.FieldGetMulti(be.authRoleForm, "permissions"))
	r.Inherits = core.SplitCSV(core.FieldGetMulti(be.authRoleForm, "inherits"))
}

func (be BackendEditor) updateAuthPermList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authPerms)
	switch key.String() {
	case "j", "down":
		if n > 0 && be.authPermsIdx < n-1 {
			be.authPermsIdx++
		}
	case "k", "up":
		if be.authPermsIdx > 0 {
			be.authPermsIdx--
		}
	case "u":
		if snap, ok := be.permsUndo.Pop(); ok {
			be.authPerms = snap
			if be.authPermsIdx >= len(be.authPerms) && be.authPermsIdx > 0 {
				be.authPermsIdx = len(be.authPerms) - 1
			}
		}
	case "a":
		be.permsUndo.Push(core.CopySlice(be.authPerms))
		be.authPerms = append(be.authPerms, manifest.PermissionDef{})
		be.authPermsIdx = len(be.authPerms) - 1
		be.authPermForm = defaultPermFormFields()
		existing := make([]string, 0, len(be.authPerms)-1)
		for i, p := range be.authPerms {
			if i != be.authPermsIdx {
				existing = append(existing, p.Name)
			}
		}
		be.authPermForm = core.SetFieldValue(be.authPermForm, "name", core.UniqueName("permission", existing))
		be.authPermFormIdx = 0
		be.authSubView = beAuthViewPermForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.permsUndo.Push(core.CopySlice(be.authPerms))
			be.authPerms = append(be.authPerms[:be.authPermsIdx], be.authPerms[be.authPermsIdx+1:]...)
			if be.authPermsIdx > 0 && be.authPermsIdx >= len(be.authPerms) {
				be.authPermsIdx = len(be.authPerms) - 1
			}
		}
	case "enter", "i":
		if n > 0 {
			p := be.authPerms[be.authPermsIdx]
			be.authPermForm = defaultPermFormFields()
			be.authPermForm = core.SetFieldValue(be.authPermForm, "name", p.Name)
			be.authPermForm = core.SetFieldValue(be.authPermForm, "description", p.Description)
			be.authPermFormIdx = 0
			be.authSubView = beAuthViewPermForm
			be.activeField = 0
		}
	case "b", "esc":
		be.authSubView = beAuthViewConfig
		be.activeField = 0
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
		}
	}
	return be, nil
}

func (be BackendEditor) updateAuthPermForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	n := len(be.authPermForm)
	switch key.String() {
	case "j", "down":
		if be.authPermFormIdx < n-1 {
			be.authPermFormIdx++
		}
		be.activeField = be.authPermFormIdx
	case "k", "up":
		if be.authPermFormIdx > 0 {
			be.authPermFormIdx--
		}
		be.activeField = be.authPermFormIdx
	case "enter", "i", "a":
		if be.authPermFormIdx < n && be.authPermForm[be.authPermFormIdx].Kind == core.KindText {
			return be.enterAuthPermFormInsert()
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
		}
	case "l", "right":
		tabs := be.activeTabs()
		if be.activeTabIdx < len(tabs)-1 {
			be.activeTabIdx++
		}
	case "b", "esc":
		be.saveAuthPermForm()
		be.authSubView = beAuthViewPermList
	}
	be.saveAuthPermForm()
	return be, nil
}

func (be BackendEditor) enterAuthPermFormInsert() (BackendEditor, tea.Cmd) {
	if be.authPermFormIdx < len(be.authPermForm) {
		f := be.authPermForm[be.authPermFormIdx]
		if f.Kind == core.KindText {
			be.internalMode = core.ModeInsert
			be.formInput.SetValue(f.Value)
			be.formInput.Width = be.width - 22
			be.formInput.CursorEnd()
			return be, be.formInput.Focus()
		}
	}
	return be, nil
}

func (be *BackendEditor) saveAuthPermForm() {
	if be.authPermsIdx >= len(be.authPerms) {
		return
	}
	p := &be.authPerms[be.authPermsIdx]
	p.Name = core.FieldGet(be.authPermForm, "name")
	p.Description = core.FieldGet(be.authPermForm, "description")
}

// permissionNames returns names of all defined permissions.
func (be BackendEditor) permissionNames() []string {
	names := make([]string, 0, len(be.authPerms))
	for _, p := range be.authPerms {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return names
}

// authProviderNeedsService returns true for providers that are self-hosted and
// require a backend service unit to handle authentication.
func authProviderNeedsService(provider string) bool {
	switch provider {
	case "Self-managed", "Keycloak":
		return true
	}
	return false
}

// refreshAuthServiceUnitOptions updates the service_unit field options based on
// the currently selected provider. External providers get a single "None (external)"
// option; self-managed providers get the list of configured service names.
func (be *BackendEditor) refreshAuthServiceUnitOptions(f *core.Field) {
	provider := core.FieldGet(be.AuthFields, "provider")
	var opts []string
	if authProviderNeedsService(provider) {
		svcNames := be.ServiceNames()
		if len(svcNames) == 0 {
			opts = []string{"(no services configured)"}
		} else {
			opts = append([]string{"None"}, svcNames...)
		}
	} else {
		opts = []string{"None (external)"}
	}
	f.Options = opts
	// Keep current value if still present; otherwise reset to first option.
	found := false
	for j, o := range opts {
		if o == f.Value {
			f.SelIdx = j
			found = true
			break
		}
	}
	if !found {
		f.SelIdx = 0
		f.Value = opts[0]
	}
}

// roleNamesExcept returns names of all roles except the one at excludeIdx.
func (be BackendEditor) roleNamesExcept(excludeIdx int) []string {
	names := make([]string, 0, len(be.authRoles))
	for i, r := range be.authRoles {
		if i != excludeIdx && r.Name != "" {
			names = append(names, r.Name)
		}
	}
	return names
}

// viewAuth renders the AUTH tab content.
func (be BackendEditor) viewAuth(w int) []string {
	if !be.authEnabled {
		return []string{core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)")}
	}
	switch be.authSubView {
	case beAuthViewConfig:
		var visibleAuthFields []core.Field
		skippedBefore := 0
		for i, f := range be.AuthFields {
			if be.isAuthFieldHidden(f.Key) {
				if i < be.activeField {
					skippedBefore++
				}
				continue
			}
			visibleAuthFields = append(visibleAuthFields, f)
		}
		filteredActiveIdx := be.activeField - skippedBefore
		if filteredActiveIdx < 0 {
			filteredActiveIdx = 0
		}
		lines := core.RenderFormFields(w, visibleAuthFields, filteredActiveIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)
		permCount := fmt.Sprintf("%d", len(be.authPerms))
		roleCount := fmt.Sprintf("%d", len(be.authRoles))
		lines = append(lines,
			"",
			core.StyleSectionDesc.Render("  # Permissions ("+permCount+" defined) — press 'p' to manage"),
			core.StyleSectionDesc.Render("  # Roles ("+roleCount+" defined) — press 'r' to manage"),
		)
		return lines
	case beAuthViewPermList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Permissions — a: add  d: delete  Enter: edit  b: back"), "")
		if len(be.authPerms) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no permissions yet — press 'a' to add)"))
		} else {
			for i, p := range be.authPerms {
				name := p.Name
				if name == "" {
					name = fmt.Sprintf("(perm #%d)", i+1)
				}
				lines = append(lines, core.RenderListItem(w, i == be.authPermsIdx, "  ▶ ", name, p.Description))
			}
		}
		return lines
	case beAuthViewPermForm:
		name := core.FieldGet(be.authPermForm, "name")
		if name == "" {
			name = "(new permission)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		lines = append(lines, core.RenderFormFields(w, be.authPermForm, be.authPermFormIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		return lines
	case beAuthViewRoleList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Roles — a: add  d: delete  Enter: edit  b: back"), "")
		if len(be.authRoles) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no roles yet — press 'a' to add)"))
		} else {
			for i, r := range be.authRoles {
				name := r.Name
				if name == "" {
					name = fmt.Sprintf("(role #%d)", i+1)
				}
				detail := ""
				if len(r.Permissions) > 0 {
					detail = strings.Join(r.Permissions[:min(3, len(r.Permissions))], ", ")
					if len(r.Permissions) > 3 {
						detail += "…"
					}
				}
				lines = append(lines, core.RenderListItem(w, i == be.authRolesIdx, "  ▶ ", name, detail))
			}
		}
		return lines
	case beAuthViewRoleForm:
		name := core.FieldGet(be.authRoleForm, "name")
		if name == "" {
			name = "(new role)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		lines = append(lines, core.RenderFormFields(w, be.authRoleForm, be.authRoleFormIdx, be.internalMode == core.ModeInsert, be.formInput, be.dd.Open, be.dd.OptIdx)...)
		return lines
	}
	return nil
}
