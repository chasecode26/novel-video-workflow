package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"novel-video-workflow/pkg/database"

	"gorm.io/gorm"
)

// RunStorage persists workflow-run state in the database.
type RunStorage struct {
	db *gorm.DB
}

// Storage defines the interface for persisting workflow runs.
type Storage interface {
	Save(run WorkflowRun) error
	LoadByID(id uint) (WorkflowRunRecord, error)
	LoadByChapterID(chapterID uint) (WorkflowRun, error)
	LoadRecordByChapterID(chapterID uint) (WorkflowRunRecord, error)
}

// WorkflowRunRecord exposes persisted workflow run metadata needed by APIs.
type WorkflowRunRecord struct {
	ID            uint
	ChapterID     uint
	CurrentStep   Step
	Status        Status
	Artifacts     map[Step]ArtifactMetadata
	ErrorCategory string
	ErrorMessage  string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewRunStorage(db *gorm.DB) *RunStorage {
	return &RunStorage{db: db}
}

func (s *RunStorage) Save(run WorkflowRun) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("run storage is not initialized")
	}

	artifactsJSON, err := marshalArtifacts(run.Artifacts)
	if err != nil {
		return err
	}

	var record database.WorkflowRun
	lookup := s.db.Where("chapter_id = ?", run.ChapterID).First(&record)
	if lookup.Error != nil {
		if lookup.Error != gorm.ErrRecordNotFound {
			return lookup.Error
		}
		record.ChapterID = run.ChapterID
	}

	record.CurrentStep = string(run.CurrentStep)
	record.Status = string(run.Status)
	record.ArtifactMetadata = artifactsJSON
	record.ErrorCategory = run.ErrorCategory
	record.ErrorMessage = run.ErrorMessage
	record.StartedAt = run.StartedAt
	record.FinishedAt = run.FinishedAt

	return s.db.Save(&record).Error
}

func (s *RunStorage) LoadByID(id uint) (WorkflowRunRecord, error) {
	if s == nil || s.db == nil {
		return WorkflowRunRecord{}, fmt.Errorf("run storage is not initialized")
	}

	var record database.WorkflowRun
	if err := s.db.First(&record, id).Error; err != nil {
		return WorkflowRunRecord{}, err
	}

	artifacts, err := unmarshalArtifacts(record.ArtifactMetadata)
	if err != nil {
		return WorkflowRunRecord{}, err
	}

	return WorkflowRunRecord{
		ID:            record.ID,
		ChapterID:     record.ChapterID,
		CurrentStep:   Step(record.CurrentStep),
		Status:        Status(record.Status),
		Artifacts:     artifacts,
		ErrorCategory: record.ErrorCategory,
		ErrorMessage:  record.ErrorMessage,
		StartedAt:     record.StartedAt,
		FinishedAt:    record.FinishedAt,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}, nil
}

func (s *RunStorage) LoadRecordByChapterID(chapterID uint) (WorkflowRunRecord, error) {
	if s == nil || s.db == nil {
		return WorkflowRunRecord{}, fmt.Errorf("run storage is not initialized")
	}

	var record database.WorkflowRun
	if err := s.db.Where("chapter_id = ?", chapterID).First(&record).Error; err != nil {
		return WorkflowRunRecord{}, err
	}

	artifacts, err := unmarshalArtifacts(record.ArtifactMetadata)
	if err != nil {
		return WorkflowRunRecord{}, err
	}

	return WorkflowRunRecord{
		ID:            record.ID,
		ChapterID:     record.ChapterID,
		CurrentStep:   Step(record.CurrentStep),
		Status:        Status(record.Status),
		Artifacts:     artifacts,
		ErrorCategory: record.ErrorCategory,
		ErrorMessage:  record.ErrorMessage,
		StartedAt:     record.StartedAt,
		FinishedAt:    record.FinishedAt,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}, nil
}

func (s *RunStorage) LoadByChapterID(chapterID uint) (WorkflowRun, error) {
	if s == nil || s.db == nil {
		return WorkflowRun{}, fmt.Errorf("run storage is not initialized")
	}

	var record database.WorkflowRun
	if err := s.db.Where("chapter_id = ?", chapterID).First(&record).Error; err != nil {
		return WorkflowRun{}, err
	}

	artifacts, err := unmarshalArtifacts(record.ArtifactMetadata)
	if err != nil {
		return WorkflowRun{}, err
	}

	return WorkflowRun{
		ChapterID:     record.ChapterID,
		CurrentStep:   Step(record.CurrentStep),
		Status:        Status(record.Status),
		Artifacts:     artifacts,
		ErrorCategory: record.ErrorCategory,
		ErrorMessage:  record.ErrorMessage,
		StartedAt:     record.StartedAt,
		FinishedAt:    record.FinishedAt,
	}, nil
}

func marshalArtifacts(artifacts map[Step]ArtifactMetadata) (string, error) {
	if artifacts == nil {
		artifacts = map[Step]ArtifactMetadata{}
	}
	payload := make(map[string]ArtifactMetadata, len(artifacts))
	for step, metadata := range artifacts {
		payload[string(step)] = cloneArtifactMetadata(metadata)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal workflow artifacts: %w", err)
	}
	return string(data), nil
}

func unmarshalArtifacts(raw string) (map[Step]ArtifactMetadata, error) {
	if raw == "" {
		return map[Step]ArtifactMetadata{}, nil
	}
	payload := make(map[string]ArtifactMetadata)
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("unmarshal workflow artifacts: %w", err)
	}
	artifacts := make(map[Step]ArtifactMetadata, len(payload))
	for step, metadata := range payload {
		artifacts[Step(step)] = cloneArtifactMetadata(metadata)
	}
	return artifacts, nil
}
