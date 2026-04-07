package data

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Governance form fields + caching utilities + file storage fields ─────────

func defaultGovFormFields(dbAliases []string, cloudProvider string) []core.Field {
	dbPlaceholder := core.PlaceholderFor(dbAliases, "(no databases configured)")
	archivalOpts := archivalStorageOptionsFor(cloudProvider)
	backupOpts := govBackupStrategyOptions(nil, cloudProvider)
	return []core.Field{
		{Key: "name", Label: "policy name   ", Kind: core.KindText},
		{
			Key: "databases", Label: "applies to    ", Kind: core.KindMultiSelect,
			Options: dbAliases,
			Value:   dbPlaceholder,
		},
		{
			Key: "retention_policy", Label: "retention     ", Kind: core.KindSelect,
			Options: []string{"30 days", "90 days", "1 year", "3 years", "7 years", "Indefinite", "Custom"},
			Value:   "Indefinite", SelIdx: 5,
		},
		{
			Key: "delete_strategy", Label: "delete strat. ", Kind: core.KindSelect,
			Options: []string{"Soft-delete", "Hard-delete", "Archival", "Soft + periodic purge"},
			Value:   "Soft-delete",
		},
		{
			Key: "pii_encryption", Label: "pii encryption", Kind: core.KindSelect,
			Options: []string{"core.Field-level AES-256", "Full database encryption", "Application-level", "None"},
			Value:   "None", SelIdx: 3,
		},
		{
			Key: "compliance_frameworks", Label: "compliance    ", Kind: core.KindMultiSelect,
			Options: []string{"GDPR", "HIPAA", "SOC2 Type II", "PCI-DSS", "ISO-27001", "CCPA", "PIPEDA"},
		},
		{
			Key: "data_residency", Label: "data residency", Kind: core.KindSelect,
			Options: []string{"US", "EU", "APAC", "US + EU", "Global", "Custom"},
			Value:   "Global", SelIdx: 4,
		},
		{
			Key: "archival_storage", Label: "archival      ", Kind: core.KindSelect,
			Options: archivalOpts,
			Value:   archivalOpts[len(archivalOpts)-1],
			SelIdx:  len(archivalOpts) - 1,
		},
		{
			Key: "migration_tool", Label: "migration     ", Kind: core.KindSelect,
			Options: allMigrationTools,
			Value:   "None", SelIdx: len(allMigrationTools) - 1,
		},
		{
			Key: "backup_strategy", Label: "backup strat. ", Kind: core.KindSelect,
			Options: backupOpts,
			Value:   "None",
			SelIdx:  len(backupOpts) - 1,
		},
		{
			Key: "search_tech", Label: "search tech   ", Kind: core.KindSelect,
			Options: []string{"Elasticsearch", "Meilisearch", "Algolia", "Typesense", "None"},
			Value:   "None", SelIdx: 4,
		},
	}
}

func govFormFromDef(def manifest.DataGovernanceConfig, dbAliases []string, cloudProvider string) []core.Field {
	f := defaultGovFormFields(dbAliases, cloudProvider)
	f = core.SetFieldValue(f, "name", def.Name)
	if len(def.Databases) > 0 {
		f = core.RestoreMultiSelectValue(f, "databases", strings.Join(def.Databases, ", "))
	}
	if def.RetentionPolicy != "" {
		f = core.SetFieldValue(f, "retention_policy", def.RetentionPolicy)
	}
	if def.DeleteStrategy != "" {
		f = core.SetFieldValue(f, "delete_strategy", def.DeleteStrategy)
	}
	if def.PIIEncryption != "" {
		f = core.SetFieldValue(f, "pii_encryption", def.PIIEncryption)
	}
	if def.ComplianceFrameworks != "" {
		f = core.RestoreMultiSelectValue(f, "compliance_frameworks", def.ComplianceFrameworks)
	}
	if def.DataResidency != "" {
		f = core.SetFieldValue(f, "data_residency", def.DataResidency)
	}
	if def.ArchivalStorage != "" {
		f = core.SetFieldValue(f, "archival_storage", def.ArchivalStorage)
	}
	if def.MigrationTool != "" {
		f = core.SetFieldValue(f, "migration_tool", def.MigrationTool)
	}
	if def.BackupStrategy != "" {
		f = core.SetFieldValue(f, "backup_strategy", def.BackupStrategy)
	}
	if def.SearchTech != "" {
		f = core.SetFieldValue(f, "search_tech", def.SearchTech)
	}
	return f
}

func govDefFromForm(fields []core.Field) manifest.DataGovernanceConfig {
	var dbs []string
	for _, f := range fields {
		if f.Key == "databases" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					dbs = append(dbs, f.Options[idx])
				}
			}
			break
		}
	}
	return manifest.DataGovernanceConfig{
		Name:                 core.FieldGet(fields, "name"),
		Databases:            dbs,
		RetentionPolicy:      core.FieldGet(fields, "retention_policy"),
		DeleteStrategy:       core.FieldGet(fields, "delete_strategy"),
		PIIEncryption:        core.FieldGet(fields, "pii_encryption"),
		ComplianceFrameworks: core.FieldGetMulti(fields, "compliance_frameworks"),
		DataResidency:        core.FieldGet(fields, "data_residency"),
		ArchivalStorage:      core.FieldGet(fields, "archival_storage"),
		MigrationTool:        core.FieldGet(fields, "migration_tool"),
		BackupStrategy:       core.FieldGet(fields, "backup_strategy"),
		SearchTech:           core.FieldGet(fields, "search_tech"),
	}
}

// withRefreshedCachingEntities returns a copy of the DataTabEditor with the
// entities multiselect options in cachingForm updated to reflect current domain
// and DTO names.
func (dt DataTabEditor) withRefreshedCachingEntities() DataTabEditor {
	domOpts := dt.DomainNames()
	opts := make([]string, 0, len(domOpts)+len(dt.availableDTOs))
	opts = append(opts, domOpts...)
	for _, name := range dt.availableDTOs {
		opts = append(opts, "dto:"+name)
	}
	newFields := make([]core.Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key == "entities" {
			// Preserve existing selections by re-mapping
			oldOpts := newFields[i].Options
			newFields[i].Options = opts
			newFields[i].Value = core.PlaceholderFor(opts, "(no domains or DTOs configured)")
			newSelected := make([]int, 0)
			for _, oldIdx := range newFields[i].SelectedIdxs {
				if oldIdx < len(oldOpts) {
					oldVal := oldOpts[oldIdx]
					for j, newOpt := range opts {
						if newOpt == oldVal {
							newSelected = append(newSelected, j)
							break
						}
					}
				}
			}
			newFields[i].SelectedIdxs = newSelected
			break
		}
	}
	dt.cachingForm = newFields
	return dt
}

// isCachingFieldDisabled returns true when cache_db should be hidden because
// the selected layer is not "Dedicated cache".
func isCachingFieldDisabled(fields []core.Field, idx int) bool {
	if fields[idx].Key != "cache_db" {
		return false
	}
	for _, f := range fields {
		if f.Key == "layer" {
			return f.DisplayValue() != "Dedicated cache"
		}
	}
	return true
}

func nextCachingFormIdx(fields []core.Field, cur int) int {
	return core.NextFormIdx(fields, cur, isCachingFieldDisabled)
}

func prevCachingFormIdx(fields []core.Field, cur int) int {
	return core.PrevFormIdx(fields, cur, isCachingFieldDisabled)
}

// cachingVisibleFields returns only the fields that should be rendered.
func cachingVisibleFields(fields []core.Field) []core.Field {
	out := make([]core.Field, 0, len(fields))
	for i, f := range fields {
		if !isCachingFieldDisabled(fields, i) {
			out = append(out, f)
		}
	}
	return out
}

// cachingVisibleIdx maps a full-list index to its position in the visible list.
func cachingVisibleIdx(fields []core.Field, fullIdx int) int {
	vis := 0
	for i := range fullIdx {
		if !isCachingFieldDisabled(fields, i) {
			vis++
		}
	}
	return vis
}

// withRefreshedCachingDBs returns a copy of the DataTabEditor with the cache_db
// select options in cachingForm populated from database sources that have IsCache == true.
func (dt DataTabEditor) withRefreshedCachingDBs() DataTabEditor {
	var aliases []string
	for _, src := range dt.dbEditor.Sources {
		if src.IsCache {
			aliases = append(aliases, src.Alias)
		}
	}
	newFields := make([]core.Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key == "cache_db" {
			cur := newFields[i].Value
			newFields[i].Options = aliases
			newFields[i].Value = core.PlaceholderFor(aliases, "(no cache DBs configured)")
			newFields[i].SelIdx = 0
			for j, a := range aliases {
				if a == cur {
					newFields[i].SelIdx = j
					newFields[i].Value = a
					break
				}
			}
			if len(aliases) > 0 && newFields[i].Value == "" {
				newFields[i].Value = aliases[0]
			}
			break
		}
	}
	dt.cachingForm = newFields
	return dt
}

// strategyOptionsForLayer returns the valid caching strategy options for a given layer.
// CDN caching only supports read-oriented and CDN-specific strategies;
// application-level and dedicated cache do not expose CDN purge.
func strategyOptionsForLayer(layer string) []string {
	switch layer {
	case "CDN":
		return []string{"Cache-aside", "Read-through", "CDN purge"}
	case "Application-level", "Dedicated cache":
		return []string{"Cache-aside", "Read-through", "Write-through", "Write-behind"}
	default:
		return []string{"Cache-aside", "Read-through", "Write-through", "Write-behind", "CDN purge"}
	}
}

// withRefreshedCachingStrategies returns a copy of the DataTabEditor with the
// strategy multiselect options filtered to those valid for the currently selected layer.
// Existing selections that are no longer valid are dropped.
func (dt DataTabEditor) withRefreshedCachingStrategies() DataTabEditor {
	layer := core.FieldGet(dt.cachingForm, "layer")
	validOpts := strategyOptionsForLayer(layer)

	newFields := make([]core.Field, len(dt.cachingForm))
	copy(newFields, dt.cachingForm)
	for i := range newFields {
		if newFields[i].Key != "strategy" {
			continue
		}
		oldOpts := newFields[i].Options
		newFields[i].Options = validOpts
		// Re-map selected indices, dropping any that are no longer valid
		newSelected := make([]int, 0, len(newFields[i].SelectedIdxs))
		for _, oldIdx := range newFields[i].SelectedIdxs {
			if oldIdx >= len(oldOpts) {
				continue
			}
			oldVal := oldOpts[oldIdx]
			for j, opt := range validOpts {
				if opt == oldVal {
					newSelected = append(newSelected, j)
					break
				}
			}
		}
		newFields[i].SelectedIdxs = newSelected
		// Clamp DDCursor
		if newFields[i].DDCursor >= len(validOpts) {
			newFields[i].DDCursor = 0
		}
		break
	}
	dt.cachingForm = newFields
	return dt
}

func defaultFSFormFields(domainOptions []string, cloudProvider string, environmentNames []string, serviceOptions []string) []core.Field {
	techOpts := fsStorageOptionsFor(cloudProvider)
	envOpts, envDefault := core.NoneOrPlaceholder(environmentNames, "(no environments configured)")
	return []core.Field{
		{
			Key: "technology", Label: "technology    ", Kind: core.KindSelect,
			Options: techOpts,
			Value:   techOpts[0],
		},
		{Key: "purpose", Label: "purpose       ", Kind: core.KindText},
		{
			Key: "used_by_service", Label: "used_by       ", Kind: core.KindSelect,
			Options: append([]string{"(any / unspecified)"}, serviceOptions...),
			Value:   "(any / unspecified)",
		},
		{
			Key: "environment", Label: "environment   ", Kind: core.KindSelect,
			Options: envOpts,
			Value:   envDefault,
		},
		{
			Key: "access", Label: "access        ", Kind: core.KindSelect,
			Options: []string{"Public (CDN-fronted)", "Private (signed URLs)", "Internal only"},
			Value:   "Private (signed URLs)", SelIdx: 1,
		},
		{
			Key: "max_size", Label: "max_size      ", Kind: core.KindSelect,
			Options: []string{"1 MB", "5 MB", "10 MB", "25 MB", "50 MB", "100 MB", "500 MB", "1 GB", "Unlimited"},
			Value:   "10 MB", SelIdx: 2,
		},
		{
			Key: "domains", Label: "domains       ", Kind: core.KindMultiSelect,
			Options: domainOptions,
			Value:   core.PlaceholderFor(domainOptions, "(no domains configured)"),
		},
		{
			Key: "ttl_minutes", Label: "ttl_minutes   ", Kind: core.KindSelect,
			Options: []string{"30", "60", "1440", "10080", "Custom"},
			Value:   "1440", SelIdx: 2,
		},
		{
			Key: "allowed_types", Label: "allowed_types ", Kind: core.KindMultiSelect,
			Options: []string{"image/*", "application/pdf", "video/*", "audio/*", "text/*", "application/json"},
		},
	}
}

func fsFormFromDef(def manifest.FileStorageDef, domainOptions []string, cloudProvider string, environmentNames []string, serviceOptions []string) []core.Field {
	f := defaultFSFormFields(domainOptions, cloudProvider, environmentNames, serviceOptions)
	f = core.SetFieldValue(f, "technology", def.Technology)
	f = core.SetFieldValue(f, "purpose", def.Purpose)
	if def.UsedByService != "" {
		f = core.SetFieldValue(f, "used_by_service", def.UsedByService)
	}
	f = core.SetFieldValue(f, "environment", def.Environment)
	if def.Access != "" {
		f = core.SetFieldValue(f, "access", def.Access)
	}
	f = core.SetFieldValue(f, "max_size", def.MaxSize)
	f = core.SetFieldValue(f, "ttl_minutes", def.TTLMinutes)
	f = core.SetFieldValue(f, "allowed_types", def.AllowedTypes)
	// Restore multi-select for domains
	if def.Domains != "" {
		for i := range f {
			if f[i].Key == "domains" {
				for _, sel := range strings.Split(def.Domains, ", ") {
					for j, opt := range f[i].Options {
						if opt == sel {
							f[i].SelectedIdxs = append(f[i].SelectedIdxs, j)
						}
					}
				}
				break
			}
		}
	}
	return f
}

func fsDefFromForm(fields []core.Field) manifest.FileStorageDef {
	usedBy := core.FieldGet(fields, "used_by_service")
	if usedBy == "(any / unspecified)" {
		usedBy = ""
	}
	return manifest.FileStorageDef{
		Technology:    core.FieldGet(fields, "technology"),
		Purpose:       core.FieldGet(fields, "purpose"),
		UsedByService: usedBy,
		Environment:   core.FieldGet(fields, "environment"),
		Access:        core.FieldGet(fields, "access"),
		MaxSize:       core.FieldGet(fields, "max_size"),
		Domains:       core.FieldGetMulti(fields, "domains"),
		TTLMinutes:    core.FieldGet(fields, "ttl_minutes"),
		AllowedTypes:  core.FieldGetMulti(fields, "allowed_types"),
	}
}
