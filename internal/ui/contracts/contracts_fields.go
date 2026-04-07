package contracts

import "github.com/vibe-menu/internal/ui/core"

// ── field definitions ─────────────────────────────────────────────────────────

func defaultDTOFormFields(domainOptions []string) []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key: "category", Label: "category      ", Kind: core.KindSelect,
			Options: []string{"Request", "Response", "Event Payload", "Shared/Common"},
			Value:   "Request",
		},
		{
			Key: "source_domains", Label: "source_domains", Kind: core.KindMultiSelect,
			Options: domainOptions,
			Value:   core.PlaceholderFor(domainOptions, "(no domains configured)"),
		},
		{Key: "description", Label: "description   ", Kind: core.KindText},
		{
			Key: "protocol", Label: "protocol      ", Kind: core.KindSelect,
			Options: []string{"REST/JSON", "Protobuf", "Avro", "MessagePack", "Thrift", "FlatBuffers", "Cap'n Proto"},
			Value:   "REST/JSON",
		},
		// ── Protobuf-specific ────────────────────────────────────────────────────
		{Key: "proto_package", Label: "proto_package ", Kind: core.KindText},
		{
			Key: "proto_syntax", Label: "proto_syntax  ", Kind: core.KindSelect,
			Options: []string{"proto3", "proto2"}, Value: "proto3",
		},
		{Key: "proto_options", Label: "proto_options ", Kind: core.KindText},
		// ── Avro-specific ────────────────────────────────────────────────────────
		{Key: "avro_namespace", Label: "avro_namespace", Kind: core.KindText},
		{Key: "schema_registry", Label: "schema_reg    ", Kind: core.KindText},
		// ── Thrift-specific ──────────────────────────────────────────────────────
		{Key: "thrift_namespace", Label: "thrift_ns     ", Kind: core.KindText},
		{
			Key: "thrift_language", Label: "thrift_lang   ", Kind: core.KindSelect,
			Options: []string{"go", "java", "python", "cpp", "js", "php", "ruby"},
			Value:   "go",
		},
		// ── FlatBuffers / Cap'n Proto ────────────────────────────────────────────
		{Key: "namespace", Label: "namespace     ", Kind: core.KindText},
	}
}

// typeOptionsForDTOProtocol returns the native types for a given DTO serialisation protocol.
func typeOptionsForDTOProtocol(proto string) []string {
	switch proto {
	case "Protobuf":
		return []string{
			"string", "bool", "bytes",
			"int32", "int64", "uint32", "uint64", "sint32", "sint64",
			"fixed32", "fixed64", "sfixed32", "sfixed64",
			"float", "double",
			"enum", "message", "repeated", "map", "oneof",
			"google.Any", "google.Timestamp", "google.Duration",
		}
	case "Avro":
		return []string{
			"null", "boolean", "int", "long", "float", "double",
			"bytes", "string",
			"record", "enum", "array", "map", "union", "fixed",
		}
	case "MessagePack":
		return []string{
			"string", "int", "float", "bool", "binary",
			"array", "map", "nil", "timestamp", "ext",
		}
	case "Thrift":
		return []string{
			"bool", "byte", "i16", "i32", "i64", "double",
			"string", "binary",
			"list", "set", "map", "enum", "struct", "void",
		}
	case "FlatBuffers":
		return []string{
			"bool",
			"int8", "int16", "int32", "int64",
			"uint8", "uint16", "uint32", "uint64",
			"float32", "float64",
			"string", "[type]", "struct", "table", "enum", "union",
		}
	case "Cap'n Proto":
		return []string{
			"Bool",
			"Int8", "Int16", "Int32", "Int64",
			"UInt8", "UInt16", "UInt32", "UInt64",
			"Float32", "Float64",
			"Text", "Data",
			"List", "Struct", "Enum", "Union", "AnyPointer",
		}
	default: // REST/JSON
		return []string{
			"string", "int", "float", "boolean", "datetime",
			"uuid", "enum(values)", "array(type)", "nested(DTO)", "map(key,value)",
		}
	}
}

func defaultDTOFieldForm(protocol string) []core.Field {
	typeOpts := typeOptionsForDTOProtocol(protocol)
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key: "type", Label: "type          ", Kind: core.KindSelect,
			Options: typeOpts, Value: typeOpts[0],
		},
		// ── REST/JSON · MessagePack · Avro ───────────────────────────────────────
		{
			Key: "required", Label: "required      ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "nullable", Label: "nullable      ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{
			Key: "validation", Label: "validation    ", Kind: core.KindMultiSelect,
			Options: []string{
				"required", "min_length", "max_length", "min_value", "max_value",
				"email", "url", "regex", "uuid", "enum", "phone", "pattern", "custom",
			},
		},
		// ── Default value (Avro, Thrift, FlatBuffers, Cap'n Proto, REST/JSON) ───
		{Key: "default", Label: "default       ", Kind: core.KindText},
		// ── Protobuf-specific ────────────────────────────────────────────────────
		{Key: "field_number", Label: "field_number  ", Kind: core.KindText},
		{
			Key: "proto_modifier", Label: "proto_modifier", Kind: core.KindSelect,
			Options: []string{"optional", "repeated", "oneof"}, Value: "optional",
		},
		{Key: "json_name", Label: "json_name     ", Kind: core.KindText},
		// ── Thrift / Cap'n Proto ─────────────────────────────────────────────────
		{Key: "field_id", Label: "field_id      ", Kind: core.KindText},
		// ── Thrift-specific ──────────────────────────────────────────────────────
		{
			Key: "thrift_mod", Label: "thrift_mod    ", Kind: core.KindSelect,
			Options: []string{"required", "optional", "default"}, Value: "optional",
		},
		// ── FlatBuffers-specific ─────────────────────────────────────────────────
		{
			Key: "deprecated", Label: "deprecated    ", Kind: core.KindSelect,
			Options: core.OptionsOffOn, Value: "false",
		},
		{Key: "notes", Label: "notes         ", Kind: core.KindText},
	}
}

// refreshDTOFieldTypeOptions updates the type field options in a field form to match
// the given protocol, preserving the current value when possible.
func refreshDTOFieldTypeOptions(form []core.Field, protocol string) []core.Field {
	opts := typeOptionsForDTOProtocol(protocol)
	for i := range form {
		if form[i].Key != "type" {
			continue
		}
		cur := form[i].DisplayValue()
		form[i].Options = opts
		form[i].SelIdx = 0
		form[i].Value = opts[0]
		for j, t := range opts {
			if t == cur {
				form[i].SelIdx = j
				form[i].Value = t
				break
			}
		}
		break
	}
	return form
}

// currentDTOProtocol returns the serialisation protocol selected in the DTO form.
func (ce ContractsEditor) currentDTOProtocol() string {
	proto := core.FieldGet(ce.dtoForm, "protocol")
	if proto == "" {
		return "REST/JSON"
	}
	return proto
}

// visibleDTOFieldFormFields returns only the field-form fields relevant to the
// current DTO protocol, hiding inapplicable options.
func (ce ContractsEditor) visibleDTOFieldFormFields() []core.Field {
	proto := ce.currentDTOProtocol()
	var visible []core.Field
	for _, f := range ce.dtoFieldForm {
		switch f.Key {
		case "required", "nullable":
			if proto != "REST/JSON" && proto != "MessagePack" && proto != "Avro" {
				continue
			}
		case "validation":
			if proto != "REST/JSON" && proto != "MessagePack" {
				continue
			}
		case "default":
			if proto == "Protobuf" {
				continue
			}
		case "field_number", "proto_modifier", "json_name":
			if proto != "Protobuf" {
				continue
			}
		case "field_id":
			if proto != "Thrift" && proto != "Cap'n Proto" {
				continue
			}
		case "thrift_mod":
			if proto != "Thrift" {
				continue
			}
		case "deprecated":
			if proto != "FlatBuffers" {
				continue
			}
		}
		visible = append(visible, f)
	}
	return visible
}

// dtoFieldFormFieldByKey returns a pointer to the field-form field with the given key.
func (ce *ContractsEditor) dtoFieldFormFieldByKey(key string) *core.Field {
	for i := range ce.dtoFieldForm {
		if ce.dtoFieldForm[i].Key == key {
			return &ce.dtoFieldForm[i]
		}
	}
	return nil
}
