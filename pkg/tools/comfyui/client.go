package comfyui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	configpkg "novel-video-workflow/pkg/config"

	"go.uber.org/zap"
)

type Client struct {
	BaseURL        string
	Checkpoint     string
	OutputNodeID   string
	FilenamePrefix string
	Logger         *zap.Logger
	HTTPClient     *http.Client
}

type promptRequest struct {
	Prompt map[string]map[string]interface{} `json:"prompt"`
}

type promptResponse struct {
	PromptID   string                 `json:"prompt_id"`
	NodeErrors map[string]interface{} `json:"node_errors"`
}

type historyResponse map[string]struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
	} `json:"outputs"`
}

func NewClient(logger *zap.Logger, cfg configpkg.ImageComfyUIConfig) *Client {
	baseURL := cfg.APIURL
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8188"
	}
	outputNode := cfg.OutputNodeID
	if outputNode == "" {
		outputNode = "9"
	}
	filenamePrefix := cfg.FilenamePrefix
	if filenamePrefix == "" {
		filenamePrefix = "novel_workflow"
	}

	return &Client{
		BaseURL:        stringsTrimRightSlash(baseURL),
		Checkpoint:     cfg.Checkpoint,
		OutputNodeID:   outputNode,
		FilenamePrefix: filenamePrefix,
		Logger:         logger,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (c *Client) CheckHealth() error {
	if c.Checkpoint == "" {
		return fmt.Errorf("comfyui checkpoint is not configured")
	}

	resp, err := c.HTTPClient.Get(c.BaseURL + "/system_stats")
	if err != nil {
		return fmt.Errorf("connect comfyui: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("comfyui health check failed: %s", string(body))
	}

	checkpoints, err := c.AvailableCheckpoints()
	if err != nil {
		return err
	}
	for _, checkpoint := range checkpoints {
		if checkpoint == c.Checkpoint {
			return nil
		}
	}
	return fmt.Errorf("configured comfyui checkpoint %q not found", c.Checkpoint)
}

func (c *Client) AvailableCheckpoints() ([]string, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/models/checkpoints")
	if err != nil {
		return nil, fmt.Errorf("list comfyui checkpoints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list comfyui checkpoints failed: %s", string(body))
	}

	var checkpoints []string
	if err := json.NewDecoder(resp.Body).Decode(&checkpoints); err != nil {
		return nil, fmt.Errorf("decode comfyui checkpoints: %w", err)
	}
	return checkpoints, nil
}

func (c *Client) GenerateImage(promptText, negativePrompt, outputFile string, width, height, steps int, cfgScale float64) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return err
	}

	graph := c.defaultWorkflow(promptText, negativePrompt, width, height, steps, cfgScale)
	reqBody, err := json.Marshal(promptRequest{Prompt: graph})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/prompt", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("submit comfyui prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("submit comfyui prompt failed: %s", string(body))
	}

	var promptResp promptResponse
	if err := json.NewDecoder(resp.Body).Decode(&promptResp); err != nil {
		return fmt.Errorf("decode comfyui prompt response: %w", err)
	}
	if promptResp.PromptID == "" {
		return fmt.Errorf("comfyui returned empty prompt id")
	}

	imageMeta, err := c.waitForImage(promptResp.PromptID)
	if err != nil {
		return err
	}

	return c.downloadImage(imageMeta.Filename, imageMeta.Subfolder, imageMeta.Type, outputFile)
}

func (c *Client) waitForImage(promptID string) (struct {
	Filename  string
	Subfolder string
	Type      string
}, error) {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		resp, err := c.HTTPClient.Get(c.BaseURL + "/history/" + promptID)
		if err != nil {
			return struct {
				Filename  string
				Subfolder string
				Type      string
			}{}, fmt.Errorf("fetch comfyui history: %w", err)
		}

		var history historyResponse
		err = json.NewDecoder(resp.Body).Decode(&history)
		resp.Body.Close()
		if err != nil {
			return struct {
				Filename  string
				Subfolder string
				Type      string
			}{}, fmt.Errorf("decode comfyui history: %w", err)
		}

		if item, ok := history[promptID]; ok {
			if output, ok := item.Outputs[c.OutputNodeID]; ok && len(output.Images) > 0 {
				image := output.Images[0]
				return struct {
					Filename  string
					Subfolder string
					Type      string
				}{
					Filename:  image.Filename,
					Subfolder: image.Subfolder,
					Type:      image.Type,
				}, nil
			}
		}

		time.Sleep(1500 * time.Millisecond)
	}

	return struct {
		Filename  string
		Subfolder string
		Type      string
	}{}, fmt.Errorf("timed out waiting for comfyui image output")
}

func (c *Client) downloadImage(filename, subfolder, fileType, outputFile string) error {
	query := url.Values{}
	query.Set("filename", filename)
	query.Set("subfolder", subfolder)
	query.Set("type", fileType)

	resp, err := c.HTTPClient.Get(c.BaseURL + "/view?" + query.Encode())
	if err != nil {
		return fmt.Errorf("download comfyui image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download comfyui image failed: %s", string(body))
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func (c *Client) defaultWorkflow(promptText, negativePrompt string, width, height, steps int, cfgScale float64) map[string]map[string]interface{} {
	seed := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1<<62) + 1
	return map[string]map[string]interface{}{
		"4": {
			"class_type": "CheckpointLoaderSimple",
			"inputs": map[string]interface{}{
				"ckpt_name": c.Checkpoint,
			},
		},
		"5": {
			"class_type": "EmptyLatentImage",
			"inputs": map[string]interface{}{
				"width":      width,
				"height":     height,
				"batch_size": 1,
			},
		},
		"6": {
			"class_type": "CLIPTextEncode",
			"inputs": map[string]interface{}{
				"text": promptText,
				"clip": []interface{}{"4", 1},
			},
		},
		"7": {
			"class_type": "CLIPTextEncode",
			"inputs": map[string]interface{}{
				"text": negativePrompt,
				"clip": []interface{}{"4", 1},
			},
		},
		"3": {
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"seed":         seed,
				"steps":        steps,
				"cfg":          cfgScale,
				"sampler_name": "euler",
				"scheduler":    "normal",
				"denoise":      1,
				"model":        []interface{}{"4", 0},
				"positive":     []interface{}{"6", 0},
				"negative":     []interface{}{"7", 0},
				"latent_image": []interface{}{"5", 0},
			},
		},
		"8": {
			"class_type": "VAEDecode",
			"inputs": map[string]interface{}{
				"samples": []interface{}{"3", 0},
				"vae":     []interface{}{"4", 2},
			},
		},
		"9": {
			"class_type": "SaveImage",
			"inputs": map[string]interface{}{
				"filename_prefix": c.FilenamePrefix,
				"images":          []interface{}{"8", 0},
			},
		},
	}
}

func stringsTrimRightSlash(value string) string {
	for len(value) > 0 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	return value
}
