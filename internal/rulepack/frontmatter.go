package rulepack

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v4"
)

func decodeStrictJSON(data []byte, target any) error {
	if err := rejectDuplicateJSONFields(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("inventory JSON contains multiple values")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("JSON object key is not a string")
				}
				if _, duplicate := seen[key]; duplicate {
					return fmt.Errorf("duplicate JSON field %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delim)
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("inventory JSON contains multiple values")
		}
		return err
	}
	return nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func yamlDiagnostic(sourcePath string, err error, recovery string) Diagnostic {
	line := 1
	var loadError *yaml.LoadError
	if errors.As(err, &loadError) && loadError.Mark.Line > 0 {
		line = loadError.Mark.Line + 1
	}
	return Diagnostic{
		Path:     sourcePath,
		Line:     line,
		Field:    "frontmatter",
		Message:  err.Error(),
		Recovery: recovery,
	}
}

func prefixDiagnosticFields(diagnostics []Diagnostic, prefix string) []Diagnostic {
	for index := range diagnostics {
		diagnostics[index].Field = prefix + diagnostics[index].Field
	}
	return diagnostics
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	var opening []byte
	var closing []byte
	switch {
	case bytes.HasPrefix(data, []byte("---\n")):
		opening = []byte("---\n")
		closing = []byte("\n---\n")
	case bytes.HasPrefix(data, []byte("---\r\n")):
		opening = []byte("---\r\n")
		closing = []byte("\r\n---\r\n")
	default:
		return nil, nil, errors.New("file must begin with --- on its own line")
	}
	rest := data[len(opening):]
	index := bytes.Index(rest, closing)
	if index < 0 {
		return nil, nil, errors.New("frontmatter closing --- marker is missing")
	}
	frontmatter := rest[:index]
	body := rest[index+len(closing):]
	return frontmatter, body, nil
}

func diagnostic(filePath, field, message, recovery string) Diagnostic {
	return Diagnostic{
		Path:     filePath,
		Field:    field,
		Message:  message,
		Recovery: recovery,
	}
}
