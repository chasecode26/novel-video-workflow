package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"novel-video-workflow/pkg/broadcast"
	"novel-video-workflow/pkg/database"
	"novel-video-workflow/pkg/workflow"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BroadcastEventPublisher publishes workflow events to the broadcast service.
type BroadcastEventPublisher struct{}

// PublishWorkflowStarted publishes a workflow started event.
func (b *BroadcastEventPublisher) PublishWorkflowStarted(chapterID uint) {
	if broadcast.GlobalBroadcastService == nil {
		return
	}
	env := NewWebSocketEventEnvelope(EventWorkflowStarted, WorkflowStartedPayload{
		ChapterID: chapterID,
	})
	msg, _ := json.Marshal(env)
	broadcast.GlobalBroadcastService.SendMessage("workflow", string(msg), time.Now().Format(time.RFC3339))
}

// PublishWorkflowStepChanged publishes a workflow step changed event.
func (b *BroadcastEventPublisher) PublishWorkflowStepChanged(chapterID uint, step string, status string) {
	if broadcast.GlobalBroadcastService == nil {
		return
	}
	env := NewWebSocketEventEnvelope(EventWorkflowStepChanged, WorkflowStepChangedPayload{
		ChapterID: chapterID,
		Step:      step,
		Status:    status,
	})
	msg, _ := json.Marshal(env)
	broadcast.GlobalBroadcastService.SendMessage("workflow", string(msg), time.Now().Format(time.RFC3339))
}

// PublishWorkflowLog publishes a workflow log event.
func (b *BroadcastEventPublisher) PublishWorkflowLog(chapterID uint, level string, message string) {
	if broadcast.GlobalBroadcastService == nil {
		return
	}
	env := NewWebSocketEventEnvelope(EventWorkflowLog, WorkflowLogPayload{
		ChapterID: chapterID,
		Level:     level,
		Message:   message,
	})
	msg, _ := json.Marshal(env)
	broadcast.GlobalBroadcastService.SendMessage("workflow", string(msg), time.Now().Format(time.RFC3339))
}

// PublishWorkflowCompleted publishes a workflow completed event.
func (b *BroadcastEventPublisher) PublishWorkflowCompleted(chapterID uint, durationSec float64) {
	if broadcast.GlobalBroadcastService == nil {
		return
	}
	env := NewWebSocketEventEnvelope(EventWorkflowCompleted, WorkflowCompletedPayload{
		ChapterID:   chapterID,
		DurationSec: durationSec,
	})
	msg, _ := json.Marshal(env)
	broadcast.GlobalBroadcastService.SendMessage("workflow", string(msg), time.Now().Format(time.RFC3339))
}

// PublishWorkflowFailed publishes a workflow failed event.
func (b *BroadcastEventPublisher) PublishWorkflowFailed(chapterID uint, failedStep string, errorMessage string) {
	if broadcast.GlobalBroadcastService == nil {
		return
	}
	env := NewWebSocketEventEnvelope(EventWorkflowFailed, WorkflowFailedPayload{
		ChapterID:    chapterID,
		FailedStep:   failedStep,
		ErrorMessage: errorMessage,
	})
	msg, _ := json.Marshal(env)
	broadcast.GlobalBroadcastService.SendMessage("workflow", string(msg), time.Now().Format(time.RFC3339))
}

// WorkflowRunAPI handles workflow run endpoints.
type WorkflowRunAPI struct {
	executor *workflow.Executor
	storage  workflow.Storage
}

// NewWorkflowRunAPI creates a new workflow run API.
func NewWorkflowRunAPI(executor *workflow.Executor, storage workflow.Storage) *WorkflowRunAPI {
	return &WorkflowRunAPI{
		executor: executor,
		storage:  storage,
	}
}

// RegisterRoutes registers workflow run routes.
func (api *WorkflowRunAPI) RegisterRoutes(router *gin.Engine) {
	router.POST("/api/workflow/runs", api.StartRun)
	router.GET("/api/workflow/runs/:id", api.GetRun)
}

// StartRun starts a new workflow run.
func (api *WorkflowRunAPI) StartRun(c *gin.Context) {
	var req struct {
		ChapterID uint `json:"chapter_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "invalid_request",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	chapter, err := database.GetChapterByID(req.ChapterID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, ErrorResponse{
			Code:    "chapter_not_found",
			Message: "Chapter not found",
		})
		return
	}

	record, err := api.storage.LoadRecordByChapterID(req.ChapterID)
	if err != nil {
		now := time.Now()
		if saveErr := api.storage.Save(workflow.WorkflowRun{
			ChapterID:   req.ChapterID,
			CurrentStep: workflow.StepTTS,
			Status:      workflow.StatusRunning,
			Artifacts:   map[workflow.Step]workflow.ArtifactMetadata{},
			StartedAt:   &now,
		}); saveErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:    "run_create_failed",
				Message: "Failed to create workflow run",
				Details: saveErr.Error(),
			})
			return
		}

		record, err = api.storage.LoadRecordByChapterID(req.ChapterID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:    "run_load_failed",
				Message: "Failed to load workflow run",
				Details: err.Error(),
			})
			return
		}

		go func(chapterID uint) {
			_, _ = api.executor.RunChapterWorkflow(context.Background(), workflow.RunRequest{ChapterID: chapterID})
		}(req.ChapterID)
	} else if record.Status != workflow.StatusRunning {
		go func(chapterID uint) {
			_, _ = api.executor.RunChapterWorkflow(context.Background(), workflow.RunRequest{ChapterID: chapterID})
		}(req.ChapterID)
	}

	artifactMeta := ""
	if encoded, marshalErr := json.Marshal(record.Artifacts); marshalErr == nil {
		artifactMeta = string(encoded)
	}

	resource := WorkflowRunResponse{
		ID:           record.ID,
		ProjectID:    chapter.ProjectID,
		ChapterID:    chapter.ID,
		Status:       string(record.Status),
		CurrentStep:  string(record.CurrentStep),
		ArtifactMeta: artifactMeta,
		StartedAt:    derefTime(record.StartedAt),
		CompletedAt:  derefTime(record.FinishedAt),
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Status: "success",
		Data:   resource,
	})
}

// GetRun retrieves a workflow run by ID.
func (api *WorkflowRunAPI) GetRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid run ID",
		})
		return
	}

	record, err := api.storage.LoadByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "not_found",
			Message: "Workflow run not found",
		})
		return
	}

	chapter, err := database.GetChapterByID(record.ChapterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "chapter_lookup_failed",
			Message: "Failed to load chapter for workflow run",
			Details: err.Error(),
		})
		return
	}

	artifactMeta := ""
	if encoded, marshalErr := json.Marshal(record.Artifacts); marshalErr == nil {
		artifactMeta = string(encoded)
	}

	c.JSON(http.StatusOK, WorkflowRunResponse{
		ID:           record.ID,
		ProjectID:    chapter.ProjectID,
		ChapterID:    record.ChapterID,
		Status:       string(record.Status),
		CurrentStep:  string(record.CurrentStep),
		ErrorMessage: record.ErrorMessage,
		ArtifactMeta: artifactMeta,
		StartedAt:    derefTime(record.StartedAt),
		CompletedAt:  derefTime(record.FinishedAt),
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	})
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
