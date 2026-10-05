package services_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyStandaloneCommandWrappersAreRemoved(t *testing.T) {
	legacyDirs := []string{"core-api", "llm-stream", "parser-service", "export-service"}
	for _, name := range legacyDirs {
		path := filepath.Join("..", "cmd", name)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("legacy command wrapper %s must be removed", path)
		}
	}
}

var backendServices = []string{
	"core-api",
	"llm-stream",
	"parser-service",
	"export-service",
	"review-service",
}

func TestLocalServiceEntrypointsDoNotRestoreJWTAuthMiddleware(t *testing.T) {
	paths := make([]string, 0, len(backendServices)+1)
	for _, service := range backendServices {
		paths = append(paths, filepath.Join(service, "app", "bootstrap", "bootstrap.go"))
	}
	paths = append(paths, filepath.Join("..", "cmd", "server", "main.go"))

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			//nolint:gosec
			contentsBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if strings.Contains(string(contentsBytes), "httpx.AuthMiddleware()") {
				t.Fatalf("%s must resolve the fixed local workspace or compatibility owner instead of trusting request JWT identity", path)
			}
		})
	}
}

func TestLegacyAuthenticationPackagesAreRemoved(t *testing.T) {
	for _, path := range []string{
		filepath.Join("core-api", "domain", "auth"),
		filepath.Join("..", "pkg", "jwt"),
	} {
		matches, err := filepath.Glob(filepath.Join(path, "*.go"))
		if err != nil {
			t.Fatalf("inspect %s: %v", path, err)
		}
		if len(matches) != 0 {
			t.Fatalf("legacy authentication package %s must stay removed: %v", path, matches)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "shared", "kernel", "httpx", "auth.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("shared JWT authentication middleware must stay removed")
	}

	routesPath := filepath.Join("core-api", "transport", "http", "v1", "routes.go")
	//nolint:gosec
	routesBytes, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("read %s: %v", routesPath, err)
	}
	for _, forbidden := range []string{"/auth", "AuthLogin", "AuthOAuth", "AuthGetCaptcha"} {
		if strings.Contains(string(routesBytes), forbidden) {
			t.Fatalf("%s must not restore legacy authentication surface %q", routesPath, forbidden)
		}
	}

}

func TestLegacyProjectCourseHTTPMutationSurfaceIsRemoved(t *testing.T) {
	for _, retiredDir := range []string{
		filepath.Join("core-api", "domain", "projectcourse"),
		filepath.Join("llm-stream", "app", "projectcourse"),
		filepath.Join("..", "shared", "kernel", "projectcourse"),
	} {
		matches, err := filepath.Glob(filepath.Join(retiredDir, "*.go"))
		if err != nil {
			t.Fatalf("inspect %s: %v", retiredDir, err)
		}
		if len(matches) != 0 {
			t.Fatalf("legacy ProjectCourse package must stay removed: %v", matches)
		}
	}

	for path, forbidden := range map[string][]string{
		filepath.Join("core-api", "transport", "http", "v1", "routes.go"):   {"/project-courses", "ProjectCourseCreate"},
		filepath.Join("core-api", "app", "bootstrap", "bootstrap.go"):       {"projectCourseHandler", "projectCourseRepo", "projectcoursedomain"},
		filepath.Join("llm-stream", "domain", "stream", "task_consumer.go"): {"projectCourseRunner", "project_course_generate"},
		filepath.Join("course-runner", "cmd", "main.go"):                    {"StartVerificationConsumer", "inkwords.course-verification"},
		filepath.Join("export-service", "domain", "export", "worker.go"):    {"coursePackageBuilder", "project_course_package"},
		filepath.Join("..", "shared", "platform", "rabbitmq", "message.go"): {"type VerificationRequestedMessage struct", "course.verification.requested"},
		filepath.Join("..", "cmd", "server", "main.go"):                     {"projectCourseHandler", "projectCourseRepo", "projectcoursedomain"},
	} {
		//nolint:gosec
		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		contents := string(contentsBytes)
		for _, symbol := range forbidden {
			if strings.Contains(contents, symbol) {
				t.Fatalf("%s must not restore retired ProjectCourse surface %q", path, symbol)
			}
		}
	}
}

func TestLocalSingleUserSurfaceDoesNotRestoreAccountProfileHTTP(t *testing.T) {
	routesPath := filepath.Join("core-api", "transport", "http", "v1", "routes.go")
	//nolint:gosec
	routesBytes, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("read %s: %v", routesPath, err)
	}
	for _, forbidden := range []string{"/user", "UserProfile", "UserStats", "UserUploadAvatar", "UserGetPromptSetting"} {
		if strings.Contains(string(routesBytes), forbidden) {
			t.Fatalf("%s must not restore retired local account surface %q", routesPath, forbidden)
		}
	}

	legacyDomain := filepath.Join("core-api", "domain", "user")
	matches, err := filepath.Glob(filepath.Join(legacyDomain, "*.go"))
	if err != nil {
		t.Fatalf("inspect %s: %v", legacyDomain, err)
	}
	if len(matches) != 0 {
		t.Fatalf("legacy user account domain must stay removed: %v", matches)
	}
}

func TestLocalSingleUserRuntimeDoesNotRestorePerUserQuotaOrPromptTables(t *testing.T) {
	for _, retiredFile := range []string{
		filepath.Join("llm-stream", "app", "generation", "quota_service.go"),
		filepath.Join("parser-service", "domain", "parse", "quota_store.go"),
		filepath.Join("..", "internal", "model", "user.go"),
		filepath.Join("..", "internal", "model", "user_prompt_settings.go"),
		filepath.Join("..", "internal", "model", "oauth_token.go"),
	} {
		if _, err := os.Stat(retiredFile); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("local single-user runtime must not restore per-user quota adapter %s", retiredFile)
		}
	}

	legacyMigrationPath := filepath.Join("..", "internal", "infra", "db", "db.go")
	//nolint:gosec
	legacyMigrationBytes, err := os.ReadFile(legacyMigrationPath)
	if err != nil {
		t.Fatalf("read %s: %v", legacyMigrationPath, err)
	}
	for _, forbidden := range []string{"model.User{}", "model.UserPromptSettings{}", "model.OAuthToken{}"} {
		if strings.Contains(string(legacyMigrationBytes), forbidden) {
			t.Fatalf("%s must not restore retired account or per-user prompt model %q", legacyMigrationPath, forbidden)
		}
	}

	legacyOwnerPath := filepath.Join("..", "internal", "model", "legacy_owner.go")
	if _, err := os.Stat(legacyOwnerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy owner model must be removed after workspace cleanup: %s", legacyOwnerPath)
	}
	for _, required := range []string{"legacyCoreCleanupVersion", "migrationApplied", "legacyOwnerBootstrap"} {
		if !strings.Contains(string(legacyMigrationBytes), required) {
			t.Fatalf("%s must gate migration-only bootstrap projection with cleanup state %q", legacyMigrationPath, required)
		}
	}

	promptPath := filepath.Join("llm-stream", "app", "generation", "prompt_service.go")
	//nolint:gosec
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read %s: %v", promptPath, err)
	}
	for _, forbidden := range []string{"user_prompt_settings", "gorm.io/", "UserID"} {
		if strings.Contains(string(promptBytes), forbidden) {
			t.Fatalf("%s must resolve workspace-default prompt requirements without per-user storage %q", promptPath, forbidden)
		}
	}

	for _, path := range []string{
		filepath.Join("core-api", "domain", "task", "generation_result_repository.go"),
		filepath.Join("core-api", "domain", "blog", "persistence.go"),
		filepath.Join("llm-stream", "domain", "stream", "blog_persistence.go"),
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(contents), "tokens_used") {
			t.Fatalf("%s must keep generation usage in task/candidate telemetry instead of a per-user token balance", path)
		}
	}
}

func TestProviderRequestsDoNotSendLegacyUserIdentity(t *testing.T) {
	providerPath := filepath.Join("..", "shared", "platform", "llm", "deepseek.go")
	//nolint:gosec
	providerBytes, err := os.ReadFile(providerPath)
	if err != nil {
		t.Fatalf("read %s: %v", providerPath, err)
	}
	for _, forbidden := range []string{"UserID", "user_id", "sanitizeUserID"} {
		if strings.Contains(string(providerBytes), forbidden) {
			t.Fatalf("%s must not send local legacy identity to the provider via %q", providerPath, forbidden)
		}
	}

	for _, path := range []string{
		filepath.Join("llm-stream", "app", "generation", "generator_service.go"),
		filepath.Join("llm-stream", "app", "generation", "decomposition_continue.go"),
		filepath.Join("llm-stream", "app", "generation", "decomposition_quality.go"),
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(contents), "options.UserID") {
			t.Fatalf("%s must not attach legacy identity to provider options", path)
		}
	}

	for _, path := range []string{
		filepath.Join("llm-stream", "app", "generation", "decomposition_types.go"),
		filepath.Join("llm-stream", "app", "generation", "decomposition_quality.go"),
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(contents), "UserID") {
			t.Fatalf("%s must not retain synthetic provider identity inside the quality pipeline", path)
		}
	}
}

func TestReviewRuntimeUsesLocalWorkspaceInsteadOfLegacyOwner(t *testing.T) {
	for path, forbidden := range map[string][]string{
		filepath.Join("review-service", "app", "bootstrap", "bootstrap.go"): {
			"LocalLegacyOwnerContext", "NewLocalLegacyOwnerResolver", "legacyOwnerMiddleware",
		},
		filepath.Join("review-service", "transport", "http", "v1", "routes.go"): {
			"legacyOwnerMiddleware", "user_id",
		},
		filepath.Join("review-service", "domain", "review", "handler.go"): {
			"getUserID", "user_id",
		},
		filepath.Join("review-service", "domain", "review", "model.go"): {
			"UserID", "user_id",
		},
		filepath.Join("review-service", "domain", "review", "repository.go"): {
			"UserID", "userID", "user_id",
		},
		filepath.Join("review-service", "domain", "review", "session_service.go"): {
			"UserID", "userID", "user_id",
		},
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, symbol := range forbidden {
			if strings.Contains(string(contents), symbol) {
				t.Fatalf("%s must use the installation workspace instead of legacy review identity %q", path, symbol)
			}
		}
	}
}

func TestTaskRuntimeUsesLocalWorkspaceInsteadOfLegacyOwner(t *testing.T) {
	for path, forbidden := range map[string][]string{
		filepath.Join("core-api", "domain", "task", "model.go"):                        {"RequestedBy", "requested_by"},
		filepath.Join("core-api", "domain", "task", "handler.go"):                      {"getUserID", "user_id", "UserID", "userID"},
		filepath.Join("core-api", "domain", "task", "download_handler.go"):             {"getUserID", "user_id", "UserID", "userID"},
		filepath.Join("core-api", "domain", "task", "service.go"):                      {"requestedBy", "UserID", "userID"},
		filepath.Join("core-api", "domain", "task", "generation_result_repository.go"): {"RequestedBy", "requested_by", "legacy_user_id"},
		filepath.Join("core-api", "infra", "mq", "publisher.go"):                       {"UserID", "user_id"},
		filepath.Join("core-api", "app", "bootstrap", "bootstrap.go"):                  {"LocalLegacyOwnerContext", "NewLocalLegacyOwnerResolver", "legacyOwnerMiddleware"},
		filepath.Join("core-api", "transport", "http", "v1", "routes.go"):              {"RegisterLegacyRoutes", "LegacyHandlers", "legacyOwnerMiddleware"},
		filepath.Join("llm-stream", "domain", "stream", "task_consumer.go"):            {"message.UserID", "user_id"},
		filepath.Join("parser-service", "domain", "parse", "task_consumer.go"):         {"message.UserID", "user_id"},
		filepath.Join("export-service", "domain", "export", "dto.go"):                  {"UserID", "user_id"},
		filepath.Join("..", "shared", "kernel", "httpx", "local_workspace.go"):         {"LocalLegacyOwnerContext", "legacyOwnerContextKey", "LOCAL_LEGACY_OWNER_UNAVAILABLE", "user_id"},
		filepath.Join("..", "shared", "platform", "postgres", "local_workspace.go"):    {"NewLocalLegacyOwnerResolver", "LegacyOwnerID", "DEV_AUTH_USER_ID", "local_workspace_legacy_owner"},
		filepath.Join("..", "shared", "platform", "rabbitmq", "message.go"):            {"UserID", "user_id"},
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, symbol := range forbidden {
			if strings.Contains(string(contents), symbol) {
				t.Fatalf("%s must use workspace task identity instead of legacy owner symbol %q", path, symbol)
			}
		}
	}

	//nolint:gosec
	routes, err := os.ReadFile(filepath.Join("core-api", "transport", "http", "v1", "routes.go"))
	if err != nil {
		t.Fatalf("read task routes: %v", err)
	}
	if !strings.Contains(string(routes), "RegisterTaskRoutes(r *gin.Engine, workspaceMiddleware") {
		t.Fatal("task routes must require the installation workspace middleware")
	}
}

func TestBlogRuntimeUsesLocalWorkspaceInsteadOfLegacyOwner(t *testing.T) {
	for path, forbidden := range map[string][]string{
		filepath.Join("core-api", "domain", "blog", "handler.go"):              {"user_id", "UserID", "userID"},
		filepath.Join("core-api", "domain", "blog", "model.go"):                {"user_id", "UserID"},
		filepath.Join("core-api", "domain", "blog", "repository.go"):           {"user_id", "UserID", "userID"},
		filepath.Join("core-api", "domain", "blog", "service.go"):              {"user_id", "UserID", "userID"},
		filepath.Join("llm-stream", "domain", "stream", "blog_persistence.go"): {"user_id", "UserID", "userID"},
		filepath.Join("llm-stream", "domain", "stream", "blog_readable.go"):    {"user_id", "UserID", "userID"},
		filepath.Join("export-service", "domain", "export", "model.go"):        {"user_id", "UserID"},
		filepath.Join("export-service", "domain", "export", "repository.go"):   {"user_id", "UserID", "userID"},
		filepath.Join("export-service", "domain", "export", "handler.go"):      {"currentUserID", "user_id", "UserID", "userID"},
		filepath.Join("llm-stream", "app", "bootstrap", "bootstrap.go"):        {"LocalLegacyOwnerContext", "NewLocalLegacyOwnerResolver"},
		filepath.Join("export-service", "app", "bootstrap", "bootstrap.go"):    {"LocalLegacyOwnerContext", "NewLocalLegacyOwnerResolver"},
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, symbol := range forbidden {
			if strings.Contains(string(contents), symbol) {
				t.Fatalf("%s must use the installation workspace for blog ownership instead of %q", path, symbol)
			}
		}
	}
}

func TestProjectPreparationHTTPUsesLocalWorkspaceInsteadOfLegacyOwner(t *testing.T) {
	for path, forbidden := range map[string][]string{
		filepath.Join("core-api", "domain", "project", "handler.go"):            {"user_id", "UserID", "userID"},
		filepath.Join("parser-service", "domain", "parse", "handler.go"):        {"user_id", "UserID", "userID"},
		filepath.Join("parser-service", "app", "bootstrap", "bootstrap.go"):     {"LocalLegacyOwnerContext", "NewLocalLegacyOwnerResolver"},
		filepath.Join("parser-service", "transport", "http", "v1", "routes.go"): {"legacyOwnerMiddleware"},
	} {
		//nolint:gosec
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, symbol := range forbidden {
			if strings.Contains(string(contents), symbol) {
				t.Fatalf("%s must use the installation workspace for project preparation instead of %q", path, symbol)
			}
		}
	}

	//nolint:gosec
	coreBootstrap, err := os.ReadFile(filepath.Join("core-api", "app", "bootstrap", "bootstrap.go"))
	if err != nil {
		t.Fatalf("read core bootstrap: %v", err)
	}
	if !strings.Contains(string(coreBootstrap), "RegisterProjectRoutes(r, workspaceMiddleware") {
		t.Fatal("core project routes must be registered with the installation workspace middleware")
	}
}

func TestServiceDockerfilesAreOwnedByEachService(t *testing.T) {
	//nolint:gosec
	composeBytes, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	compose := string(composeBytes)

	for _, service := range backendServices {
		t.Run(service, func(t *testing.T) {
			dockerfilePath := filepath.Join(service, "Dockerfile")
			//nolint:gosec
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

func TestLLMStreamDefaultsToTaskOnlyPersistence(t *testing.T) {
	//nolint:gosec
	composeBytes, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}

	const expected = "INKWORDS_TASK_PERSISTENCE_MODE: ${INKWORDS_TASK_PERSISTENCE_MODE:-task_only}"
	compose := string(composeBytes)
	llmStart := strings.Index(compose, "\n  llm-stream:")
	parserStart := strings.Index(compose, "\n  parser-service:")
	if llmStart < 0 || parserStart <= llmStart {
		t.Fatal("docker-compose.yml must contain ordered llm-stream and parser-service sections")
	}
	if !strings.Contains(compose[llmStart:parserStart], expected) {
		t.Fatalf("llm-stream must default to task-only persistence so generation workers can build final task results")
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

				//nolint:gosec
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

				//nolint:gosec
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

				//nolint:gosec
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

		//nolint:gosec
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
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		//nolint:gosec
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

		//nolint:gosec
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

				//nolint:gosec
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

		//nolint:gosec
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

		//nolint:gosec
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

				//nolint:gosec
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

func TestTextbookPackagesDoNotDependOnLegacyIdentityOrProjectCourse(t *testing.T) {
	textbookDirs := []string{
		filepath.Join("core-api", "app", "textbookartifact"),
		filepath.Join("core-api", "app", "textbookgeneration"),
		filepath.Join("core-api", "app", "textbookimport"),
		filepath.Join("core-api", "domain", "textbook"),
		filepath.Join("llm-stream", "app", "textbook"),
		filepath.Join("course-runner", "domain", "textbookverification"),
		filepath.Join("..", "shared", "kernel", "textbook"),
		filepath.Join("..", "shared", "platform", "teachingartifact"),
		filepath.Join("..", "shared", "platform", "visualasset"),
	}
	disallowedImports := []string{
		"inkwords-backend/pkg/jwt",
		"inkwords-backend/services/core-api/domain/auth",
		"inkwords-backend/services/core-api/domain/projectcourse",
		"inkwords-backend/services/core-api/domain/user",
	}

	for _, dir := range textbookDirs {
		t.Run(dir, func(t *testing.T) {
			err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}

				//nolint:gosec
				contentsBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				contents := string(contentsBytes)
				for _, disallowedImport := range disallowedImports {
					if strings.Contains(contents, disallowedImport) {
						t.Fatalf("%s imports legacy package %q; textbook packages must use the local workspace and textbook contracts", path, disallowedImport)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", dir, err)
			}
		})
	}
}

func TestTextbookTaskAccessUsesWorkspaceIdentityInsteadOfLegacyOwner(t *testing.T) {
	path := filepath.Join("core-api", "app", "textbookgeneration", "task_access.go")
	//nolint:gosec
	contentsBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	contents := string(contentsBytes)
	for _, required := range []string{"GetTextbookTask", "RetryTextbookTask"} {
		if !strings.Contains(contents, required) {
			t.Fatalf("%s must authorize textbook tasks through %s", path, required)
		}
	}
	for _, forbidden := range []string{"LegacyOwnerID", "shared/platform/postgres", "RetryGenerationTask"} {
		if strings.Contains(contents, forbidden) {
			t.Fatalf("%s must not use legacy task ownership marker %q", path, forbidden)
		}
	}
}

func TestTextbookTaskCreatorsDoNotResolveLegacyOwners(t *testing.T) {
	paths := []string{
		filepath.Join("core-api", "app", "textbookgeneration", "service.go"),
		filepath.Join("core-api", "app", "textbookimport", "service.go"),
		filepath.Join("core-api", "app", "textbookartifact", "verification_task.go"),
	}
	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			//nolint:gosec
			contentsBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			contents := string(contentsBytes)
			for _, forbidden := range []string{"LegacyOwnerID", "shared/platform/postgres", "resolveLocal"} {
				if strings.Contains(contents, forbidden) {
					t.Fatalf("%s must not resolve textbook tasks through legacy ownership marker %q", path, forbidden)
				}
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

		//nolint:gosec
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

		//nolint:gosec
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

		//nolint:gosec
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

		//nolint:gosec
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

func TestTextbookKernelDoesNotImportRuntimeOrServicePackages(t *testing.T) {
	kernelDir := filepath.Join("..", "shared", "kernel", "textbook")
	disallowedImports := []string{
		"github.com/gin-gonic/gin",
		"gorm.io/",
		"rabbitmq",
		"inkwords-backend/services/",
		"inkwords-backend/shared/platform/",
	}

	err := filepath.WalkDir(kernelDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		contentsBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(contentsBytes)
		for _, disallowedImport := range disallowedImports {
			if strings.Contains(contents, disallowedImport) {
				return fmt.Errorf("%s imports runtime or service package %q", path, disallowedImport)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
