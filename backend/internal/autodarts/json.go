package autodarts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrInvalidPayload = errors.New("invalid_payload")
var ErrInvalidJSON = errors.New("invalid_json")
var ErrPayloadTooLarge = errors.New("payload_too_large")
var ErrUnsupportedVersion = errors.New("unsupported_version")

// Check the entire input, including discarded fields, before typed projection.
func checkedJSON(body []byte) error {
	if len(body) > 128*1024 {
		return ErrPayloadTooLarge
	}
	if !utf8.Valid(body) {
		return ErrInvalidJSON
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if err := walkJSON(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}

func walkJSON(d *json.Decoder, depth int) error {
	token, err := d.Token()
	if err != nil {
		return ErrInvalidJSON
	}
	switch value := token.(type) {
	case json.Number:
		n, err := strconv.ParseFloat(string(value), 64)
		if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
			return ErrInvalidJSON
		}
	case json.Delim:
		if depth >= 12 {
			return ErrInvalidPayload
		}
		keys := map[string]bool{}
		count := 0
		for d.More() {
			count++
			if count > 3000 {
				return ErrInvalidPayload
			}
			if value == '{' {
				key, err := d.Token()
				if err != nil {
					return ErrInvalidJSON
				}
				s, ok := key.(string)
				if !ok || keys[s] {
					return ErrInvalidJSON
				}
				keys[s] = true
			}
			if err := walkJSON(d, depth+1); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if err != nil || (value == '{' && end != json.Delim('}')) || (value == '[' && end != json.Delim(']')) {
			return ErrInvalidJSON
		}
	}
	return nil
}

// The DTO is the allowlist. Check presence/nullability separately because Go's
// decoder otherwise silently accepts missing fields and null scalar values.
func project(body []byte, typ reflect.Type) ([]byte, error) {
	if typ.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
			return []byte("null"), nil
		}
		return project(body, typ.Elem())
	}
	if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return nil, ErrInvalidPayload
	}
	switch typ.Kind() {
	case reflect.Struct:
		var object map[string]json.RawMessage
		if err := json.Unmarshal(body, &object); err != nil {
			return nil, ErrInvalidPayload
		}
		filtered := map[string]json.RawMessage{}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			value, ok := object[name]
			if !ok {
				if field.Tag.Get("optional") == "true" {
					continue
				}
				return nil, ErrInvalidPayload
			}
			projected, err := project(value, field.Type)
			if err != nil {
				return nil, err
			}
			filtered[name] = projected
		}
		return json.Marshal(filtered)
	case reflect.Slice:
		var array []json.RawMessage
		if err := json.Unmarshal(body, &array); err != nil {
			return nil, ErrInvalidPayload
		}
		for i, item := range array {
			value, err := project(item, typ.Elem())
			if err != nil {
				return nil, err
			}
			array[i] = value
		}
		return json.Marshal(array)
	default:
		if err := json.Unmarshal(body, reflect.New(typ).Interface()); err != nil {
			return nil, ErrInvalidPayload
		}
		return body, nil
	}
}
