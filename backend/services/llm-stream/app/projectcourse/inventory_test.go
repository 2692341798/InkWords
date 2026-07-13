package projectcourse

import (
	"reflect"
	"testing"
)

func TestBuildInventoryIsDeterministicAcrossRepeatedRuns(t *testing.T) {
	inputs := []InventoryInput{
		{Path: "README.md", Content: []byte("run it")},
		{Path: "backend/services/core-api/domain/task/service.go", Content: []byte("package task")},
		{Path: "backend/services/core-api/transport/http/v1/routes.go", Content: []byte("package v1")},
		{Path: "frontend/src/hooks/generator/useProjectAnalyzer.ts", Content: []byte("export function analyze() {}")},
		{Path: "backend/services/core-api/domain/task/service_test.go", Content: []byte("package task")},
		{Path: "docs/runbook.md", Content: []byte("check")},
	}
	want, err := BuildInventory(inputs, InventoryOptions{MaxContentBytes: 1000})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		got, err := BuildInventory(inputs, InventoryOptions{MaxContentBytes: 1000})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("inventory changed on run %d: %#v != %#v", i, got, want)
		}
	}
}

func TestBuildInventoryClassifiesEvidenceFriendlyFiles(t *testing.T) {
	entries, err := BuildInventory([]InventoryInput{
		{Path: "docs/architecture.md", Content: []byte("docs")},
		{Path: "scripts/test_reader.py", Content: []byte("print(1)")},
		{Path: "docker-compose.yml", Content: []byte("services: {}")},
		{Path: "assets/logo.png", Content: []byte("binary")},
	}, InventoryOptions{MaxContentBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	byPath := make(map[string]InventoryEntry, len(entries))
	for _, entry := range entries {
		byPath[entry.Path] = entry
	}
	if byPath["docs/architecture.md"].Disposition != DispositionIndexed || byPath["scripts/test_reader.py"].Disposition != DispositionIndexed {
		t.Fatal("documentation and scripts must remain indexed")
	}
	if byPath["assets/logo.png"].Disposition != DispositionExcluded {
		t.Fatal("binary must be excluded with a reason")
	}
	if byPath["assets/logo.png"].Reason == "" {
		t.Fatal("excluded file must carry a machine-readable reason")
	}
}

func TestBuildInventoryProtectsPathsAndDelaysLargeContent(t *testing.T) {
	if _, err := BuildInventory([]InventoryInput{{Path: "../secret", Content: []byte("x")}}, InventoryOptions{}); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	entries, err := BuildInventory([]InventoryInput{{Path: "main.go", Content: []byte("package main")}}, InventoryOptions{MaxContentBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Content != "" || entries[0].ContentHash == "" {
		t.Fatal("large content should keep metadata and hash while delaying body")
	}
}
