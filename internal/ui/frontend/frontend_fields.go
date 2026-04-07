package frontend

import (
	"strings"

	"github.com/vibe-menu/internal/ui/core"
)

// ── Meta-tag injection options per framework ─────────────────────────────────

var metaTagByFramework = map[string][]string{
	"React":   {"Manual", "react-helmet", "Framework-native", "None"},
	"Vue":     {"Manual", "@vueuse/head", "Framework-native", "None"},
	"Svelte":  {"Manual", "svelte:head", "Framework-native", "None"},
	"Angular": {"Manual", "Framework-native", "None"},
	"Solid":   {"Manual", "@solidjs/meta", "Framework-native", "None"},
	"Preact":  {"Manual", "react-helmet", "Framework-native", "None"},
	"Lit":     {"Manual", "Framework-native", "None"},
}

// refreshMetaTagOptions rebuilds the Options (and clamps SelIdx/Value) for the
// meta_tag_injection field inside the supplied a11y field slice.
func refreshMetaTagOptions(fields []core.Field, framework string) []core.Field {
	opts, ok := metaTagByFramework[framework]
	if !ok {
		opts = []string{"Manual", "Framework-native", "None"}
	}
	updated := make([]core.Field, len(fields))
	copy(updated, fields)
	for i, f := range updated {
		if f.Key != "meta_tag_injection" {
			continue
		}
		f.Options = opts
		found := false
		for j, o := range opts {
			if o == f.Value {
				f.SelIdx = j
				found = true
				break
			}
		}
		if !found {
			f.SelIdx = len(opts) - 1
			f.Value = opts[len(opts)-1]
		}
		updated[i] = f
		break
	}
	return updated
}

// ── SEO render strategy options per meta-framework ───────────────────────────

var seoRenderByMetaFramework = map[string][]string{
	"Next.js":   {"SSR", "SSG", "ISR", "Prerender", "None"},
	"Nuxt":      {"SSR", "SSG", "ISR", "None"},
	"SvelteKit": {"SSR", "SSG", "Prerender", "None"},
	"Remix":     {"SSR", "None"},
	"Astro":     {"SSG", "SSR", "None"},
	"None":      {"Prerender", "None"},
}

// seoRenderOptions returns the valid seo_render_strategy options given the
// current platform and meta-framework selections.
// Mobile and Desktop platforms do not support server rendering strategies.
func seoRenderOptions(platform, metaFramework string) []string {
	lower := strings.ToLower(platform)
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "desktop") {
		return []string{"None"}
	}
	if opts, ok := seoRenderByMetaFramework[metaFramework]; ok {
		return opts
	}
	// Web platform with unrecognised meta-framework: omit ISR (Next.js-only)
	return []string{"SSR", "SSG", "Prerender", "None"}
}

// metaFrameworkSupportsSSRSSG returns true when the given meta-framework can
// perform server-side or static-site rendering, making "Instant (SSR/SSG)" a
// valid page loading strategy.
func metaFrameworkSupportsSSRSSG(metaFramework string) bool {
	switch metaFramework {
	case "Next.js", "Nuxt", "SvelteKit", "Remix", "Astro":
		return true
	}
	return false
}

// loadingOptions returns the valid page loading strategy options given the
// current meta-framework selection.
func loadingOptions(metaFramework string) []string {
	if metaFrameworkSupportsSSRSSG(metaFramework) {
		return []string{"Skeleton", "Spinner", "Progressive", "Instant (SSR/SSG)"}
	}
	return []string{"Skeleton", "Spinner", "Progressive"}
}

// refreshLoadingOptions rebuilds the Options (and clamps SelIdx/Value) for the
// loading field inside the supplied page form field slice.
func refreshLoadingOptions(fields []core.Field, metaFramework string) []core.Field {
	opts := loadingOptions(metaFramework)
	updated := make([]core.Field, len(fields))
	copy(updated, fields)
	for i, f := range updated {
		if f.Key != "loading" {
			continue
		}
		f.Options = opts
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
		updated[i] = f
		break
	}
	return updated
}

// refreshSEORenderOptions rebuilds the Options (and clamps SelIdx/Value) for the
// seo_render_strategy field inside the supplied a11y field slice.
func refreshSEORenderOptions(fields []core.Field, platform, metaFramework string) []core.Field {
	opts := seoRenderOptions(platform, metaFramework)
	updated := make([]core.Field, len(fields))
	copy(updated, fields)
	for i, f := range updated {
		if f.Key != "seo_render_strategy" {
			continue
		}
		f.Options = opts
		// Keep current value if still valid; otherwise fall back to last option.
		found := false
		for j, o := range opts {
			if o == f.Value {
				f.SelIdx = j
				found = true
				break
			}
		}
		if !found {
			f.SelIdx = len(opts) - 1
			f.Value = opts[len(opts)-1]
		}
		updated[i] = f
		break
	}
	return updated
}

// ── framework options per language/platform ───────────────────────────────────

var frontendFrameworksByLang = map[string][]string{
	"TypeScript": {"React", "Vue", "Svelte", "Angular", "Solid", "Qwik", "HTMX"},
	"JavaScript": {"React", "Vue", "Svelte", "Angular", "Solid", "Qwik", "HTMX"},
	"Dart":       {"Flutter"},
	"Kotlin":     {"Jetpack Compose", "KMP (Compose Multiplatform)"},
	"Swift":      {"SwiftUI", "UIKit"},
}

// ── field definitions ─────────────────────────────────────────────────────────

func defaultFETechFields() []core.Field {
	return []core.Field{
		{
			Key: "language", Label: "language      ", Kind: core.KindSelect,
			Options: []string{"TypeScript", "JavaScript", "Dart", "Kotlin", "Swift"},
			Value:   "TypeScript",
		},
		{
			Key: "language_version", Label: "lang version  ", Kind: core.KindSelect,
			Options: core.LangVersions["TypeScript"],
			Value:   core.LangVersions["TypeScript"][0],
		},
		{
			Key: "platform", Label: "platform      ", Kind: core.KindSelect,
			Options: []string{
				"Web (SPA)", "Web (SSR/SSG)", "Mobile (cross-platform)",
				"Mobile (native)", "Desktop",
			},
			Value: "Web (SPA)",
		},
		{
			Key: "framework", Label: "framework     ", Kind: core.KindSelect,
			Options: frontendFrameworksByLang["TypeScript"],
			Value:   "React",
		},
		{
			Key: "framework_version", Label: "fw version    ", Kind: core.KindSelect,
			Options: core.CompatibleFrameworkVersions("TypeScript", core.LangVersions["TypeScript"][0], "React"),
			Value:   core.CompatibleFrameworkVersions("TypeScript", core.LangVersions["TypeScript"][0], "React")[0],
		},
		{
			Key: "meta_framework", Label: "meta_framework", Kind: core.KindSelect,
			Options: frontendMetaframeworksByFramework["React"],
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "pkg_manager", Label: "pkg_manager   ", Kind: core.KindSelect,
			Options: []string{"npm", "yarn", "pnpm", "bun"},
			Value:   "pnpm", SelIdx: 2,
		},
		{
			Key: "styling", Label: "styling       ", Kind: core.KindSelect,
			Options: []string{
				"Tailwind CSS", "CSS Modules", "Styled Components",
				"Sass/SCSS", "Vanilla CSS", "UnoCSS",
			},
			Value: "Tailwind CSS",
		},
		{
			Key: "component_lib", Label: "component_lib ", Kind: core.KindSelect,
			Options: []string{
				"shadcn/ui", "Radix", "Material UI", "Ant Design",
				"Headless UI", "DaisyUI", "None", "Custom",
			},
			Value: "shadcn/ui",
		},
		{
			Key: "state_mgmt", Label: "state_mgmt    ", Kind: core.KindSelect,
			Options: []string{
				"React Context", "Zustand", "Redux Toolkit", "Jotai",
				"Pinia", "Svelte stores", "Signals", "None",
			},
			Value: "Zustand", SelIdx: 1,
		},
		{
			Key: "data_fetching", Label: "data_fetching ", Kind: core.KindSelect,
			Options: []string{
				"TanStack Query", "SWR", "Apollo Client",
				"tRPC client", "gRPC-web client", "Connect client",
				"RTK Query", "Native fetch",
			},
			Value: "TanStack Query",
		},
		{
			Key: "form_handling", Label: "form_handling ", Kind: core.KindSelect,
			Options: []string{"React Hook Form", "Formik", "Zod + native", "Vee-Validate", "None"},
			Value:   "React Hook Form",
		},
		{
			Key: "validation", Label: "validation    ", Kind: core.KindSelect,
			Options: []string{"Zod", "Yup", "Valibot", "Joi", "Class-validator", "None"},
			Value:   "Zod",
		},
		{
			Key: "pwa_support", Label: "PWA Support   ", Kind: core.KindSelect,
			Options: []string{"None", "Basic (manifest + service worker)", "Full offline", "Push notifications"},
			Value:   "None",
		},
		{
			Key: "realtime", Label: "Real-time     ", Kind: core.KindSelect,
			Options: []string{"WebSocket", "SSE", "Polling", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "image_opt", Label: "Image Optim.  ", Kind: core.KindSelect,
			Options: []string{"Next/Image (built-in)", "Cloudinary", "Imgix", "Sharp (self-hosted)", "CDN transform", "None"},
			Value:   "None", SelIdx: 5,
		},
		{
			Key: "auth_flow", Label: "Auth Flow     ", Kind: core.KindSelect,
			Options: []string{"Redirect (OAuth/OIDC)", "Modal login", "Magic link", "Passwordless", "Social only"},
			Value:   "Redirect (OAuth/OIDC)",
		},
		{
			Key: "error_boundary", Label: "Error Boundary", Kind: core.KindSelect,
			Options: []string{"React Error Boundary", "Global try-catch", "Framework default", "Custom"},
			Value:   "Framework default", SelIdx: 2,
		},
		{
			Key: "bundle_opt", Label: "Bundle Optim. ", Kind: core.KindSelect,
			Options: []string{"Code splitting (route-based)", "Dynamic imports", "Tree shaking only", "None"},
			Value:   "None", SelIdx: 3,
		},
	}
}

func defaultFEThemeFields() []core.Field {
	return []core.Field{
		{
			Key: "dark_mode", Label: "dark_mode     ", Kind: core.KindSelect,
			Options: []string{"None", "Toggle (user preference)", "System preference", "Dark only"},
			Value:   "System preference", SelIdx: 2,
		},
		{
			Key: "border_radius", Label: "border_radius ", Kind: core.KindSelect,
			Options: []string{"Sharp (0)", "Subtle (4px)", "Rounded (8px)", "Pill (999px)", "Custom"},
			Value:   "Rounded (8px)", SelIdx: 2,
		},
		{
			Key: "spacing", Label: "spacing       ", Kind: core.KindSelect,
			Options: []string{"Compact (4px base)", "Default (8px base)", "Spacious (12px base)"},
			Value:   "Default (8px base)", SelIdx: 1,
		},
		{
			Key: "elevation", Label: "elevation     ", Kind: core.KindSelect,
			Options: []string{"Shadows", "Borders", "Both", "Flat"},
			Value:   "Shadows",
		},
		{
			Key: "motion", Label: "motion        ", Kind: core.KindSelect,
			Options: []string{"None", "Subtle transitions", "Animated (spring/ease)"},
			Value:   "Subtle transitions", SelIdx: 1,
		},
		{
			Key: "vibe", Label: "vibe          ", Kind: core.KindSelect,
			Options: []string{
				"Professional", "Playful", "Minimal", "Bold",
				"Elegant", "Technical", "Creative", "Friendly", "Serious", "Modern",
				"Custom",
			},
			Value: "Professional",
		},
		{
			Key: "font", Label: "font          ", Kind: core.KindSelect,
			Options: []string{
				"Inter", "Roboto", "Open Sans", "Lato", "Poppins", "Nunito",
				"Source Sans Pro", "Raleway", "Montserrat", "Playfair Display",
				"Merriweather", "Fira Code", "JetBrains Mono", "System default", "Custom",
			},
			Value: "Inter",
		},
		{
			Key: "colors", Label: "colors        ", Kind: core.KindMultiSelect,
			ColorSwatch: true,
			Options:     themeColorPalette,
		},
		{Key: "description", Label: "description   ", Kind: core.KindTextArea},
	}
}

func defaultPageFormFields(metaFramework string, authRoleOptions, linkedPageOptions, assetNameOptions, componentNameOptions []string) []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "route", Label: "route         ", Kind: core.KindText},
		{
			Key: "purpose", Label: "purpose       ", Kind: core.KindSelect,
			Options: []string{
				"Landing/Marketing", "Dashboard/Overview", "List/Index",
				"Detail/View", "Create/Form", "Edit/Form",
				"Auth/Login", "Settings/Profile", "Error/404", "Admin", "Other",
			},
			Value: "Other", SelIdx: 10,
		},
		{
			Key: "auth_required", Label: "auth_required ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "layout", Label: "layout        ", Kind: core.KindSelect,
			Options: []string{"Default", "Sidebar", "Full-width", "Blank", "Custom (specify)"},
			Value:   "Default",
		},
		{Key: "description", Label: "description   ", Kind: core.KindText},
		{Key: "core_actions", Label: "core_actions  ", Kind: core.KindText},
		{
			Key: "loading", Label: "loading       ", Kind: core.KindSelect,
			Options: loadingOptions(metaFramework),
			Value:   "Skeleton",
		},
		{
			Key: "error_handling", Label: "error_handling", Kind: core.KindSelect,
			Options: []string{"Inline", "Toast", "Error boundary / fallback page", "Retry"},
			Value:   "Toast", SelIdx: 1,
		},
		{
			Key: "auth_roles", Label: "auth_roles    ", Kind: core.KindMultiSelect,
			Options: authRoleOptions,
			Value:   core.PlaceholderFor(authRoleOptions, "(no auth roles configured)"),
		},
		{
			Key: "linked_pages", Label: "linked_pages  ", Kind: core.KindMultiSelect,
			Options: linkedPageOptions,
			Value:   core.PlaceholderFor(linkedPageOptions, "(no other pages)"),
		},
		{
			Key: "assets", Label: "assets        ", Kind: core.KindMultiSelect,
			Options: assetNameOptions,
			Value:   core.PlaceholderFor(assetNameOptions, "(no assets configured)"),
		},
		{
			Key: "component_refs", Label: "components    ", Kind: core.KindMultiSelect,
			Options: componentNameOptions,
			Value:   core.PlaceholderFor(componentNameOptions, "(no components defined)"),
		},
	}
}

func defaultComponentFormFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key: "comp_type", Label: "comp_type     ", Kind: core.KindSelect,
			Options: []string{"Form", "Table", "Card", "List", "Chart", "Modal", "Button", "Navigation", "Custom"},
			Value:   "Form",
		},
		{Key: "description", Label: "description   ", Kind: core.KindText},
	}
}

func defaultI18nFields() []core.Field {
	return []core.Field{
		{
			Key: "enabled", Label: "enabled       ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "default_locale", Label: "default_locale", Kind: core.KindSelect,
			Options: []string{
				"en", "en-US", "en-GB", "en-AU", "en-CA",
				"fr", "fr-FR", "fr-CA", "de", "de-DE", "de-AT",
				"es", "es-ES", "es-MX", "es-AR", "pt", "pt-BR", "pt-PT",
				"it", "nl", "nl-NL", "pl", "ru", "ja", "zh", "zh-CN", "zh-TW",
				"ko", "ar", "hi", "tr", "sv", "da", "fi", "nb", "cs", "hu",
				"ro", "vi", "th", "id", "ms", "uk", "he",
			},
			Value: "en",
		},
		{
			Key: "supported_locales", Label: "locales       ", Kind: core.KindMultiSelect,
			Options: []string{
				"en", "en-US", "en-GB", "en-AU", "en-CA",
				"fr", "fr-FR", "fr-CA", "de", "de-DE", "de-AT",
				"es", "es-ES", "es-MX", "es-AR", "pt", "pt-BR", "pt-PT",
				"it", "nl", "nl-NL", "pl", "ru", "ja", "zh", "zh-CN", "zh-TW",
				"ko", "ar", "hi", "tr", "sv", "da", "fi", "nb", "cs", "hu",
				"ro", "vi", "th", "id", "ms", "uk", "he",
			},
		},
		{
			Key: "translation_strategy", Label: "i18n_library  ", Kind: core.KindSelect,
			Options: []string{"i18next", "next-intl", "react-i18next", "LinguiJS", "vue-i18n", "Custom", "None"},
			Value:   "None", SelIdx: 6,
		},
		{
			Key: "timezone_handling", Label: "timezone      ", Kind: core.KindSelect,
			Options: []string{"UTC always", "User preference", "Auto-detect", "Manual"},
			Value:   "UTC always",
		},
	}
}

func defaultA11ySEOFields() []core.Field {
	return []core.Field{
		{
			Key: "wcag_level", Label: "wcag_level    ", Kind: core.KindSelect,
			Options: []string{"A", "AA", "AAA", "None"},
			Value:   "AA", SelIdx: 1,
		},
		{
			Key: "seo_render_strategy", Label: "seo_rendering ", Kind: core.KindSelect,
			Options: []string{"SSR", "SSG", "ISR", "Prerender", "None"},
			Value:   "None", SelIdx: 4,
		},
		{
			Key: "sitemap", Label: "sitemap       ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "meta_tag_injection", Label: "meta_tags     ", Kind: core.KindSelect,
			Options: []string{"Manual", "Framework-native", "None"},
			Value:   "None", SelIdx: 2,
		},
		{
			Key: "analytics", Label: "analytics     ", Kind: core.KindSelect,
			Options: []string{"PostHog", "Google Analytics 4", "Plausible", "Mixpanel", "Segment", "Custom", "None"},
			Value:   "None", SelIdx: 6,
		},
		{
			Key: "telemetry", Label: "frontend_rum  ", Kind: core.KindSelect,
			Options: []string{"Sentry", "Datadog RUM", "LogRocket", "New Relic Browser", "Custom", "None"},
			Value:   "None", SelIdx: 5,
		},
	}
}

func defaultNavFields() []core.Field {
	return []core.Field{
		{
			Key: "nav_type", Label: "nav_type      ", Kind: core.KindSelect,
			Options: []string{
				"Top bar", "Sidebar", "Bottom tabs (mobile)",
				"Hamburger menu", "Combined",
			},
			Value: "Top bar",
		},
		{
			Key: "breadcrumbs", Label: "breadcrumbs   ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "auth_aware", Label: "auth_aware    ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "true", SelIdx: 1,
		},
	}
}
