package workflow

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/providers"
)

// ChapterLoader loads chapter and project data for workflow execution.
type ChapterLoader interface {
	LoadChapter(chapterID uint) (*database.Chapter, error)
	LoadProject(projectID uint) (*database.Project, error)
}

// DBChapterLoader uses the real database to load chapter and project data.
type DBChapterLoader struct{}

func (d *DBChapterLoader) LoadChapter(chapterID uint) (*database.Chapter, error) {
	return database.GetChapterByID(chapterID)
}

func (d *DBChapterLoader) LoadProject(projectID uint) (*database.Project, error) {
	return database.GetProjectByID(projectID)
}

// EventPublisher publishes workflow lifecycle events.
type EventPublisher interface {
	PublishWorkflowStarted(chapterID uint)
	PublishWorkflowStepChanged(chapterID uint, step string, status string)
	PublishWorkflowLog(chapterID uint, level string, message string)
	PublishWorkflowCompleted(chapterID uint, durationSec float64)
	PublishWorkflowFailed(chapterID uint, failedStep string, errorMessage string)
}

// RunRequest initiates a workflow run.
type RunRequest struct {
	ChapterID uint
}

// RunResult captures the outcome of a workflow run.
type RunResult struct {
	ChapterID uint
	Status    Status
}

// Executor runs chapter workflows.
type Executor struct {
	providers      providers.ProviderBundle
	storage        Storage
	eventPublisher EventPublisher
	chapterLoader  ChapterLoader
}

// NewExecutor creates a workflow executor.
func NewExecutor(providers providers.ProviderBundle, storage Storage) *Executor {
	return &Executor{
		providers:     providers,
		storage:       storage,
		chapterLoader: &DBChapterLoader{},
	}
}

// SetEventPublisher sets the event publisher for the executor.
func (e *Executor) SetEventPublisher(pub EventPublisher) {
	e.eventPublisher = pub
}

// SetChapterLoader sets the chapter loader for the executor (for testing).
func (e *Executor) SetChapterLoader(loader ChapterLoader) {
	e.chapterLoader = loader
}

// RunChapterWorkflow executes the full chapter workflow.
func (e *Executor) RunChapterWorkflow(ctx context.Context, req RunRequest) (RunResult, error) {
	startTime := time.Now()
	if e.eventPublisher != nil {
		e.eventPublisher.PublishWorkflowStarted(req.ChapterID)
	}

	// Load or initialize run state
	run, err := e.storage.LoadByChapterID(req.ChapterID)
	if err != nil {
		// New run
		now := time.Now()
		run = WorkflowRun{
			ChapterID:   req.ChapterID,
			CurrentStep: StepTTS,
			Status:      StatusRunning,
			Artifacts:   map[Step]ArtifactMetadata{},
			StartedAt:   &now,
		}
	} else if run.Status == StatusFailed {
		// Resume from failure
		run = PrepareResume(run)
		run.Status = StatusRunning
	}

	// Save initial state
	if err := e.storage.Save(run); err != nil {
		return RunResult{}, fmt.Errorf("save initial state: %w", err)
	}

	// Execute steps in order
	for _, step := range orderedSteps() {
		// Skip completed steps (from resume)
		if _, ok := run.Artifacts[step]; ok && run.CurrentStep != step {
			continue
		}

		run.CurrentStep = step
		if err := e.storage.Save(run); err != nil {
			return RunResult{}, fmt.Errorf("save step transition: %w", err)
		}
		if e.eventPublisher != nil {
			e.eventPublisher.PublishWorkflowStepChanged(req.ChapterID, string(step), string(StatusRunning))
		}

		// Execute step
		artifact, err := e.executeStep(ctx, step, run)
		if err != nil {
			now := time.Now()
			run.Status = StatusFailed
			run.ErrorCategory = categorizeError(err)
			run.ErrorMessage = err.Error()
			run.FinishedAt = &now
			_ = e.storage.Save(run)
			if e.eventPublisher != nil {
				e.eventPublisher.PublishWorkflowFailed(req.ChapterID, string(step), err.Error())
			}
			return RunResult{}, err
		}

		run.Artifacts[step] = artifact
		if err := e.storage.Save(run); err != nil {
			return RunResult{}, fmt.Errorf("save artifact: %w", err)
		}
	}

	// Mark succeeded
	now := time.Now()
	durationSec := now.Sub(startTime).Seconds()
	run.Status = StatusSucceeded
	run.FinishedAt = &now
	if err := e.storage.Save(run); err != nil {
		return RunResult{}, fmt.Errorf("save final state: %w", err)
	}
	if e.eventPublisher != nil {
		e.eventPublisher.PublishWorkflowCompleted(req.ChapterID, durationSec)
	}

	return RunResult{
		ChapterID: run.ChapterID,
		Status:    run.Status,
	}, nil
}

func (e *Executor) executeStep(ctx context.Context, step Step, run WorkflowRun) (ArtifactMetadata, error) {
	switch step {
	case StepTTS:
		return e.executeTTS(ctx, run)
	case StepSubtitle:
		return e.executeSubtitle(ctx, run)
	case StepImage:
		return e.executeImage(ctx, run)
	case StepProject:
		return e.executeProject(ctx, run)
	default:
		return nil, fmt.Errorf("unknown step: %s", step)
	}
}

func (e *Executor) executeTTS(ctx context.Context, run WorkflowRun) (ArtifactMetadata, error) {
	chapter, err := e.chapterLoader.LoadChapter(run.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("load chapter for TTS: %w", err)
	}

	req := providers.TTSRequest{
		Text:      chapter.Content,
		ProjectID: fmt.Sprintf("%d", chapter.ProjectID),
	}
	result, err := e.providers.TTS.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"audio_path": result.AudioPath,
		"duration":   result.Duration,
	}, nil
}

func (e *Executor) executeSubtitle(ctx context.Context, run WorkflowRun) (ArtifactMetadata, error) {
	chapter, err := e.chapterLoader.LoadChapter(run.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("load chapter for subtitle: %w", err)
	}

	ttsArtifact := run.Artifacts[StepTTS]
	audioPath, _ := ttsArtifact["audio_path"].(string)

	req := providers.SubtitleRequest{
		AudioPath: audioPath,
		Text:      chapter.Content,
	}
	result, err := e.providers.Subtitle.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"subtitle_path": result.SubtitlePath,
		"format":        result.Format,
	}, nil
}

func (e *Executor) executeImage(ctx context.Context, run WorkflowRun) (ArtifactMetadata, error) {
	chapter, err := e.chapterLoader.LoadChapter(run.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("load chapter for image: %w", err)
	}

	project, err := e.chapterLoader.LoadProject(chapter.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("load project for image: %w", err)
	}

	prompt := chapter.Prompt
	if prompt == "" {
		prompt = project.GlobalPrompt
	}

	req := providers.ImageRequest{
		Prompt: prompt,
	}
	result, err := e.providers.Image.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"image_paths": result.ImagePaths,
	}, nil
}

func (e *Executor) executeProject(ctx context.Context, run WorkflowRun) (ArtifactMetadata, error) {
	chapter, err := e.chapterLoader.LoadChapter(run.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("load chapter for project: %w", err)
	}

	project, err := e.chapterLoader.LoadProject(chapter.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("load project for project step: %w", err)
	}

	chapterDir := filepath.Join("projects", project.Name, fmt.Sprintf("chapter_%d", chapter.ID))

	req := providers.ProjectRequest{
		ChapterDir: chapterDir,
	}
	result, err := e.providers.Project.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"project_path": result.ProjectPath,
	}, nil
}

func categorizeError(err error) string {
	var providerErr providers.ProviderError
	if errors.As(err, &providerErr) {
		return string(providerErr.Category)
	}
	return "unknown"
}