package database

import (
	"path/filepath"
	"testing"

	"gorm.io/gorm"
)

func TestInitDB_InitializesPureGoSQLiteDatabase(t *testing.T) {
	setupTestDB(t)
}

func TestInitDB_ExposesChapterSchemaViaPragma(t *testing.T) {
	db := setupTestDB(t)

	rows, err := db.Raw("PRAGMA table_info(chapters);").Rows()
	if err != nil {
		t.Fatalf("PRAGMA table_info failed: %v", err)
	}
	defer rows.Close()

	foundWorkflowParams := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan pragma row failed: %v", err)
		}
		if name == "workflow_params" {
			foundWorkflowParams = true
		}
	}

	if !foundWorkflowParams {
		t.Fatal("expected workflow_params column in chapters schema")
	}
}

func TestInitDB_SupportsMinimalProjectAndChapterCRUD(t *testing.T) {
	db := setupTestDB(t)

	project := &Project{Name: "project-1", Description: "demo", GlobalPrompt: "mood", PasswordHash: "hash"}
	if err := db.Create(project).Error; err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	chapter := &Chapter{Title: "chapter-1", Content: "body", Prompt: "prompt", ProjectID: project.ID, ImagePaths: "[]"}
	if err := db.Create(chapter).Error; err != nil {
		t.Fatalf("create chapter failed: %v", err)
	}

	var loaded Chapter
	if err := db.Where("id = ?", chapter.ID).First(&loaded).Error; err != nil {
		t.Fatalf("load chapter failed: %v", err)
	}
	if loaded.ProjectID != project.ID {
		t.Fatalf("expected project id %d, got %d", project.ID, loaded.ProjectID)
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	DB = nil

	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	if DB == nil {
		t.Fatal("expected global DB to be initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("DB.DB failed: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		DB = nil
	})

	return DB
}
