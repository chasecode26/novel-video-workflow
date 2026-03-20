package main

import (
	"path/filepath"
	"testing"

	configpkg "novel-video-workflow/pkg/config"
)

func TestLoadBridgeConfig_LoadsTypedConfig(t *testing.T) {
	cfg, err := loadBridgeConfig(filepath.Join("..", "..", "pkg", "config", "testdata", "config-minimal.yaml"))
	if err != nil {
		t.Fatalf("load bridge config: %v", err)
	}
	if cfg == (configpkg.Config{}) {
		t.Fatal("expected non-zero config")
	}
	if cfg.Database.Path == "" {
		t.Fatal("expected database path from config")
	}
}
