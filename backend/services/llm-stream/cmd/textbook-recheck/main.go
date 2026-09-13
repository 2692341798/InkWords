// textbook-recheck checks a private rejected draft offline. It cannot call a
// provider, write a revision, approve a manuscript, or execute teaching code.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	textbook "inkwords-backend/services/llm-stream/app/textbook"
	"inkwords-backend/services/llm-stream/infra/rejecteddraft"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("textbook-recheck", flag.ContinueOnError)
	root := flags.String("store", os.Getenv("TEXTBOOK_REJECTED_DRAFTS_DIR"), "private rejected draft directory")
	hash := flags.String("hash", "", "immutable rejected draft hash")
	replace := flags.Bool("markdown-stdin", false, "read replacement Markdown from stdin; original receipt stays unchanged")
	structured := flags.Bool("correction-stdin", false, "read a structured local correction; emit a proposal without saving a candidate")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *root == "" || *hash == "" || flags.NArg() != 0 || *replace && *structured {
		return fmt.Errorf("store and hash are required; no positional arguments")
	}
	store, err := rejecteddraft.Open(*root)
	if err != nil {
		return err
	}
	defer store.Close()
	data, err := store.Load(*hash)
	if err != nil {
		return err
	}
	if *structured {
		edit, err := io.ReadAll(io.LimitReader(input, (1<<20)+1))
		if err != nil {
			return fmt.Errorf("correction input unavailable")
		}
		correction, err := textbook.DecodeSampleCorrection(edit)
		if err != nil {
			return err
		}
		proposal, err := textbook.PrepareSampleCorrection(data, correction)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(output).Encode(proposal); err != nil {
			return fmt.Errorf("write correction proposal")
		}
		if !proposal.Quality.Passed {
			return fmt.Errorf("correction quality checks failed; no candidate was created")
		}
		return nil
	}
	var draft textbook.RejectedDraft
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&draft); err != nil {
		return fmt.Errorf("invalid rejected draft document")
	}
	markdown := draft.Generation.Chapter.Markdown
	if *replace {
		data, err = io.ReadAll(io.LimitReader(input, (1<<20)+1))
		if err != nil || len(data) > 1<<20 {
			return fmt.Errorf("replacement Markdown unavailable or too large")
		}
		markdown = string(data)
	}
	report, err := textbook.RecheckRejectedDraft(draft, markdown)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("write recheck result")
	}
	if !report.Quality.Passed {
		return fmt.Errorf("offline quality checks failed; no candidate was created")
	}
	return nil
}
