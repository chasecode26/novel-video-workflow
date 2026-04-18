package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/providers"
	"novel-video-workflow/pkg/workflow"

	"github.com/gin-gonic/gin"
)

func initAPITestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "api.db")
	if err := database.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	sqlDB, err := database.DB.DB()
	if err != nil {
		t.Fatalf("DB.DB: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		database.DB = nil
	})
}

func createTestChapter(t *testing.T, chapterID uint) uint {
	t.Helper()
	project, err := database.CreateProject("project-"+strconv.Itoa(int(chapterID)), "desc", "prompt", "secret")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	chapter, err := database.CreateChapter(project.ID, "chapter", "content", "prompt")
	if err != nil {
		t.Fatalf("create chapter: %v", err)
	}
	return chapter.ID
}

func TestSystemCheckAPI_ReturnsProviderHealthChecks(t *testing.T) {
	router := gin.New()
	bundle := providers.ProviderBundle{
		TTS:      mockTTSProviderForSystemCheck{},
		Subtitle: mockSubtitleProviderForSystemCheck{},
		Image:    mockImageProviderForSystemCheck{},
		Project:  mockProjectProviderForSystemCheck{},
	}
	api := NewSystemCheckAPI(bundle)
	api.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/system/check", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Status != "success" {
		t.Fatalf("expected status success, got %s", resp.Status)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	results, ok := data["results"].([]interface{})
	if !ok {
		t.Fatal("expected results array")
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 provider results, got %d", len(results))
	}
}

func TestWorkflowRunAPI_StartsNewRunWithRealRunResource(t *testing.T) {
	initAPITestDB(t)
	_ = createTestChapter(t, 1)
	chapterID := createTestChapter(t, 2)

	router := gin.New()
	storage := workflow.NewRunStorage(database.DB)
	exec := workflow.NewExecutor(providers.ProviderBundle{
		TTS:      mockTTSProviderForRun{},
		Subtitle: mockSubtitleProviderForRun{},
		Image:    mockImageProviderForRun{},
		Project:  mockProjectProviderForRun{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	api := NewWorkflowRunAPI(exec, storage)
	api.RegisterRoutes(router)

	body := `{"chapter_id": ` + strconv.Itoa(int(chapterID)) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/workflow/runs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var resp SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Status != "success" {
		t.Fatalf("expected status success, got %s", resp.Status)
	}

	payload, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map payload, got %T", resp.Data)
	}
	if payload["id"] == float64(0) {
		t.Fatal("expected non-zero run id")
	}
	if payload["chapter_id"] != float64(chapterID) {
		t.Fatalf("expected chapter_id %d, got %#v", chapterID, payload["chapter_id"])
	}
	if payload["id"] == float64(chapterID) {
		t.Fatalf("expected run id distinct from chapter id %d", chapterID)
	}
	if _, err := storage.LoadByID(uint(payload["id"].(float64))); err != nil {
		t.Fatalf("expected persisted run for returned id: %v", err)
	}
}

func TestWorkflowRunAPI_ReusesExistingActiveRunForChapter(t *testing.T) {
	initAPITestDB(t)
	chapterID := createTestChapter(t, 3)

	router := gin.New()
	storage := workflow.NewRunStorage(database.DB)
	exec := workflow.NewExecutor(providers.ProviderBundle{
		TTS:      blockingTTSProviderForRun{},
		Subtitle: mockSubtitleProviderForRun{},
		Image:    mockImageProviderForRun{},
		Project:  mockProjectProviderForRun{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	api := NewWorkflowRunAPI(exec, storage)
	api.RegisterRoutes(router)

	firstReq := httptest.NewRequest(http.MethodPost, "/api/workflow/runs", strings.NewReader(`{"chapter_id": `+strconv.Itoa(int(chapterID))+`}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusCreated {
		t.Fatalf("expected first status 201, got %d", firstResp.Code)
	}

	var first SuccessResponse
	if err := json.Unmarshal(firstResp.Body.Bytes(), &first); err != nil {
		t.Fatalf("unmarshal first response: %v", err)
	}
	firstPayload, ok := first.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected first payload map, got %T", first.Data)
	}
	firstRunID := uint(firstPayload["id"].(float64))

	secondReq := httptest.NewRequest(http.MethodPost, "/api/workflow/runs", strings.NewReader(`{"chapter_id": `+strconv.Itoa(int(chapterID))+`}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusCreated {
		t.Fatalf("expected second status 201, got %d", secondResp.Code)
	}

	var second SuccessResponse
	if err := json.Unmarshal(secondResp.Body.Bytes(), &second); err != nil {
		t.Fatalf("unmarshal second response: %v", err)
	}
	secondPayload, ok := second.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected second payload map, got %T", second.Data)
	}
	secondRunID := uint(secondPayload["id"].(float64))

	if secondRunID != firstRunID {
		t.Fatalf("expected same active run id %d, got %d", firstRunID, secondRunID)
	}
}

func TestWorkflowRunAPI_GetsRunByRunID(t *testing.T) {
	initAPITestDB(t)
	chapterID := createTestChapter(t, 2)

	router := gin.New()
	storage := workflow.NewRunStorage(database.DB)
	exec := workflow.NewExecutor(providers.ProviderBundle{
		TTS:      mockTTSProviderForRun{},
		Subtitle: mockSubtitleProviderForRun{},
		Image:    mockImageProviderForRun{},
		Project:  mockProjectProviderForRun{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	api := NewWorkflowRunAPI(exec, storage)
	api.RegisterRoutes(router)

	_, _ = exec.RunChapterWorkflow(context.Background(), workflow.RunRequest{ChapterID: chapterID})

	req := httptest.NewRequest(http.MethodGet, "/api/workflow/runs/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp WorkflowRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.ID == 0 {
		t.Fatal("expected non-zero run id")
	}
	if resp.ChapterID != chapterID {
		t.Fatalf("expected chapter_id %d, got %d", chapterID, resp.ChapterID)
	}
}

func TestWorkflowRunAPI_ReturnsNotFoundForMissingChapter(t *testing.T) {
	initAPITestDB(t)

	router := gin.New()
	storage := workflow.NewRunStorage(database.DB)
	exec := workflow.NewExecutor(providers.ProviderBundle{
		TTS:      mockTTSProviderForRun{},
		Subtitle: mockSubtitleProviderForRun{},
		Image:    mockImageProviderForRun{},
		Project:  mockProjectProviderForRun{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	api := NewWorkflowRunAPI(exec, storage)
	api.RegisterRoutes(router)

	body := `{"chapter_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/api/workflow/runs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

type mockTTSProviderForSystemCheck struct{}

func (mockTTSProviderForSystemCheck) Name() string { return "mock-tts" }
func (mockTTSProviderForSystemCheck) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-tts", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockTTSProviderForSystemCheck) Generate(req providers.TTSRequest) (providers.TTSResult, error) {
	return providers.TTSResult{AudioPath: "/tmp/audio.wav"}, nil
}

type mockSubtitleProviderForSystemCheck struct{}

func (mockSubtitleProviderForSystemCheck) Name() string { return "mock-subtitle" }
func (mockSubtitleProviderForSystemCheck) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-subtitle", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockSubtitleProviderForSystemCheck) Generate(req providers.SubtitleRequest) (providers.SubtitleResult, error) {
	return providers.SubtitleResult{SubtitlePath: "/tmp/subtitle.srt"}, nil
}

type mockImageProviderForSystemCheck struct{}

func (mockImageProviderForSystemCheck) Name() string { return "mock-image" }
func (mockImageProviderForSystemCheck) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-image", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockImageProviderForSystemCheck) Generate(req providers.ImageRequest) (providers.ImageResult, error) {
	return providers.ImageResult{ImagePaths: []string{"/tmp/image.png"}}, nil
}

type mockProjectProviderForSystemCheck struct{}

func (mockProjectProviderForSystemCheck) Name() string { return "mock-project" }
func (mockProjectProviderForSystemCheck) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-project", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockProjectProviderForSystemCheck) Generate(req providers.ProjectRequest) (providers.ProjectResult, error) {
	return providers.ProjectResult{ProjectPath: "/tmp/project.json"}, nil
}

// Mock providers for run tests (same as above, just renamed for clarity)
type mockTTSProviderForRun struct{}

func (mockTTSProviderForRun) Name() string { return "mock-tts" }
func (mockTTSProviderForRun) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-tts", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockTTSProviderForRun) Generate(req providers.TTSRequest) (providers.TTSResult, error) {
	return providers.TTSResult{AudioPath: "/tmp/audio.wav"}, nil
}

type blockingTTSProviderForRun struct{}

func (blockingTTSProviderForRun) Name() string { return "blocking-tts" }
func (blockingTTSProviderForRun) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "blocking-tts", Severity: providers.SeverityInfo, Message: "ready"}
}
func (blockingTTSProviderForRun) Generate(req providers.TTSRequest) (providers.TTSResult, error) {
	select {}
}

type mockSubtitleProviderForRun struct{}

func (mockSubtitleProviderForRun) Name() string { return "mock-subtitle" }
func (mockSubtitleProviderForRun) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-subtitle", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockSubtitleProviderForRun) Generate(req providers.SubtitleRequest) (providers.SubtitleResult, error) {
	return providers.SubtitleResult{SubtitlePath: "/tmp/subtitle.srt"}, nil
}

type mockImageProviderForRun struct{}

func (mockImageProviderForRun) Name() string { return "mock-image" }
func (mockImageProviderForRun) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-image", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockImageProviderForRun) Generate(req providers.ImageRequest) (providers.ImageResult, error) {
	return providers.ImageResult{ImagePaths: []string{"/tmp/image.png"}}, nil
}

type mockProjectProviderForRun struct{}

func (mockProjectProviderForRun) Name() string { return "mock-project" }
func (mockProjectProviderForRun) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "mock-project", Severity: providers.SeverityInfo, Message: "ready"}
}
func (mockProjectProviderForRun) Generate(req providers.ProjectRequest) (providers.ProjectResult, error) {
	return providers.ProjectResult{ProjectPath: "/tmp/project.json"}, nil
}
