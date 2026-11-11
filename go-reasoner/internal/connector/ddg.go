package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func DuckDuckGo(ctx context.Context, q string) (string, error) {
	u := "https://api.duckduckgo.com/?q=" + url.QueryEscape(q) + "&format=json&no_html=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "go-reasoner/0.2")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ddg: %s", resp.Status)
	}
	var r struct {
		AbstractText string `json:"AbstractText"`
		Answer       string `json:"Answer"`
		Definition   string `json:"Definition"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	for _, v := range []string{r.AbstractText, r.Answer, r.Definition} {
		if v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("ddg: no answer")
}
