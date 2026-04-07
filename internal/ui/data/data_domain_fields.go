package data

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Domain form field definitions ────────────────────────────────────────────

func defaultDomainFormFields(dbOptions []string) []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{Key: "description", Label: "description   ", Kind: core.KindText},
		{
			Key: "databases", Label: "databases     ", Kind: core.KindMultiSelect,
			Options: dbOptions,
			Value:   core.PlaceholderFor(dbOptions, "(no databases configured)"),
		},
	}
}

// typesForDBTech returns native data types for a given database technology.
func typesForDBTech(tech string) []string {
	switch tech {
	case "PostgreSQL":
		return []string{
			"varchar", "text", "char", "int", "bigint", "smallint",
			"serial", "bigserial", "boolean", "float", "double precision",
			"decimal", "numeric", "uuid", "timestamp", "timestamptz",
			"date", "time", "interval", "json", "jsonb", "bytea",
			"enum", "array", "inet", "tsvector", "xml",
		}
	case "MySQL":
		return []string{
			"varchar", "text", "char", "tinytext", "mediumtext", "longtext",
			"int", "bigint", "smallint", "tinyint", "mediumint",
			"float", "double", "decimal",
			"boolean", "date", "datetime", "timestamp", "time", "year",
			"json", "binary", "varbinary", "blob", "enum", "set",
		}
	case "SQLite":
		return []string{"TEXT", "INTEGER", "REAL", "NUMERIC", "BLOB", "NULL"}
	case "MongoDB":
		return []string{
			"String", "Int32", "Int64", "Double", "Decimal128",
			"Boolean", "Date", "ObjectId", "UUID",
			"Array", "Object", "Binary", "Null", "Timestamp", "Mixed",
		}
	case "DynamoDB":
		return []string{
			"String (S)", "Number (N)", "Binary (B)",
			"StringSet (SS)", "NumberSet (NS)", "BinarySet (BS)",
			"List (L)", "Map (M)", "Boolean (BOOL)", "Null (NULL)",
		}
	case "Cassandra":
		return []string{
			"text", "varchar", "ascii", "int", "bigint", "smallint", "tinyint", "varint",
			"float", "double", "decimal", "boolean",
			"date", "timestamp", "time", "uuid", "timeuuid",
			"blob", "list", "set", "map", "tuple", "frozen",
		}
	case "Redis", "Memcached":
		return []string{"String", "List", "Set", "Sorted Set", "Hash", "Stream"}
	case "ClickHouse":
		return []string{
			"UInt8", "UInt16", "UInt32", "UInt64",
			"Int8", "Int16", "Int32", "Int64",
			"Float32", "Float64", "Decimal",
			"String", "FixedString", "Date", "DateTime", "UUID",
			"Array", "Tuple", "Nullable", "Enum", "LowCardinality",
		}
	case "Elasticsearch":
		return []string{
			"text", "keyword", "long", "integer", "short", "byte",
			"double", "float", "boolean", "date", "binary", "ip",
			"object", "nested", "geo_point",
		}
	default:
		return []string{
			"String", "Int", "Float", "Boolean", "DateTime",
			"UUID", "JSON", "Binary", "Array", "Enum", "Ref",
		}
	}
}

// attrTypesForSources resolves attribute type options for a domain based on its
// selected database aliases. Returns (nil, true) when no databases are selected.
// When multiple databases are selected, types are merged (deduplicated, first-seen order).
func attrTypesForSources(selectedDBs string, sources []manifest.DBSourceDef) (types []string, noDB bool) {
	if selectedDBs == "" {
		return nil, true
	}
	seen := map[string]bool{}
	for _, alias := range strings.Split(selectedDBs, ", ") {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		for _, src := range sources {
			if src.Alias == alias {
				for _, t := range typesForDBTech(string(src.Type)) {
					if !seen[t] {
						seen[t] = true
						types = append(types, t)
					}
				}
			}
		}
	}
	if len(types) == 0 {
		return nil, true
	}
	return types, false
}

// refreshAttrTypeOptions updates the "type" field options in an attr form to match
// the current database selection, preserving the selected value when possible.
func refreshAttrTypeOptions(form []core.Field, types []string) []core.Field {
	for i := range form {
		if form[i].Key != "type" {
			continue
		}
		cur := form[i].DisplayValue()
		if len(types) == 0 {
			form[i].Options = nil
			form[i].Value = "(select a database first)"
			form[i].SelIdx = 0
		} else {
			form[i].Options = types
			form[i].SelIdx = 0
			form[i].Value = types[0]
			for j, t := range types {
				if t == cur {
					form[i].SelIdx = j
					form[i].Value = t
					break
				}
			}
		}
		break
	}
	return form
}

// currentDomainAttrTypes returns type options for the domain currently being edited,
// based on its selected databases. Returns nil when no databases are selected.
func (dt DataTabEditor) currentDomainAttrTypes() []string {
	dbs := core.FieldGetMulti(dt.domainForm, "databases")
	types, _ := attrTypesForSources(dbs, dt.dbEditor.Sources)
	return types
}

func defaultAttrFields(types []string) []core.Field {
	var typeOpts []string
	var typeVal string
	if len(types) == 0 {
		// No database selected — use empty options so DisplayValue returns the placeholder.
		typeVal = "(select a database first)"
	} else {
		typeOpts = types
		typeVal = types[0]
	}
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key: "type", Label: "type          ", Kind: core.KindSelect,
			Options: typeOpts,
			Value:   typeVal,
		},
		{
			Key: "constraints", Label: "constraints   ", Kind: core.KindMultiSelect,
			Options: []string{
				"required", "unique", "not_null", "min", "max",
				"min_length", "max_length", "email", "url", "regex",
				"positive", "future", "past", "enum",
			},
		},
		{Key: "default", Label: "default       ", Kind: core.KindText},
		{
			Key: "sensitive", Label: "sensitive     ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "validation", Label: "validation    ", Kind: core.KindMultiSelect,
			Options: []string{
				"email", "url", "regex", "min_length", "max_length",
				"min_value", "max_value", "phone", "uuid", "date_format", "enum", "custom",
			},
		},
		{
			Key: "indexed", Label: "indexed       ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "unique", Label: "unique        ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
	}
}

func defaultRelFields(domainOptions []string) []core.Field {
	return []core.Field{
		{
			Key: "related_domain", Label: "related_domain", Kind: core.KindSelect,
			Options: domainOptions,
			Value:   core.PlaceholderFor(domainOptions, "(no domains configured)"),
		},
		{
			Key: "rel_type", Label: "rel_type      ", Kind: core.KindSelect,
			Options: []string{"One-to-One", "One-to-Many", "Many-to-Many"},
			Value:   "One-to-Many",
		},
		{
			Key: "cascade", Label: "cascade       ", Kind: core.KindSelect,
			Options: []string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION", "SET DEFAULT"},
			Value:   "NO ACTION",
		},
	}
}
