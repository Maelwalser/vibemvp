package ui

import "github.com/vibe-menu/internal/manifest"

// ── Column form helpers ───────────────────────────────────────────────────────

// defaultColForm returns a fresh, zeroed column form with all 16 fields.
func defaultColForm() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "type", Label: "type          ", Kind: KindSelect,
			Options: []string{
				"text", "varchar", "char", "int", "bigint", "smallint",
				"serial", "bigserial", "boolean", "float", "double", "decimal",
				"json", "jsonb", "uuid", "timestamp", "timestamptz",
				"date", "time", "bytea", "enum", "array", "other",
			},
			Value: "text",
		},
		{Key: "length", Label: "length        ", Kind: KindText},
		{Key: "nullable", Label: "nullable      ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
		},
		{Key: "primary_key", Label: "primary_key   ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
		},
		{Key: "unique", Label: "unique        ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
		},
		{Key: "default", Label: "default       ", Kind: KindText},
		{Key: "check", Label: "check         ", Kind: KindText},
		{Key: "foreign_key", Label: "foreign_key   ", Kind: KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		{Key: "fk_entity", Label: "  fk_entity   ", Kind: KindText},
		{Key: "fk_column", Label: "  fk_column   ", Kind: KindText},
		{Key: "fk_on_delete", Label: "  fk_on_delete", Kind: KindSelect,
			Options: []string{"NO ACTION", "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT"},
			Value:   "NO ACTION",
		},
		{Key: "fk_on_update", Label: "  fk_on_update", Kind: KindSelect,
			Options: []string{"NO ACTION", "RESTRICT", "CASCADE", "SET NULL", "SET DEFAULT"},
			Value:   "NO ACTION",
		},
		{Key: "indexed", Label: "indexed       ", Kind: KindSelect,
			Options: []string{"no", "yes"}, Value: "no",
		},
		{Key: "index_type", Label: "  index_type  ", Kind: KindSelect,
			Options: []string{"btree", "hash", "gin", "gist", "brin"},
			Value:   "btree",
		},
		{Key: "notes", Label: "notes         ", Kind: KindText},
	}
}

// colFormFromColumnDef populates a form from an existing ColumnDef.
func colFormFromColumnDef(col manifest.ColumnDef) []Field {
	f := defaultColForm()

	setVal := func(key, val string) {
		for i := range f {
			if f[i].Key != key {
				continue
			}
			f[i].Value = val
			if f[i].Kind == KindSelect {
				for j, opt := range f[i].Options {
					if opt == val {
						f[i].SelIdx = j
						break
					}
				}
			}
			return
		}
	}

	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	setVal("name", col.Name)
	if col.Type != "" {
		setVal("type", string(col.Type))
	}
	setVal("length", col.Length)
	setVal("nullable", boolStr(col.Nullable))
	setVal("primary_key", boolStr(col.PrimaryKey))
	setVal("unique", boolStr(col.Unique))
	setVal("default", col.Default)
	setVal("check", col.Check)
	if col.ForeignKey != nil {
		setVal("foreign_key", "yes")
		setVal("fk_entity", col.ForeignKey.RefEntity)
		setVal("fk_column", col.ForeignKey.RefColumn)
		setVal("fk_on_delete", string(col.ForeignKey.OnDelete))
		setVal("fk_on_update", string(col.ForeignKey.OnUpdate))
	}
	if col.Index {
		setVal("indexed", "yes")
		if col.IndexType != "" {
			setVal("index_type", string(col.IndexType))
		}
	}
	setVal("notes", col.Notes)
	return f
}

// colFormToColumnDef converts the current form state back to a ColumnDef.
func colFormToColumnDef(form []Field) manifest.ColumnDef {
	get := func(key string) string {
		for _, f := range form {
			if f.Key == key {
				return f.DisplayValue()
			}
		}
		return ""
	}

	col := manifest.ColumnDef{
		Name:       get("name"),
		Type:       manifest.ColumnType(get("type")),
		Length:     get("length"),
		Nullable:   get("nullable") == "true",
		PrimaryKey: get("primary_key") == "true",
		Unique:     get("unique") == "true",
		Default:    get("default"),
		Check:      get("check"),
		Index:      get("indexed") == "yes",
		Notes:      get("notes"),
	}
	if col.Index {
		col.IndexType = manifest.IndexType(get("index_type"))
	}
	if get("foreign_key") == "yes" {
		col.ForeignKey = &manifest.ForeignKey{
			RefEntity: get("fk_entity"),
			RefColumn: get("fk_column"),
			OnDelete:  manifest.CascadeAction(get("fk_on_delete")),
			OnUpdate:  manifest.CascadeAction(get("fk_on_update")),
		}
	}
	return col
}

// isColFormFieldDisabled returns true when a field is gated behind a parent toggle.
func isColFormFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	switch key {
	case "fk_entity", "fk_column", "fk_on_delete", "fk_on_update":
		for _, f := range form {
			if f.Key == "foreign_key" {
				return f.DisplayValue() != "yes"
			}
		}
	case "index_type":
		for _, f := range form {
			if f.Key == "indexed" {
				return f.DisplayValue() != "yes"
			}
		}
	}
	return false
}

func nextColFormIdx(form []Field, cur int) int { return nextFormIdx(form, cur, isColFormFieldDisabled) }
func prevColFormIdx(form []Field, cur int) int { return prevFormIdx(form, cur, isColFormFieldDisabled) }

// ── Entity settings form helpers ──────────────────────────────────────────────

// buildEntitySettingsForm constructs the entity-level settings form, dynamically
// populating database and cache_store selects from the available DBSources.
func buildEntitySettingsForm(ent manifest.EntityDef, dbs []manifest.DBSourceDef) []Field {
	var dbAliases, cacheAliases []string
	for _, db := range dbs {
		dbAliases = append(dbAliases, db.Alias)
		if db.IsCache {
			cacheAliases = append(cacheAliases, db.Alias)
		}
	}

	dbOptions, dbDefault := noneOrPlaceholder(dbAliases, "(no databases configured)")
	cacheOptions, cacheDefault := noneOrPlaceholder(cacheAliases, "(no cache DBs configured)")

	findIdx := func(opts []string, val string) int {
		for i, o := range opts {
			if o == val {
				return i
			}
		}
		return 0
	}

	cachedVal := "no"
	cachedIdx := 0
	if ent.Cached {
		cachedVal = "yes"
		cachedIdx = 1
	}

	dbVal := ent.Database
	if dbVal == "" {
		dbVal = dbDefault
	}
	cacheVal := ent.CacheStore
	if cacheVal == "" {
		cacheVal = cacheDefault
	}

	return []Field{
		{Key: "database", Label: "database      ", Kind: KindSelect,
			Options: dbOptions,
			SelIdx:  findIdx(dbOptions, dbVal),
			Value:   dbVal,
		},
		{Key: "description", Label: "description   ", Kind: KindText, Value: ent.Description},
		{Key: "cached", Label: "cached        ", Kind: KindSelect,
			Options: []string{"no", "yes"},
			SelIdx:  cachedIdx,
			Value:   cachedVal,
		},
		{Key: "cache_store", Label: "  cache_store ", Kind: KindSelect,
			Options: cacheOptions,
			SelIdx:  findIdx(cacheOptions, cacheVal),
			Value:   cacheVal,
		},
		{Key: "cache_ttl", Label: "  cache_ttl   ", Kind: KindText, Value: ent.CacheTTL},
		{Key: "notes", Label: "notes         ", Kind: KindText, Value: ent.Notes},
	}
}

func entitySettingsToEntityDef(form []Field, ent manifest.EntityDef) manifest.EntityDef {
	get := func(key string) string {
		for _, f := range form {
			if f.Key == key {
				return f.DisplayValue()
			}
		}
		return ""
	}
	db := get("database")
	if db == "(none)" {
		db = ""
	}
	ent.Database = db
	ent.Description = get("description")
	ent.Cached = get("cached") == "yes"
	cs := get("cache_store")
	if cs == "(none)" {
		cs = ""
	}
	ent.CacheStore = cs
	ent.CacheTTL = get("cache_ttl")
	ent.Notes = get("notes")
	return ent
}

func isEntFormFieldDisabled(form []Field, idx int) bool {
	key := form[idx].Key
	if key == "cache_store" || key == "cache_ttl" {
		for _, f := range form {
			if f.Key == "cached" {
				return f.DisplayValue() != "yes"
			}
		}
	}
	return false
}

func nextEntFormIdx(form []Field, cur int) int { return nextFormIdx(form, cur, isEntFormFieldDisabled) }
func prevEntFormIdx(form []Field, cur int) int { return prevFormIdx(form, cur, isEntFormFieldDisabled) }

