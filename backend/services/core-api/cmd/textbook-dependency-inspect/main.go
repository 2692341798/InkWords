// Command textbook-dependency-inspect checks an operator-selected offline Go
// dependency inventory without staging, activating, compiling or running it.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	artifact "inkwords-backend/services/core-api/app/textbookartifact"
	textbook "inkwords-backend/services/core-api/domain/textbook"
	"io"
	"os"
	"time"

	"inkwords-backend/shared/platform/teachingartifact"
)

func main() {
	root := flag.String("root", "", "absolute path to the operator-owned vendor tree")
	manifest := flag.String("manifest", "", "path to the separately reviewed inventory JSON")
	hash := flag.String("manifest-hash", "", "expected sha256 inventory digest from review")
	selection := flag.String("selection", "", "optional explicit chapter/source selection JSON; requires DATABASE_URL")
	selectionHash := flag.String("selection-hash", "", "reviewed SHA-256 of the selection file")
	flag.Parse()
	if *selection != "" || *selectionHash != "" {
		if err := inspectSelection(*root, *manifest, *hash, *selection, *selectionHash); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := inspect(*root, *manifest, *hash); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func inspectSelection(root, manifest, hash, selectionPath, selectionHash string) error {
	if os.Getenv("DATABASE_URL") == "" {
		return fmt.Errorf("core database is not configured")
	}
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("core database unavailable")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("core database unavailable")
	}
	defer sqlDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	selected, err := artifact.ReadSelectedOfflineGoBundle(ctx, textbook.NewGormRepository(db), selectionPath, selectionHash, root, manifest)
	if err != nil {
		return err
	}
	if selected.Selection().DependencyManifestHash != hash {
		return fmt.Errorf("selection and command manifest hashes differ")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"selection": selected.Selection(), "selection_hash": selectionHash, "validated": true, "activated": false, "executed": false, "registered": false})
}

func inspect(root, manifest, hash string) error {
	file, err := os.Open(manifest)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(file, (2<<20)+1))
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	bundle, err := teachingartifact.LoadOfflineGoBundle(context.Background(), root, data, hash)
	if err != nil {
		return err
	}
	files := bundle.Files()
	var size int64
	for _, file := range files {
		size += int64(len(file.Content))
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		ManifestHash string                              `json:"manifest_hash"`
		Origin       teachingartifact.GoDependencyOrigin `json:"origin"`
		Toolchain    string                              `json:"toolchain"`
		FileCount    int                                 `json:"file_count"`
		Bytes        int64                               `json:"bytes"`
		Activated    bool                                `json:"activated"`
		Executed     bool                                `json:"executed"`
	}{bundle.Hash(), bundle.Origin(), bundle.Toolchain(), len(files), size, false, false})
}
