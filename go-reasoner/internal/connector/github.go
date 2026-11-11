package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Repo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
}

func GitHubRepos(ctx context.Context, q string, limit int) ([]Repo, error) {
	u := "https://api.github.com/search/repositories?q=" +
		url.QueryEscape(q) + "&sort=stars&per_page=" + fmt.Sprint(limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-reasoner/0.2")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: %s", resp.Status)
	}
	var out struct {
		Items []Repo `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
