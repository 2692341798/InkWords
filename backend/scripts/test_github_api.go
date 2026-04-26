package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func parseGithubOwnerRepo(urlStr string) (owner, repo string, ok bool) {
	urlStr = strings.TrimSpace(urlStr)
	urlStr = strings.TrimSuffix(urlStr, ".git")
	urlStr = strings.TrimSuffix(urlStr, "/")

	if strings.HasPrefix(urlStr, "https://github.com/") || strings.HasPrefix(urlStr, "http://github.com/") {
		parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(urlStr, "https://"), "http://"), "/")
		if len(parts) >= 3 && parts[0] == "github.com" {
			return parts[1], parts[2], true
		}
	} else if strings.HasPrefix(urlStr, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(urlStr, "git@github.com:"), "/")
		if len(parts) == 2 {
			return parts[0], parts[1], true
		}
	}
	return "", "", false
}

func main() {
	urls := []string{
		"https://github.com/samber/lo",
		"https://github.com/samber/lo.git",
		"git@github.com:samber/lo.git",
		"https://gitlab.com/samber/lo",
	}

	for _, u := range urls {
		owner, repo, ok := parseGithubOwnerRepo(u)
		fmt.Printf("%s -> %s / %s (ok: %v)\n", u, owner, repo, ok)
	}

	owner, repo := "samber", "lo"
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", owner, repo)
	
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	// req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Status:", resp.Status)
		return
	}

	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		fmt.Println("Decode error:", err)
		return
	}

	fmt.Println("Directories:")
	for _, item := range contents {
		if item.Type == "dir" {
			fmt.Println("-", item.Name)
		}
	}
}
