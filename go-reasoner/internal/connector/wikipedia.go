package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func Wikipedia(ctx context.Context, topic string) (string, error) {
	u := "https://en.wikipedia.org/api/rest_v1/page/summary/" + url.PathEscape(topic)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "go-reasoner/0.2 (educational)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wikipedia: %s", resp.Status)
	}
	var s struct {
		Extract string `json:"extract"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", err
	}
	if s.Extract == "" {
		return "", fmt.Errorf("wikipedia: empty extract")
	}
	return s.Extract, nil
}
