package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractChapterTxt_SingleFileNovel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	novelPath := filepath.Join(dir, "novel.txt")
	content := "第一章 重生\n第一章内容\n\n第十二章 围城\n第十二章内容\n\n第13章 终局\n第十三章内容\n"
	if err := os.WriteFile(novelPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write novel: %v", err)
	}

	fm := NewFileManager()
	chapters, err := fm.ExtractChapterTxt(novelPath)
	if err != nil {
		t.Fatalf("ExtractChapterTxt returned error: %v", err)
	}

	if len(chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(chapters))
	}
	if got := chapters[1]; got == "" {
		t.Fatalf("expected chapter 1 content")
	}
	if got := chapters[12]; got == "" {
		t.Fatalf("expected chapter 12 content")
	}
	if got := chapters[13]; got == "" {
		t.Fatalf("expected chapter 13 content")
	}
}

func TestConvertChineseNumberToArabic(t *testing.T) {
	t.Parallel()

	fm := NewFileManager()
	cases := map[string]int{
		"十":     10,
		"十二":    12,
		"二十":    20,
		"二十三":   23,
		"一百零二":  102,
		"三百一十五": 315,
		"一千零一":  1001,
		"一万零三十": 10030,
	}

	for input, want := range cases {
		if got := fm.convertChineseNumberToArabic(input); got != want {
			t.Fatalf("convertChineseNumberToArabic(%q) = %d, want %d", input, got, want)
		}
	}
}
