package config

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tobiaskrok/topi/shared/workflow"
)

const (
	maxConfigSize = 10 * 1024 * 1024 // 10MB max config size
	httpTimeout   = 30 * time.Second
	maxRetries    = 3
	retryDelay    = 2 * time.Second
)

func LoadConfig(source string) ([]byte, error) {
	if isURL(source) {
		return loadFromURL(source)
	}

	return loadFromFile(source)
}

func LoadAndParse(source string) (workflow.WorkflowConfig, error) {
	data, err := LoadConfig(source)
	if err != nil {
		return workflow.WorkflowConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	config, err := workflow.ParseWorkflow(data)
	if err != nil {
		return workflow.WorkflowConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

func loadFromFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxConfigSize {
		return nil, fmt.Errorf("config file too large: %d bytes (max: %d)", info.Size(), maxConfigSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

func loadFromURL(urlStr string) ([]byte, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s (must be http or https)", parsedURL.Scheme)
	}

	client := &http.Client{
		Timeout: httpTimeout,
	}

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Topi/1.0")
		req.Header.Set("Accept", "application/x-yaml, text/yaml, text/plain")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("server returned status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries)
			continue
		}

		if resp.ContentLength > maxConfigSize {
			return nil, fmt.Errorf("remote config too large: %d bytes (max: %d)", resp.ContentLength, maxConfigSize)
		}

		reader := io.LimitReader(resp.Body, maxConfigSize)
		data, err := io.ReadAll(reader)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("failed to load config from URL after %d attempts: %w", maxRetries, lastErr)
}

func isURL(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}
