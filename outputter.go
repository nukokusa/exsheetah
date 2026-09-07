package exsheetah

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Outputter interface {
	Output(ctx context.Context, path string, sheets ...*Sheet) error
}

type YAMLOutputter struct{}

func (o *YAMLOutputter) Output(ctx context.Context, path string, sheets ...*Sheet) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	for _, s := range sheets {
		buf := bytes.NewBuffer(nil)
		if err := yaml.NewEncoder(buf, yaml.IndentSequence(true)).Encode(s); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(path, fmt.Sprintf("%s.yaml", s.Name())), buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

type JSONOutputter struct{}

func (o *JSONOutputter) Output(ctx context.Context, path string, sheets ...*Sheet) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	for _, s := range sheets {
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(s); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(path, fmt.Sprintf("%s.json", s.Name())), buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}
