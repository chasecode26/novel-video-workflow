package workflow

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/providers"

	"gorm.io/gorm"
)

func TestExecutor_RunChapterWorkflow_HappyPath(t *testing.T) {
	exec := newExecutorWithMocks(t)
	result, err := exec.RunChapterWorkflow(context.Background(), RunRequest{ChapterID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSucceeded {
		t.Fatalf("expected success, got %s", result.Status)
	}
}

func TestExecutor_PersistsStepTransitions(t *testing.T) {
	store := NewMemoryRunStorage()
	exec := newExecutorWithStore(t, store)

	_, err := exec.RunChapterWorkflow(context.Background(), RunRequest{ChapterID: 2})
	if err != nil {
		t.Fatal(err)
	}

	run, err := store.Load(2)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", run.Status)
	}
	if len(run.Artifacts) != 4 {
		t.Fatalf("expected 4 artifacts, got %d", len(run.Artifacts))
	}
}

func TestExecutor_RecordsArtifactMetadata(t *testing.T) {
	store := NewMemoryRunStorage()
	exec := newExecutorWithStore(t, store)

	_, err := exec.RunChapterWorkflow(context.Background(), RunRequest{ChapterID: 3})
	if err != nil {
		t.Fatal(err)
	}

	run, err := store.Load(3)
	if err != nil {
		t.Fatal(err)
	}

	ttsArtifact, ok := run.Artifacts[StepTTS]
	if !ok {
		t.Fatal("expected tts artifact")
	}
	if _, ok := ttsArtifact["audio_path"]; !ok {
		t.Fatal("expected audio_path in tts artifact")
	}
}

func TestExecutor_ReturnsCategorizedErrorOnProviderFailure(t *testing.T) {
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      failingTTSProvider{},
		Subtitle: stubSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, NewMemoryRunStorage(), "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())

	_, err := exec.RunChapterWorkflow(context.Background(), RunRequest{ChapterID: 4})
	if err == nil {
		t.Fatal("expected error")
	}

	var providerErr providers.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
}

func TestExecutor_PublishesLifecycleEventsOnSuccess(t *testing.T) {
	store := NewMemoryRunStorage()
	publisher := &recordingEventPublisher{}
	exec := newExecutorWithStoreAndPublisher(t, store, publisher)

	_, err := exec.RunChapterWorkflow(context.Background(), RunRequest{ChapterID: 6})
	if err != nil {
		t.Fatal(err)
	}

	eventTypes := publisher.eventTypes()
	if len(eventTypes) != 6 {
		t.Fatalf("expected 6 events, got %d: %#v", len(eventTypes), eventTypes)
	}
	expected := []string{"started", "step_changed", "step_changed", "step_changed", "step_changed", "completed"}
	for i, want := range expected {
		if eventTypes[i] != want {
			t.Fatalf("expected event %d to be %s, got %s", i, want, eventTypes[i])
		}
	}
}

func TestExecutor_PublishesFailureEventOnProviderFailure(t *testing.T) {
	publisher := &recordingEventPublisher{}
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      failingTTSProvider{},
		Subtitle: stubSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, NewMemoryRunStorage(), "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())
	exec.SetEventPublisher(publisher)

	_, err := exec.RunChapterWorkflow(context.Background(), RunRequest{ChapterID: 7})
	if err == nil {
		t.Fatal("expected error")
	}

	eventTypes := publisher.eventTypes()
	if len(eventTypes) != 3 {
		t.Fatalf("expected 3 events, got %d: %#v", len(eventTypes), eventTypes)
	}
	expected := []string{"started", "step_changed", "failed"}
	for i, want := range expected {
		if eventTypes[i] != want {
			t.Fatalf("expected event %d to be %s, got %s", i, want, eventTypes[i])
		}
	}
	if publisher.failedStep != string(StepTTS) {
		t.Fatalf("expected failed step %s, got %s", StepTTS, publisher.failedStep)
	}
}

func newExecutorWithMocks(t *testing.T) *Executor {
	t.Helper()
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      stubTTSProvider{},
		Subtitle: stubSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, NewMemoryRunStorage(), "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())
	return exec
}

func newExecutorWithStore(t *testing.T, storage Storage) *Executor {
	t.Helper()
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      stubTTSProvider{},
		Subtitle: stubSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())
	return exec
}

func newExecutorWithStoreAndPublisher(t *testing.T, storage Storage, pub EventPublisher) *Executor {
	t.Helper()
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      stubTTSProvider{},
		Subtitle: stubSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())
	exec.SetEventPublisher(pub)
	return exec
}

func newExecutorWithMockTTSAndFailingSubtitle(t *testing.T, storage Storage) *Executor {
	t.Helper()
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      stubTTSProvider{},
		Subtitle: failingSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())
	return exec
}

func newExecutorWithMocksAndStore(t *testing.T, storage Storage) *Executor {
	t.Helper()
	exec := NewExecutor(providers.ProviderBundle{
		TTS:      stubTTSProvider{},
		Subtitle: stubSubtitleProvider{},
		Image:    stubImageProvider{},
		Project:  stubProjectProvider{},
	}, storage, "/tmp/base", "/tmp/ref.wav")
	exec.SetChapterLoader(newMockChapterLoader())
	return exec
}

// Recording event publisher for testing
type recordingEventPublisher struct {
	events     []workflowEvent
	failedStep string
}

type workflowEvent struct {
	Type      string
	ChapterID uint
	Step      string
	Message   string
}

func (r *recordingEventPublisher) PublishWorkflowStarted(chapterID uint) {
	r.events = append(r.events, workflowEvent{Type: "started", ChapterID: chapterID})
}

func (r *recordingEventPublisher) PublishWorkflowStepChanged(chapterID uint, step string, status string) {
	r.events = append(r.events, workflowEvent{Type: "step_changed", ChapterID: chapterID, Step: step})
}

func (r *recordingEventPublisher) PublishWorkflowLog(chapterID uint, level string, message string) {
	r.events = append(r.events, workflowEvent{Type: "log", ChapterID: chapterID, Message: message})
}

func (r *recordingEventPublisher) PublishWorkflowCompleted(chapterID uint, durationSec float64) {
	r.events = append(r.events, workflowEvent{Type: "completed", ChapterID: chapterID})
}

func (r *recordingEventPublisher) PublishWorkflowFailed(chapterID uint, failedStep string, errorMessage string) {
	r.events = append(r.events, workflowEvent{Type: "failed", ChapterID: chapterID, Message: errorMessage})
	r.failedStep = failedStep
}

func (r *recordingEventPublisher) eventTypes() []string {
	types := make([]string, len(r.events))
	for i, e := range r.events {
		types[i] = e.Type
	}
	return types
}

// Stub providers

type stubTTSProvider struct{}

func (stubTTSProvider) Name() string { return "stub-tts" }
func (stubTTSProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "stub-tts", Severity: providers.SeverityInfo, Message: "ready"}
}
func (stubTTSProvider) Generate(req providers.TTSRequest) (providers.TTSResult, error) {
	return providers.TTSResult{AudioPath: "/tmp/audio.wav"}, nil
}

type stubSubtitleProvider struct{}

func (stubSubtitleProvider) Name() string { return "stub-subtitle" }
func (stubSubtitleProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "stub-subtitle", Severity: providers.SeverityInfo, Message: "ready"}
}
func (stubSubtitleProvider) Generate(req providers.SubtitleRequest) (providers.SubtitleResult, error) {
	return providers.SubtitleResult{SubtitlePath: "/tmp/subtitle.srt", Format: "srt"}, nil
}

type stubImageProvider struct{}

func (stubImageProvider) Name() string { return "stub-image" }
func (stubImageProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "stub-image", Severity: providers.SeverityInfo, Message: "ready"}
}
func (stubImageProvider) Generate(req providers.ImageRequest) (providers.ImageResult, error) {
	return providers.ImageResult{ImagePaths: []string{"/tmp/image.png"}}, nil
}

type stubProjectProvider struct{}

func (stubProjectProvider) Name() string { return "stub-project" }
func (stubProjectProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "stub-project", Severity: providers.SeverityInfo, Message: "ready"}
}
func (stubProjectProvider) Generate(req providers.ProjectRequest) (providers.ProjectResult, error) {
	return providers.ProjectResult{ProjectPath: "/tmp/project.json"}, nil
}

type failingTTSProvider struct{}

func (failingTTSProvider) Name() string { return "failing-tts" }
func (failingTTSProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "failing-tts", Severity: providers.SeverityInfo, Message: "ready"}
}
func (failingTTSProvider) Generate(req providers.TTSRequest) (providers.TTSResult, error) {
	return providers.TTSResult{}, providers.NewProviderError(providers.CategoryExecutionError, "tts failed", nil)
}

type failingSubtitleProvider struct{}

func (failingSubtitleProvider) Name() string { return "failing-subtitle" }
func (failingSubtitleProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "failing-subtitle", Severity: providers.SeverityInfo, Message: "ready"}
}
func (failingSubtitleProvider) Generate(req providers.SubtitleRequest) (providers.SubtitleResult, error) {
	return providers.SubtitleResult{}, providers.NewProviderError(providers.CategoryExecutionError, "subtitle failed", nil)
}

// Capturing providers for verifying executor inputs
type capturingTTSProvider struct {
	lastReq providers.TTSRequest
}

func (c *capturingTTSProvider) Name() string { return "capturing-tts" }
func (c *capturingTTSProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "capturing-tts", Severity: providers.SeverityInfo, Message: "ready"}
}
func (c *capturingTTSProvider) Generate(req providers.TTSRequest) (providers.TTSResult, error) {
	c.lastReq = req
	return providers.TTSResult{AudioPath: "/tmp/audio.wav"}, nil
}

type capturingImageProvider struct {
	lastReq providers.ImageRequest
}

func (c *capturingImageProvider) Name() string { return "capturing-image" }
func (c *capturingImageProvider) HealthCheck() providers.HealthCheckResult {
	return providers.HealthCheckResult{Provider: "capturing-image", Severity: providers.SeverityInfo, Message: "ready"}
}
func (c *capturingImageProvider) Generate(req providers.ImageRequest) (providers.ImageResult, error) {
	c.lastReq = req
	return providers.ImageResult{ImagePaths: []string{"/tmp/image.png"}}, nil
}

// Mock chapter loader for testing
type mockChapterLoader struct {
	chapters map[uint]*database.Chapter
	projects map[uint]*database.Project
}

func newMockChapterLoader() *mockChapterLoader {
	return &mockChapterLoader{
		chapters: map[uint]*database.Chapter{
			1: {Model: gorm.Model{ID: 1}, Title: "Chapter 1", Content: "Test content for chapter 1", ProjectID: 1, Prompt: "Suspense style"},
			2: {Model: gorm.Model{ID: 2}, Title: "Chapter 2", Content: "Test content for chapter 2", ProjectID: 1, Prompt: "Mystery style"},
			3: {Model: gorm.Model{ID: 3}, Title: "Chapter 3", Content: "Test content for chapter 3", ProjectID: 1, Prompt: ""},
			4: {Model: gorm.Model{ID: 4}, Title: "Chapter 4", Content: "Test content for chapter 4", ProjectID: 1, Prompt: "Drama style"},
			5: {Model: gorm.Model{ID: 5}, Title: "Chapter 5", Content: "Test content for chapter 5", ProjectID: 1, Prompt: "Action style"},
			6: {Model: gorm.Model{ID: 6}, Title: "Chapter 6", Content: "Test content for chapter 6", ProjectID: 1, Prompt: "Comedy style"},
			7: {Model: gorm.Model{ID: 7}, Title: "Chapter 7", Content: "Test content for chapter 7", ProjectID: 1, Prompt: "Horror style"},
		},
		projects: map[uint]*database.Project{
			1: {Model: gorm.Model{ID: 1}, Name: "Test Novel", GlobalPrompt: "Default suspense atmosphere"},
		},
	}
}

func (m *mockChapterLoader) LoadChapter(chapterID uint) (*database.Chapter, error) {
	if chapter, ok := m.chapters[chapterID]; ok {
		return chapter, nil
	}
	return nil, fmt.Errorf("chapter not found: %d", chapterID)
}

func (m *mockChapterLoader) LoadProject(projectID uint) (*database.Project, error) {
	if project, ok := m.projects[projectID]; ok {
		return project, nil
	}
	return nil, fmt.Errorf("project not found: %d", projectID)
}
