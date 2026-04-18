package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
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
	baseDir        string
	referenceAudio string
}

// NewExecutor creates a workflow executor.
func NewExecutor(providers providers.ProviderBundle, storage Storage, baseDir string, referenceAudio string) *Executor {
	return &Executor{
		providers:      providers,
		storage:        storage,
		chapterLoader:  &DBChapterLoader{},
		baseDir:        strings.TrimSpace(baseDir),
		referenceAudio: strings.TrimSpace(referenceAudio),
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

	project, err := e.chapterLoader.LoadProject(chapter.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("load project for TTS: %w", err)
	}

	workflowParams := parseWorkflowParams(chapter.WorkflowParams)
	paths := e.buildChapterPaths(project, chapter, workflowParams)
	chapterNumber := workflowChapterNumber(chapter)

	req := providers.TTSRequest{
		ProjectID:      strconv.FormatUint(uint64(project.ID), 10),
		ChapterNumber:  chapterNumber,
		Text:           chapter.Content,
		ReferenceAudio: firstNonEmptyString(workflowString(workflowParams, "reference_audio"), workflowString(workflowParams, "tts.reference_audio"), e.referenceAudio),
		OutputDir:      paths.AudioPath,
	}
	result, err := e.providers.TTS.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"chapter_dir":     paths.ChapterDir,
		"audio_path":      result.AudioPath,
		"duration":        result.Duration,
		"reference_audio": req.ReferenceAudio,
		"source_dir":      paths.SourceDir,
		"chapter_number":  chapterNumber,
		"project_id":      req.ProjectID,
		"project_slug":    paths.ProjectSlug,
		"workflow_params": cloneStringMap(workflowParams),
	}, nil
}

func (e *Executor) executeSubtitle(ctx context.Context, run WorkflowRun) (ArtifactMetadata, error) {
	chapter, err := e.chapterLoader.LoadChapter(run.ChapterID)
	if err != nil {
		return nil, fmt.Errorf("load chapter for subtitle: %w", err)
	}

	project, err := e.chapterLoader.LoadProject(chapter.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("load project for subtitle: %w", err)
	}

	workflowParams := parseWorkflowParams(chapter.WorkflowParams)
	paths := e.buildChapterPaths(project, chapter, workflowParams)
	chapterNumber := workflowChapterNumber(chapter)

	ttsArtifact := run.Artifacts[StepTTS]
	audioPath, _ := ttsArtifact["audio_path"].(string)

	req := providers.SubtitleRequest{
		ProjectID:     strconv.FormatUint(uint64(project.ID), 10),
		ChapterNumber: chapterNumber,
		AudioPath:     audioPath,
		Text:          chapter.Content,
		OutputPath:    paths.SubtitlePath,
	}
	result, err := e.providers.Subtitle.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"subtitle_path": result.SubtitlePath,
		"format":        result.Format,
		"chapter_dir":   paths.ChapterDir,
		"audio_path":    audioPath,
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

	workflowParams := parseWorkflowParams(chapter.WorkflowParams)
	paths := e.buildChapterPaths(project, chapter, workflowParams)
	prompt := chapter.Prompt
	if prompt == "" {
		prompt = project.GlobalPrompt
	}
	chapterNumber := workflowChapterNumber(chapter)

	req := providers.ImageRequest{
		ProjectID:     strconv.FormatUint(uint64(project.ID), 10),
		ChapterNumber: chapterNumber,
		Prompt:        prompt,
		OutputDir:     paths.ImagesDir,
		Count:         workflowImageCount(workflowParams),
	}
	result, err := e.providers.Image.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"image_paths": result.ImagePaths,
		"chapter_dir": paths.ChapterDir,
		"prompt":      prompt,
		"image_count": len(result.ImagePaths),
		"images_dir":  paths.ImagesDir,
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

	workflowParams := parseWorkflowParams(chapter.WorkflowParams)
	paths := e.buildChapterPaths(project, chapter, workflowParams)

	ttsArtifact := run.Artifacts[StepTTS]
	audioPath, _ := ttsArtifact["audio_path"].(string)
	subtitleArtifact := run.Artifacts[StepSubtitle]
	subtitlePath, _ := subtitleArtifact["subtitle_path"].(string)
	imageArtifact := run.Artifacts[StepImage]
	imagePaths := metadataStringSlice(imageArtifact["image_paths"])

	req := providers.ProjectRequest{
		ProjectID:    strconv.FormatUint(uint64(project.ID), 10),
		ChapterDir:   paths.ChapterDir,
		AudioPath:    audioPath,
		SubtitlePath: subtitlePath,
		ImagePaths:   imagePaths,
	}
	result, err := e.providers.Project.Generate(req)
	if err != nil {
		return nil, err
	}
	return ArtifactMetadata{
		"project_path":   result.ProjectPath,
		"edit_list_path": result.EditListPath,
		"chapter_dir":    paths.ChapterDir,
		"audio_path":     audioPath,
		"subtitle_path":  subtitlePath,
		"image_paths":    imagePaths,
	}, nil
}

type chapterPaths struct {
	ProjectSlug  string
	SourceDir    string
	ChapterDir   string
	AudioPath    string
	SubtitlePath string
	ImagesDir    string
}

func (e *Executor) buildChapterPaths(project *database.Project, chapter *database.Chapter, workflowParams map[string]interface{}) chapterPaths {
	projectSlug := sanitizePathSegment(project.Name)
	if projectSlug == "" {
		projectSlug = strconv.FormatUint(uint64(project.ID), 10)
	}

	sourceDir := strings.TrimSpace(workflowString(workflowParams, "source_dir"))
	chapterDirName := firstNonEmptyString(strings.TrimSpace(workflowString(workflowParams, "chapter_dir")), fmt.Sprintf("chapter_%02d", workflowChapterNumber(chapter)))
	chapterDir := filepath.Join(e.baseOutputDir(), projectSlug, chapterDirName)

	return chapterPaths{
		ProjectSlug:  projectSlug,
		SourceDir:    sourceDir,
		ChapterDir:   chapterDir,
		AudioPath:    filepath.Join(chapterDir, fmt.Sprintf("%s.wav", chapterDirName)),
		SubtitlePath: filepath.Join(chapterDir, fmt.Sprintf("%s.srt", chapterDirName)),
		ImagesDir:    chapterDir,
	}
}

func (e *Executor) baseOutputDir() string {
	if strings.TrimSpace(e.baseDir) == "" {
		return filepath.Join("output")
	}
	return filepath.Join(e.baseDir, "output")
}

func workflowChapterNumber(chapter *database.Chapter) int {
	if chapter == nil {
		return 0
	}
	if chapter.ID > 0 {
		return int(chapter.ID)
	}
	return 0
}

func parseWorkflowParams(raw string) map[string]interface{} {
	params := map[string]interface{}{}
	if strings.TrimSpace(raw) == "" {
		return params
	}
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return map[string]interface{}{}
	}
	return params
}

func workflowImageCount(params map[string]interface{}) int {
	candidates := []string{"image_count", "generated_image_count", "max_images", "image.count", "image.max_images"}
	for _, key := range candidates {
		if value, ok := workflowInt(params, key); ok && value > 0 {
			return value
		}
	}
	return 1
}

func workflowString(params map[string]interface{}, key string) string {
	if len(params) == 0 {
		return ""
	}
	segments := strings.Split(key, ".")
	current := params
	for index, segment := range segments {
		value, ok := current[segment]
		if !ok {
			return ""
		}
		if index == len(segments)-1 {
			switch typed := value.(type) {
			case string:
				return strings.TrimSpace(typed)
			case fmt.Stringer:
				return strings.TrimSpace(typed.String())
			default:
				return strings.TrimSpace(fmt.Sprint(value))
			}
		}
		next, ok := value.(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func workflowInt(params map[string]interface{}, key string) (int, bool) {
	if len(params) == 0 {
		return 0, false
	}
	segments := strings.Split(key, ".")
	current := params
	for index, segment := range segments {
		value, ok := current[segment]
		if !ok {
			return 0, false
		}
		if index == len(segments)-1 {
			switch typed := value.(type) {
			case int:
				return typed, true
			case int32:
				return int(typed), true
			case int64:
				return int(typed), true
			case float64:
				return int(typed), true
			case json.Number:
				number, err := typed.Int64()
				return int(number), err == nil
			case string:
				number, err := strconv.Atoi(strings.TrimSpace(typed))
				return number, err == nil
			default:
				return 0, false
			}
		}
		next, ok := value.(map[string]interface{})
		if !ok {
			return 0, false
		}
		current = next
	}
	return 0, false
}

func metadataStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, strings.TrimSpace(fmt.Sprint(item)))
		}
		return items
	default:
		return nil
	}
}

func cloneStringMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizePathSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(trimmed)
}

func categorizeError(err error) string {
	var providerErr providers.ProviderError
	if errors.As(err, &providerErr) {
		return string(providerErr.Category)
	}
	return "unknown"
}
