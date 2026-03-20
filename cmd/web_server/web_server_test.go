package web_server

import (
	"path/filepath"
	"testing"

	"novel-video-workflow/pkg/api"
	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/workflow"
)

func TestLoadAppComponents_BuildsProviderBackedProcessor(t *testing.T) {
	cfg, bundle, processor, err := loadAppComponents(filepath.Join("..", "..", "pkg", "config", "testdata", "config-minimal.yaml"))
	if err != nil {
		t.Fatalf("load app components: %v", err)
	}
	if cfg.Database.Path == "" {
		t.Fatal("expected database path from config")
	}
	if bundle.TTS == nil || bundle.Subtitle == nil || bundle.Image == nil || bundle.Project == nil {
		t.Fatal("expected all providers to be initialized")
	}
	if processor == nil {
		t.Fatal("expected processor")
	}
}

func TestBuildWorkflowAPIs_WiresSystemAndRunAPIs(t *testing.T) {
	cfg, bundle, _, err := loadAppComponents(filepath.Join("..", "..", "pkg", "config", "testdata", "config-minimal.yaml"))
	if err != nil {
		t.Fatalf("load app components: %v", err)
	}
	if cfg.Database.Path == "" {
		t.Fatal("expected database path from config")
	}

	systemAPI, runAPI := buildWorkflowAPIs(bundle)
	if systemAPI == nil {
		t.Fatal("expected system check api")
	}
	if runAPI == nil {
		t.Fatal("expected workflow run api")
	}

	_ = database.DB
	_ = api.NewSystemCheckAPI
	_ = workflow.NewRunStorage
}
