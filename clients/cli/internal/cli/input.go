package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadFile reads bytes from path or stdin when path is "-".
func ReadFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

// ReadJSONFile parses a JSON or YAML file into a map.
func ReadJSONFile(path string) (map[string]any, error) {
	data, err := ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseObject(data, filepath.Ext(path))
}

// ParseObject unmarshals JSON or YAML object bytes.
func ParseObject(data []byte, ext string) (map[string]any, error) {
	ext = strings.ToLower(ext)
	if ext == ".yaml" || ext == ".yml" {
		var out map[string]any
		if err := yaml.Unmarshal(data, &out); err != nil {
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}
		return out, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return out, nil
}

// ReadTextFile reads a text file for --body/--file content flags.
func ReadTextFile(path string) (string, error) {
	b, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}