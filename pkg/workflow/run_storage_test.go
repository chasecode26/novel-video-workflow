package workflow

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"novel-video-workflow/pkg/database"
)

func initWorkflowTestDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "workflow.db")
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
	return dbPath
}

func TestRunStorage_SaveAndLoadRoundTrip(t *testing.T) {
	initWorkflowTestDB(t)

	project, err := database.CreateProject("storage-project", "desc", "prompt", "secret")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	chapter, err := database.CreateChapter(project.ID, "chapter-1", "content", "prompt")
	if err != nil {
		t.Fatalf("create chapter: %v", err)
	}

	storage := NewRunStorage(database.DB)
	run := WorkflowRun{
		ChapterID:   chapter.ID,
		CurrentStep: StepSubtitle,
		Status:      StatusRunning,
		Artifacts: map[Step]ArtifactMetadata{
			StepTTS: {"audio_path": "chapter.wav"},
		},
		ErrorCategory: "provider_error",
		ErrorMessage:  "subtitle timeout",
	}

	if err := storage.Save(run); err != nil {
		t.Fatalf("save run: %v", err)
	}

	loaded, err := storage.LoadByChapterID(chapter.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}

	if loaded.CurrentStep != StepSubtitle {
		t.Fatalf("expected current step %q, got %q", StepSubtitle, loaded.CurrentStep)
	}
	if loaded.Status != StatusRunning {
		t.Fatalf("expected status %q, got %q", StatusRunning, loaded.Status)
	}
	if got := loaded.Artifacts[StepTTS]["audio_path"]; got != "chapter.wav" {
		t.Fatalf("expected artifact metadata to round-trip, got %#v", got)
	}
	if loaded.ErrorCategory != "provider_error" {
		t.Fatalf("expected error category to round-trip, got %q", loaded.ErrorCategory)
	}
}

func TestRunStorage_SaveOverwritesExistingRun(t *testing.T) {
	initWorkflowTestDB(t)

	project, err := database.CreateProject("overwrite-project", "desc", "prompt", "secret")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	chapter, err := database.CreateChapter(project.ID, "chapter-1", "content", "prompt")
	if err != nil {
		t.Fatalf("create chapter: %v", err)
	}

	storage := NewRunStorage(database.DB)
	failed := WorkflowRun{
		ChapterID:   chapter.ID,
		CurrentStep: StepImage,
		Status:      StatusFailed,
		Artifacts: map[Step]ArtifactMetadata{
			StepTTS:      {"audio_path": "chapter.wav"},
			StepSubtitle: {"subtitle_path": "chapter.srt"},
		},
		ErrorCategory: "provider_error",
		ErrorMessage:  "image failed",
	}
	if err := storage.Save(failed); err != nil {
		t.Fatalf("save failed run: %v", err)
	}

	resumed := PrepareResume(failed)
	resumed.Status = StatusSucceeded
	resumed.CurrentStep = StepProject
	resumed.Artifacts[StepImage] = ArtifactMetadata{"image_paths": []string{"scene-1.png"}}
	resumed.Artifacts[StepProject] = ArtifactMetadata{"project_path": "draft_content.json"}
	if err := storage.Save(resumed); err != nil {
		t.Fatalf("save resumed run: %v", err)
	}

	loaded, err := storage.LoadByChapterID(chapter.ID)
	if err != nil {
		t.Fatalf("load resumed run: %v", err)
	}
	if loaded.Status != StatusSucceeded {
		t.Fatalf("expected final status %q, got %q", StatusSucceeded, loaded.Status)
	}
	if loaded.ErrorCategory != "" || loaded.ErrorMessage != "" {
		t.Fatalf("expected cleared errors after success, got %q / %q", loaded.ErrorCategory, loaded.ErrorMessage)
	}

	var records []database.WorkflowRun
	if err := database.DB.Find(&records).Error; err != nil {
		t.Fatalf("query workflow runs: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected single workflow run row, got %d", len(records))
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(records[0].ArtifactMetadata), &payload); err != nil {
		t.Fatalf("unmarshal stored metadata: %v", err)
	}
	if _, ok := payload[string(StepProject)]; !ok {
		t.Fatal("expected stored metadata json to contain project artifact")
	}
}
