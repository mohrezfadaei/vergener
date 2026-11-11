package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSchemaValidJSON(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(Schema), &schema); err != nil {
		t.Fatal(err)
	}
	if schema["$schema"] == nil || schema["$defs"] == nil {
		t.Fatal("incomplete JSON schema")
	}
	_, file, _, _ := runtime.Caller(0)
	checkedIn, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../..", "docs", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(checkedIn)) != strings.TrimSpace(Schema) {
		t.Fatal("docs/schema.json differs from config --schema output")
	}
}
