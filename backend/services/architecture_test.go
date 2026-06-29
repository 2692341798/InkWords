package services_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var backendServices = []string{
	"core-api",
	"llm-stream",
	"parser-service",
	"export-service",
	"review-service",
}

func TestServiceDockerfilesAreOwnedByEachService(t *testing.T) {
	composeBytes, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	compose := string(composeBytes)

	for _, service := range backendServices {
		t.Run(service, func(t *testing.T) {
			dockerfilePath := filepath.Join(service, "Dockerfile")
			contentsBytes, err := os.ReadFile(dockerfilePath)
			if err != nil {
				t.Fatalf("read %s: %v", dockerfilePath, err)
			}
			contents := string(contentsBytes)

			wantBuildPackage := "./services/" + service + "/cmd"
			if !strings.Contains(contents, wantBuildPackage) {
				t.Fatalf("%s must build its service-owned cmd package %q", dockerfilePath, wantBuildPackage)
			}

			wantCommand := `CMD ["./` + service + `"]`
			if !strings.Contains(contents, wantCommand) {
				t.Fatalf("%s must default to %s", dockerfilePath, wantCommand)
			}

			wantComposeDockerfile := "dockerfile: services/" + service + "/Dockerfile"
			if !strings.Contains(compose, wantComposeDockerfile) {
				t.Fatalf("docker-compose.yml must build %s with %q", service, wantComposeDockerfile)
			}
		})
	}
}

func TestServicesDoNotImportPeerServicePackages(t *testing.T) {
	for _, service := range backendServices {
		t.Run(service, func(t *testing.T) {
			serviceDir := service
			err := filepath.WalkDir(serviceDir, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				contentsBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				contents := string(contentsBytes)
				for _, peer := range backendServices {
					if peer == service {
						continue
					}
					disallowedImport := `inkwords-backend/services/` + peer
					if strings.Contains(contents, disallowedImport) {
						t.Fatalf("%s imports peer service package %q", path, disallowedImport)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", serviceDir, err)
			}
		})
	}
}

func TestServicesUseSharedHTTPRuntimeContract(t *testing.T) {
	disallowedImports := []string{
		"inkwords-backend/internal/transport/http/middleware",
		"inkwords-backend/internal/infra/mq",
		"inkwords-backend/internal/infra/llm",
		"inkwords-backend/internal/infra/cache",
		"inkwords-backend/internal/infra/parser",
	}

	for _, service := range backendServices {
		t.Run(service, func(t *testing.T) {
			err := filepath.WalkDir(service, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				contentsBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				contents := string(contentsBytes)
				for _, disallowedImport := range disallowedImports {
					if strings.Contains(contents, disallowedImport) {
						t.Fatalf("%s imports legacy runtime contract %q; use shared packages or service-owned infra instead", path, disallowedImport)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", service, err)
			}
		})
	}
}

func TestWorkerDomainsDoNotDependOnLegacyTaskDomain(t *testing.T) {
	ownedWorkerDirs := []string{
		filepath.Join("parser-service", "domain"),
		filepath.Join("export-service", "domain"),
		filepath.Join("export-service", "infra"),
	}
	disallowedImport := "inkwords-backend/internal/domain/task"

	for _, dir := range ownedWorkerDirs {
		t.Run(dir, func(t *testing.T) {
			err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				contentsBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if strings.Contains(string(contentsBytes), disallowedImport) {
					t.Fatalf("%s imports legacy task domain %q; worker domains should depend on local interfaces instead", path, disallowedImport)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", dir, err)
			}
		})
	}
}

func TestReviewDomainOwnsReviewModels(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/model"
	dir := filepath.Join("review-service", "domain")

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy model package %q; review-service domain should own review models", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}

func TestParserServiceUsesSharedParserPlatform(t *testing.T) {
	disallowedImport := "inkwords-backend/services/parser-service/infra/parser"
	serviceDir := "parser-service"

	err := filepath.WalkDir(serviceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports service-owned parser infra %q; parser-service must use shared/platform/parser instead", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", serviceDir, err)
	}
}

func TestReviewServiceDoesNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	serviceDir := "review-service"

	err := filepath.WalkDir(serviceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy internal package %q; review-service should use shared packages or service-owned code", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", serviceDir, err)
	}
}

func TestExportOwnedPackagesDoNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	ownedDirs := []string{
		"export-service",
	}

	for _, dir := range ownedDirs {
		t.Run(dir, func(t *testing.T) {
			err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				contentsBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if strings.Contains(string(contentsBytes), disallowedImport) {
					t.Fatalf("%s imports legacy internal package %q; export-service owned packages should use shared packages or service-owned code", path, disallowedImport)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", dir, err)
			}
		})
	}
}

func TestParserServiceDoesNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	serviceDir := "parser-service"

	err := filepath.WalkDir(serviceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy internal package %q; parser-service should use shared packages or service-owned code", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", serviceDir, err)
	}
}

func TestCoreAPITaskDomainDoesNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	dir := filepath.Join("core-api", "domain", "task")

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy internal package %q; core-api task domain should own task projections and contracts", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}

func TestCoreAPIOwnedUserFacingDomainsDoNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	ownedDirs := []string{
		filepath.Join("core-api", "domain", "auth"),
		filepath.Join("core-api", "domain", "blog"),
		filepath.Join("core-api", "domain", "project"),
		filepath.Join("core-api", "domain", "user"),
	}

	for _, dir := range ownedDirs {
		t.Run(dir, func(t *testing.T) {
			err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				contentsBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if strings.Contains(string(contentsBytes), disallowedImport) {
					t.Fatalf("%s imports legacy internal package %q; core-api owned domains should use shared packages or service-owned projections", path, disallowedImport)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", dir, err)
			}
		})
	}
}

func TestLLMStreamDoesNotImportLegacyInternalPackages(t *testing.T) {
	serviceDir := "llm-stream"
	disallowedImport := "inkwords-backend/internal/"

	err := filepath.WalkDir(serviceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy internal package %q; llm-stream should use service-owned or shared packages", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", serviceDir, err)
	}
}

func TestCoreAPIInfraDoesNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	dir := filepath.Join("core-api", "infra")

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy internal package %q; core-api infra should use shared platform contracts", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}

func TestCoreAPIDoesNotImportLegacyInternalPackages(t *testing.T) {
	disallowedImport := "inkwords-backend/internal/"
	serviceDir := "core-api"

	err := filepath.WalkDir(serviceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contentsBytes), disallowedImport) {
			t.Fatalf("%s imports legacy internal package %q; core-api must not depend on legacy internal/ packages", path, disallowedImport)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", serviceDir, err)
	}
}

func TestCmdServerDoesNotImportLegacyInternalBusinessPackages(t *testing.T) {
	entrypointDir := filepath.Join("..", "cmd", "server")

	err := filepath.WalkDir(entrypointDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(contentsBytes)

		disallowedPrefixes := []string{
			"inkwords-backend/internal/domain/",
			"inkwords-backend/internal/service",
			"inkwords-backend/internal/transport/",
			"inkwords-backend/internal/prompt",
			"inkwords-backend/internal/model",
			"inkwords-backend/internal/infra/cache",
			"inkwords-backend/internal/infra/llm",
			"inkwords-backend/internal/infra/mq",
			"inkwords-backend/internal/infra/parser",
		}

		for _, prefix := range disallowedPrefixes {
			if strings.Contains(contents, prefix) {
				t.Fatalf("%s imports legacy business package %q; cmd/server must use service-owned or shared packages", path, prefix)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", entrypointDir, err)
	}
}
