package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func ValidateURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	if parsedURL.Host == "" && parsedURL.Path == "" {
		return false
	}

	return true
}
func ParseGitURL(gitURL string) (owner, repo string, err error) {
	// Remove .git suffix
	gitURL = strings.TrimSuffix(gitURL, ".git")

	// Handle HTTPS URLs: https://gitea:3000/topi/api-service
	if strings.Contains(gitURL, "://") {
		parts := strings.Split(gitURL, "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid git URL")
		}
		owner = parts[len(parts)-2]
		repo = parts[len(parts)-1]
		return owner, repo, nil
	}

	// Handle SSH URLs: git@gitea:topi/api-service
	if strings.Contains(gitURL, ":") {
		parts := strings.Split(gitURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid git URL")
		}
		ownerRepo := strings.Split(parts[1], "/")
		if len(ownerRepo) != 2 {
			return "", "", fmt.Errorf("invalid git URL")
		}
		return ownerRepo[0], ownerRepo[1], nil
	}

	return "", "", fmt.Errorf("unsupported git URL format")
}
