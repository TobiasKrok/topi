package utils

import "net/url"

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
