package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GitTreeResponse struct {
	Sha       string `json:"sha"`
	Url       string `json:"url"`
	Tree      []GitTreeItem `json:"tree"`
	Truncated bool   `json:"truncated"`
}

type GitTreeItem struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	Sha  string `json:"sha"`
	Size int    `json:"size"`
	Url  string `json:"url"`
}

func main() {
	owner, repo := "samber", "lo"
	subDir := "parallel"

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo)
	fmt.Println("Fetching tree:", apiURL)
	
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
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

	var treeResp GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		fmt.Println("Decode error:", err)
		return
	}

	fmt.Println("Total items:", len(treeResp.Tree))

	var filesToFetch []string
	prefix := subDir
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	for _, item := range treeResp.Tree {
		if item.Type == "blob" {
			if prefix == "" || strings.HasPrefix(item.Path, prefix) {
				filesToFetch = append(filesToFetch, item.Path)
			}
		}
	}

	fmt.Println("Files to fetch in", subDir, ":", len(filesToFetch))
	if len(filesToFetch) > 0 {
		fmt.Println("First few files:", filesToFetch[:min(3, len(filesToFetch))])
		
		// test fetch one file using raw url
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/%s", owner, repo, filesToFetch[0])
		fmt.Println("Fetching raw:", rawURL)
		
		reqRaw, _ := http.NewRequest("GET", rawURL, nil)
		respRaw, err := client.Do(reqRaw)
		if err == nil {
			defer respRaw.Body.Close()
			b, _ := io.ReadAll(respRaw.Body)
			fmt.Printf("Content length: %d bytes\n", len(b))
		}
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
