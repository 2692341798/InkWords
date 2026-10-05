// textbook-artifact-reconcile repairs a generated revision's missing artifact
// projection. It never changes the manuscript, task state or approval records.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	artifact "inkwords-backend/services/core-api/app/textbookartifact"
	textbook "inkwords-backend/services/core-api/domain/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	task := flag.String("task", "", "original generation task ID")
	revision := flag.String("revision", "", "expected immutable revision ID")
	hash := flag.String("content-hash", "", "expected original Markdown SHA-256 hex")
	apply := flag.Bool("apply", false, "stage and register the exact previewed projection")
	toolchain := flag.String("toolchain", "", "exact operator-selected Go runner version; omit only to recover a legacy manifest")
	selection := flag.String("dependency-selection", "", "operator-reviewed chapter/source selection JSON")
	selectionHash := flag.String("selection-hash", "", "expected SHA-256 of the selection file")
	dependencyRoot := flag.String("dependency-root", "", "absolute operator-owned dependency tree")
	dependencyManifest := flag.String("dependency-manifest", "", "reviewed inventory JSON")
	flag.Parse()
	useDependencies := *selection != "" || *selectionHash != "" || *dependencyRoot != "" || *dependencyManifest != ""
	if useDependencies && (*selection == "" || *selectionHash == "" || *dependencyRoot == "" || *dependencyManifest == "" || *toolchain == "") {
		return errors.New("all dependency selection flags and exact toolchain are required")
	}
	taskID, err := uuid.Parse(*task)
	if err != nil || uuid.Validate(*revision) != nil || len(*hash) != 64 || flag.NArg() != 0 {
		return errors.New("task, revision and content-hash are required")
	}
	if os.Getenv("DATABASE_URL") == "" {
		return errors.New("core database is not configured")
	}
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("core database unavailable")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return errors.New("core database unavailable")
	}
	defer sqlDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo := textbook.NewGormRepository(db)
	generated, err := repo.GetGeneratedRevisionContext(ctx, taskID)
	if err != nil {
		return errors.New("generated candidate unavailable")
	}
	if generated.Revision.ID.String() != *revision || generated.Revision.ContentHash != *hash {
		return errors.New("candidate identity changed; recovery refused")
	}
	input, err := artifact.ManuscriptGoInput(generated.Revision)
	if *toolchain != "" {
		input, err = artifact.ManuscriptGoInputForToolchain(generated.Revision, *toolchain)
	}
	var selected *artifact.SelectedOfflineGoBundle
	if useDependencies {
		selected, err = artifact.ReadSelectedOfflineGoBundle(ctx, repo, *selection, *selectionHash, *dependencyRoot, *dependencyManifest)
		if err != nil {
			return err
		}
		if selected.Selection().Toolchain != *toolchain {
			return errors.New("selection and requested toolchains differ")
		}
		input, err = selected.ManuscriptInput(ctx, repo, generated)
	}
	if err != nil {
		return err
	}
	input.BookContractHash, input.StyleSheetHash = generated.BookContractHash, generated.StyleSheetHash
	result := map[string]any{"mode": "preview", "revision_id": *revision, "content_hash": *hash, "artifact_id": input.ArtifactID, "book_contract_hash": input.BookContractHash, "style_sheet_hash": input.StyleSheetHash, "provider_calls": 0, "code_executed": false}
	if selected != nil {
		result["dependency_selection"], result["selection_hash"] = selected.Selection(), *selectionHash
	}
	files := make([]string, 0, len(input.Files))
	for _, file := range input.Files {
		files = append(files, file.Path)
	}
	result["files"] = files
	result["toolchain_version"] = input.ToolchainVersion
	result["commands"] = input.Commands
	if *apply {
		root := os.Getenv("TEXTBOOK_TEACHING_ARTIFACTS_DIR")
		if root == "" {
			return errors.New("teaching artifact store is not configured")
		}
		service := artifact.NewService(teachingartifact.NewStore(root).WithReadGroup(teachingartifact.ReaderGroupID), textbook.NewService(repo))
		row, err := service.StageAndRegister(ctx, generated.WorkspaceID, input)
		if err != nil {
			return errors.New("artifact registration failed; original candidate unchanged")
		}
		result["mode"], result["artifact_hash"], result["status"] = "applied", row.ArtifactHash, row.Status
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
