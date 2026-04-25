package main

import (
	"fmt"
	"inkwords-backend/internal/parser"
)

func main() {
	fmt.Println("Starting...")
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	fetcher := parser.NewGitFetcher()
	
	tree, chunks, err := fetcher.Fetch("https://github.com/krahets/hello-algo")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Tree length:", len(tree))
	fmt.Println("Chunks length:", len(chunks))
}
