package backend

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Security helpers ──────────────────────────────────────────────────────────

// isSecurityFieldHidden returns true when a security config field should be
// hidden given the currently selected architecture or prior security choices.
func (be BackendEditor) isSecurityFieldHidden(key string) bool {
	arch := be.currentArch()
	switch key {
	case "rate_limit_backend":
		// Hide backend selector when strategy is "None" or delegated to API Gateway.
		strategy := core.FieldGet(be.securityFields, "rate_limit_strategy")
		return strategy == "None" || strategy == "API Gateway"
	case "internal_mtls":
		// mTLS between services is irrelevant for a pure monolith (all in-process).
		return arch == string(manifest.ArchMonolith)
	}
	return false
}

// nextSecurityFieldIdx advances activeField by delta, skipping hidden fields.
func (be BackendEditor) nextSecurityFieldIdx(delta int) int {
	n := len(be.securityFields)
	if n == 0 {
		return 0
	}
	idx := be.activeField
	for i := 0; i < n; i++ {
		idx = (idx + delta + n) % n
		if !be.isSecurityFieldHidden(be.securityFields[idx].Key) {
			return idx
		}
	}
	return be.activeField
}

// refreshSecurityOptions recomputes field options to ensure architectural and
// cloud-provider compatibility. Call before cycling or opening any security dropdown.
func (be *BackendEditor) refreshSecurityOptions() {
	arch := be.currentArch()
	cloud := be.cloudProvider

	for i := range be.securityFields {
		f := &be.securityFields[i]
		switch f.Key {
		case "rate_limit_strategy":
			var opts []string
			switch arch {
			case string(manifest.ArchMicroservices), string(manifest.ArchEventDriven):
				// In-memory is invalid: state isn't shared across instances.
				opts = []string{"Token bucket (Redis)", "Sliding window", "Fixed window", "Leaky bucket", "API Gateway", "None"}
			case string(manifest.ArchMonolith):
				// Monoliths can safely use in-memory.
				opts = []string{"Token bucket (in-memory)", "Token bucket (Redis)", "Sliding window", "Fixed window", "Leaky bucket", "None"}
			default:
				opts = []string{"Token bucket (Redis)", "Sliding window", "Fixed window", "Leaky bucket", "API Gateway", "None"}
			}
			ensureSecuritySelection(f, opts)

		case "waf_provider":
			var opts []string
			switch cloud {
			case "AWS":
				opts = []string{"AWS WAF", "Cloudflare WAF", "ModSecurity", "NGINX ModSec", "None"}
			case "GCP":
				opts = []string{"Cloud Armor", "Cloudflare WAF", "ModSecurity", "NGINX ModSec", "None"}
			case "Azure":
				opts = []string{"Azure WAF", "Cloudflare WAF", "ModSecurity", "NGINX ModSec", "None"}
			default:
				opts = []string{"Cloudflare WAF", "AWS WAF", "Cloud Armor", "Azure WAF", "ModSecurity", "NGINX ModSec", "None"}
			}
			ensureSecuritySelection(f, opts)
		}
	}
}

// ensureSecuritySelection updates a field's options and reconciles SelIdx/Value.
// When the current value is no longer in the new option set, it resets to the
// first option (rather than "None") to avoid silently losing valid configuration.
func ensureSecuritySelection(f *core.Field, opts []string) {
	f.Options = opts
	for j, o := range opts {
		if o == f.Value {
			f.SelIdx = j
			return
		}
	}
	f.SelIdx = 0
	f.Value = opts[0]
}

// ── Security updates ──────────────────────────────────────────────────────────

func (be BackendEditor) updateSecurity(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	k := key.String()
	if !be.secEnabled {
		switch k {
		case "a":
			be.secEnabled = true
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
	n := len(be.securityFields)

	// Intercept j/k before core.VimNav — security fields use custom stepping
	// that skips hidden fields.
	switch k {
	case "j", "down":
		count := core.ParseVimCount(be.vim.CountBuf)
		be.vim.Reset()
		for i := 0; i < count; i++ {
			be.activeField = be.nextSecurityFieldIdx(+1)
		}
		return be, nil
	case "k", "up":
		count := core.ParseVimCount(be.vim.CountBuf)
		be.vim.Reset()
		for i := 0; i < count; i++ {
			be.activeField = be.nextSecurityFieldIdx(-1)
		}
		return be, nil
	}

	// Let core.VimNav handle digits, gg, G.
	if newIdx, consumed := be.vim.Handle(k, be.activeField, n); consumed {
		be.activeField = newIdx
		// G: adjust for trailing hidden fields.
		if k == "G" && n > 0 && be.isSecurityFieldHidden(be.securityFields[be.activeField].Key) {
			be.activeField = be.nextSecurityFieldIdx(-1)
		}
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
	case "enter", " ":
		if be.activeField < n {
			f := &be.securityFields[be.activeField]
			if f.Kind == core.KindSelect {
				be.refreshSecurityOptions()
				if f.Key == "rate_limit_backend" {
					opts := be.rateBackendOptions()
					ensureSecuritySelection(f, opts)
				}
				if len(f.Options) > 0 {
					be.dd.Open = true
					be.dd.OptIdx = f.SelIdx
				}
			}
		}
	case "H", "shift+left":
		if be.activeField < n {
			f := &be.securityFields[be.activeField]
			if f.Kind == core.KindSelect {
				be.refreshSecurityOptions()
				if f.Key == "rate_limit_backend" {
					opts := be.rateBackendOptions()
					ensureSecuritySelection(f, opts)
				}
				f.CyclePrev()
				// After cycling rate_limit_strategy, refresh dependent fields.
				if f.Key == "rate_limit_strategy" {
					be.refreshSecurityOptions()
				}
			}
		}
	case "D":
		be.secEnabled = false
		be.securityFields = defaultSecurityFields()
		be.activeField = 0
	}
	return be, nil
}

// CloudProvider returns the selected cloud provider from the Env tab.
// Returns an empty string if the env section has not been configured.
