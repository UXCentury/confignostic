// Package v1 implements the initial HTTP, event and runner wire contracts.
package v1

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/UXCentury/confignostic/internal/workflows"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	Version          = "1"
	MaxRequestBytes  = 256 << 10
	MaxEventBytes    = 64 << 10
	MaxRunnerBytes   = 64 << 10
	MaxMessageBytes  = 32 << 10
	MaxArtifactBytes = 64 << 20
	MaxJSONDepth     = 32
)

var (
	ErrInvalidJSON        = errors.New("invalid JSON document")
	ErrInvalidSchema      = errors.New("document does not match schema v1")
	ErrUnsupportedVersion = errors.New("unsupported schema version")
	ErrTooLarge           = errors.New("document exceeds size limit")
	ErrScopeMismatch      = errors.New("workspace scope mismatch")
	ErrSchemaUnavailable  = errors.New("embedded schema unavailable")
)

//go:embed schema/*.json
var schemaFiles embed.FS
var schemaOnce sync.Once
var compiled map[string]*jsonschema.Schema
var compileErr error

type offlineLoader struct{}

func (offlineLoader) Load(string) (any, error) { return nil, ErrSchemaUnavailable }

func loadSchemas() {
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(offlineLoader{})
	compiler.AssertFormat()
	entries, err := schemaFiles.ReadDir("schema")
	if err != nil {
		compileErr = errors.Join(ErrSchemaUnavailable, err)
		return
	}
	for _, entry := range entries {
		data, err := schemaFiles.ReadFile("schema/" + entry.Name())
		if err != nil {
			compileErr = errors.Join(ErrSchemaUnavailable, err)
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			compileErr = errors.Join(ErrSchemaUnavailable, err)
			return
		}
		if err = compiler.AddResource("https://confignostic.invalid/v1/"+entry.Name(), doc); err != nil {
			compileErr = errors.Join(ErrSchemaUnavailable, err)
			return
		}
	}
	compiled = map[string]*jsonschema.Schema{}
	for _, entry := range entries {
		schema, err := compiler.Compile("https://confignostic.invalid/v1/" + entry.Name())
		if err != nil {
			compileErr = errors.Join(ErrSchemaUnavailable, err)
			return
		}
		compiled[entry.Name()] = schema
	}
}

// scanJSON rejects duplicate keys, nulls, invalid UTF-8, excessive nesting and
// trailing documents before schema validation. Optional fields must be omitted.
func scanJSON(data []byte) error {
	if !utf8.Valid(data) {
		return ErrInvalidJSON
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var visit func(int) error
	visit = func(depth int) error {
		if depth > MaxJSONDepth {
			return ErrInvalidJSON
		}
		token, err := decoder.Token()
		if err != nil || token == nil {
			return ErrInvalidJSON
		}
		// V1's numeric JSON fields are uint32 values. Counters use strings.
		// Bound numeric literals before a schema engine could expand exponents.
		if number, ok := token.(json.Number); ok {
			if len(number) > 10 {
				return ErrInvalidJSON
			}
			if _, err := strconv.ParseUint(string(number), 10, 32); err != nil {
				return ErrInvalidJSON
			}
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				name, ok := key.(string)
				if err != nil || !ok || seen[name] {
					return ErrInvalidJSON
				}
				seen[name] = true
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			close, err := decoder.Token()
			if err != nil || close != json.Delim('}') {
				return ErrInvalidJSON
			}
		case '[':
			for decoder.More() {
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			close, err := decoder.Token()
			if err != nil || close != json.Delim(']') {
				return ErrInvalidJSON
			}
		default:
			return ErrInvalidJSON
		}
		return nil
	}
	if err := visit(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}

func decode(data []byte, limit int, schemaName string, target any) error {
	if len(data) > limit {
		return ErrTooLarge
	}
	if err := scanJSON(data); err != nil {
		return err
	}
	var header map[string]json.RawMessage
	if json.Unmarshal(data, &header) != nil {
		return ErrInvalidSchema
	}
	var version string
	if json.Unmarshal(header["schema_version"], &version) != nil {
		return ErrInvalidSchema
	}
	if version != Version {
		return ErrUnsupportedVersion
	}
	schemaOnce.Do(loadSchemas)
	if compileErr != nil {
		return compileErr
	}
	schema, ok := compiled[schemaName]
	if !ok {
		return ErrSchemaUnavailable
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return ErrInvalidJSON
	}
	if schema.Validate(value) != nil {
		return ErrInvalidSchema
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return ErrInvalidSchema
	}
	// Escaping can expand a compact incoming string. Accepted values must also
	// fit when serialized by our encoder, so a decode/encode round trip is safe.
	canonical, err := json.Marshal(target)
	if err != nil {
		return ErrInvalidSchema
	}
	if len(canonical) > limit {
		return ErrTooLarge
	}
	return nil
}

func DecodeWorkflow(data []byte) (workflows.Definition, error) {
	var result workflows.Definition
	if err := decode(data, MaxRequestBytes, "workflow.schema.json", &result); err != nil {
		return result, err
	}
	return result, workflows.Validate(result)
}
