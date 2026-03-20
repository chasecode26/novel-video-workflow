package workflow

import (
	"fmt"
	"sync"
)

// MemoryRunStorage is an in-memory implementation for testing.
type MemoryRunStorage struct {
	mu       sync.RWMutex
	nextID   uint
	runs     map[uint]WorkflowRun
	runIDs   map[uint]uint
	chapters map[uint]uint
}

// NewMemoryRunStorage creates a new in-memory storage.
func NewMemoryRunStorage() *MemoryRunStorage {
	return &MemoryRunStorage{
		nextID:   1,
		runs:     map[uint]WorkflowRun{},
		runIDs:   map[uint]uint{},
		chapters: map[uint]uint{},
	}
}

// Save stores a workflow run.
func (s *MemoryRunStorage) Save(run WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runIDs[run.ChapterID]; !ok {
		s.runIDs[run.ChapterID] = s.nextID
		s.chapters[s.nextID] = run.ChapterID
		s.nextID++
	}
	s.runs[run.ChapterID] = run
	return nil
}

// Load retrieves a workflow run by chapter ID.
func (s *MemoryRunStorage) Load(chapterID uint) (WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[chapterID]
	if !ok {
		return WorkflowRun{}, fmt.Errorf("run not found for chapter %d", chapterID)
	}
	return run, nil
}

// LoadByID retrieves a workflow run by persisted run ID.
func (s *MemoryRunStorage) LoadByID(id uint) (WorkflowRunRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chapterID, ok := s.chapters[id]
	if !ok {
		return WorkflowRunRecord{}, fmt.Errorf("run not found for id %d", id)
	}
	return s.loadRecordLocked(id, chapterID)
}

// LoadRecordByChapterID retrieves a workflow run record by chapter ID.
func (s *MemoryRunStorage) LoadRecordByChapterID(chapterID uint) (WorkflowRunRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runID, ok := s.runIDs[chapterID]
	if !ok {
		return WorkflowRunRecord{}, fmt.Errorf("run not found for chapter %d", chapterID)
	}
	return s.loadRecordLocked(runID, chapterID)
}

func (s *MemoryRunStorage) loadRecordLocked(id uint, chapterID uint) (WorkflowRunRecord, error) {
	run, ok := s.runs[chapterID]
	if !ok {
		return WorkflowRunRecord{}, fmt.Errorf("run not found for chapter %d", chapterID)
	}
	return WorkflowRunRecord{
		ID:            id,
		ChapterID:     run.ChapterID,
		CurrentStep:   run.CurrentStep,
		Status:        run.Status,
		Artifacts:     run.Artifacts,
		ErrorCategory: run.ErrorCategory,
		ErrorMessage:  run.ErrorMessage,
		StartedAt:     run.StartedAt,
		FinishedAt:    run.FinishedAt,
	}, nil
}

// LoadByChapterID implements RunStorage interface (alias for Load).
func (s *MemoryRunStorage) LoadByChapterID(chapterID uint) (WorkflowRun, error) {
	return s.Load(chapterID)
}
