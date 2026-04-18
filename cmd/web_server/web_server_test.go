package web_server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"novel-video-workflow/pkg/api"
	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/mcp"
	"novel-video-workflow/pkg/workflow"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

	systemAPI, runAPI := buildWorkflowAPIs(cfg, bundle)
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

func TestRegisterProcessorBackedRoutes_SkipsWhenMCPServerIsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	originalServer := mcpServerInstance
	mcpServerInstance = nil
	t.Cleanup(func() {
		mcpServerInstance = originalServer
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected nil MCP server to avoid panic, got %v", r)
		}
	}()

	registered := registerProcessorBackedRoutes(router)
	if registered {
		t.Fatal("expected processor-backed routes to be skipped when MCP server is nil")
	}

	assertRouteStatus(t, router, http.MethodGet, "/api/prompt-templates", http.StatusServiceUnavailable)
}

func TestRegisterProcessorBackedRoutes_SkipsWhenProcessorIsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	server, err := mcp.NewServer(nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new mcp server: %v", err)
	}

	originalServer := mcpServerInstance
	mcpServerInstance = server
	t.Cleanup(func() {
		mcpServerInstance = originalServer
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected nil processor to avoid panic, got %v", r)
		}
	}()

	registered := registerProcessorBackedRoutes(router)
	if registered {
		t.Fatal("expected processor-backed routes to be skipped when processor is nil")
	}

	assertRouteStatus(t, router, http.MethodGet, "/api/prompt-templates", http.StatusServiceUnavailable)
	assertRouteStatus(t, router, http.MethodGet, "/api/workflow/chapter/1/params", http.StatusServiceUnavailable)
}

func TestRegisterProcessorBackedRoutes_RegistersWhenMCPServerIsAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	cfg, bundle, processor, err := loadAppComponents(filepath.Join("..", "..", "pkg", "config", "testdata", "config-minimal.yaml"))
	if err != nil {
		t.Fatalf("load app components: %v", err)
	}
	if cfg.Database.Path == "" || processor == nil {
		t.Fatal("expected processor-backed config for route registration")
	}

	server, err := mcp.NewServer(processor, zap.NewNop())
	if err != nil {
		t.Fatalf("new mcp server: %v", err)
	}

	originalServer := mcpServerInstance
	mcpServerInstance = server
	t.Cleanup(func() {
		mcpServerInstance = originalServer
	})

	registered := registerProcessorBackedRoutes(router)
	if !registered {
		t.Fatal("expected processor-backed routes to be registered")
	}

	assertRouteStatus(t, router, http.MethodGet, "/api/prompt-templates", http.StatusOK)
	assertRouteStatus(t, router, http.MethodGet, "/api/workflow/chapter/not-a-number/params", http.StatusBadRequest)
	assertRouteStatus(t, router, http.MethodGet, "/api/workflow/chapter/not-a-number/params", http.StatusBadRequest)

	_ = bundle
}

func TestRegisterWorkflowRoutes_RegistersSystemAndRunEndpointsWithoutProcessor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	cfg, bundle, _, err := loadAppComponents(filepath.Join("..", "..", "pkg", "config", "testdata", "config-minimal.yaml"))
	if err != nil {
		t.Fatalf("load app components: %v", err)
	}

	registered := registerWorkflowRoutes(router, cfg, bundle)
	if !registered {
		t.Fatal("expected workflow routes to register")
	}

	assertRouteStatus(t, router, http.MethodGet, "/api/system/check", http.StatusOK)
	assertRouteStatus(t, router, http.MethodGet, "/api/workflow/runs/not-a-number", http.StatusBadRequest)
}

func TestFileContentHandler_AllowsInputPreviewWithRelativePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rootDir := t.TempDir()
	inputDir := filepath.Join(rootDir, "input")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatalf("create input dir: %v", err)
	}

	filePath := filepath.Join(inputDir, "chapter.txt")
	const fileContent = "preview me"
	if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("write preview file: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir to temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/files/content?path=.%2Finput%2Fchapter.txt", nil)
	w := httptest.NewRecorder()
	c := gin.CreateTestContextOnly(w, gin.New())
	c.Request = req

	fileContentHandler(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected preview request to return status %d, got %d with body %s", http.StatusOK, w.Code, w.Body.String())
	}
	if w.Body.String() != fileContent {
		t.Fatalf("expected preview body %q, got %q", fileContent, w.Body.String())
	}
}

func TestExtractToolParams_PrefersNestedParamsMap(t *testing.T) {
	reqBody := map[string]interface{}{
		"toolName": "file_split_novel_into_chapters",
		"params": map[string]interface{}{
			"novel_path": "./input/book.txt",
		},
		"novel_path": "./input/ignored.txt",
	}

	params := extractToolParams(reqBody)
	if got := params["novel_path"]; got != "./input/book.txt" {
		t.Fatalf("expected nested params to win, got %v", got)
	}
}

func TestExtractToolParams_FallsBackToTopLevelFields(t *testing.T) {
	reqBody := map[string]interface{}{
		"toolName":    "generate_image_from_text",
		"text":        "scene description",
		"output_file": "./output/image.png",
		"width":       512.0,
	}

	params := extractToolParams(reqBody)
	if got := params["text"]; got != "scene description" {
		t.Fatalf("expected top-level text to be preserved, got %v", got)
	}
	if got := params["output_file"]; got != "./output/image.png" {
		t.Fatalf("expected top-level output_file to be preserved, got %v", got)
	}
	if _, exists := params["toolName"]; exists {
		t.Fatal("expected toolName to be excluded from params")
	}
}

func assertRouteStatus(t *testing.T, router *gin.Engine, method, path string, wantStatus int) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != wantStatus {
		t.Fatalf("expected %s %s to return status %d, got %d", method, path, wantStatus, w.Code)
	}
}
