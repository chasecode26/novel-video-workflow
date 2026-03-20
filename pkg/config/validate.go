package config

import (
	"fmt"
	"strings"
)

// ValidationError captures a single config validation failure.
type ValidationError struct {
	Field   string
	Message string
}

// ValidationErrors is a structured collection of config validation failures.
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	parts := make([]string, 0, len(errs))
	for _, validationErr := range errs {
		parts = append(parts, fmt.Sprintf("%s: %s", validationErr.Field, validationErr.Message))
	}
	return strings.Join(parts, "; ")
}

// ValidateConfig validates the typed config and only checks fields required by the enabled providers.
func ValidateConfig(cfg Config) error {
	var errs ValidationErrors

	if strings.TrimSpace(cfg.Paths.BaseDir) == "" {
		errs = append(errs, ValidationError{Field: "paths.base_dir", Message: "is required"})
	}

	switch normalizeProviderName(cfg.TTS.Provider) {
	case "":
		errs = append(errs, ValidationError{Field: "tts.provider", Message: "is required"})
	case "indextts2", "windows-indextts2":
		if strings.TrimSpace(cfg.TTS.IndexTTS2.APIURL) == "" {
			errs = append(errs, ValidationError{Field: "tts.indextts2.api_url", Message: "is required when provider is indextts2 or windows-indextts2"})
		}
	case "mock":
	default:
		errs = append(errs, ValidationError{Field: "tts.provider", Message: "unsupported provider"})
	}

	switch normalizeProviderName(cfg.Subtitle.Provider) {
	case "":
		errs = append(errs, ValidationError{Field: "subtitle.provider", Message: "is required"})
	case "aegisub", "windows-aegisub":
		if strings.TrimSpace(cfg.Subtitle.Aegisub.ScriptPath) == "" {
			errs = append(errs, ValidationError{Field: "subtitle.aegisub.script_path", Message: "is required when provider is aegisub or windows-aegisub"})
		}
	case "mock":
	default:
		errs = append(errs, ValidationError{Field: "subtitle.provider", Message: "unsupported provider"})
	}

	switch normalizeProviderName(cfg.Image.Provider) {
	case "":
		errs = append(errs, ValidationError{Field: "image.provider", Message: "is required"})
	case "drawthings", "windows-drawthings":
		if strings.TrimSpace(cfg.Image.DrawThings.APIURL) == "" {
			errs = append(errs, ValidationError{Field: "image.drawthings.api_url", Message: "is required when provider is drawthings or windows-drawthings"})
		}
	case "mock":
	default:
		errs = append(errs, ValidationError{Field: "image.provider", Message: "unsupported provider"})
	}

	switch normalizeProviderName(cfg.Project.Provider) {
	case "":
		errs = append(errs, ValidationError{Field: "project.provider", Message: "is required"})
	case "capcut", "mock":
	default:
		errs = append(errs, ValidationError{Field: "project.provider", Message: "unsupported provider"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
