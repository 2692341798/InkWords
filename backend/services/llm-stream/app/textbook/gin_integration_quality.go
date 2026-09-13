package textbook

import (
	"go/ast"
	"go/parser"
	"go/token"
	"inkwords-backend/shared/platform/teachingartifact"
	"strconv"
	"strings"
)

// validateGinIntegrationImplementation checks source structure only. Test
// outcomes and HTTP behavior still require an isolated VerificationRun.
func validateGinIntegrationImplementation(markdown string) []string {
	files, _, err := teachingartifact.ManuscriptGoFiles(markdown)
	if err != nil {
		return []string{"gin_integration_files_invalid"}
	}
	failures := validateTeachingGoSources(markdown)
	mainFile, _ := parser.ParseFile(token.NewFileSet(), "main.go", files["main.go"], 0)
	testFile, _ := parser.ParseFile(token.NewFileSet(), "main_test.go", files["main_test.go"], 0)
	imports := integrationImports(mainFile)
	ginName, httpName := imports["github.com/gin-gonic/gin"], imports["net/http"]
	if ginName == "" || ginName == "_" || httpName == "" || httpName == "_" {
		failures = append(failures, "gin_integration_imports_missing")
	}
	build, main := integrationFunction(mainFile, "buildRouter"), integrationFunction(mainFile, "main")
	if build == nil || !integrationEngineResult(build, ginName) || !integrationHasCall(build, ginName, "New") || !integrationRoute(build, "GET", "/orders") || !integrationRoute(build, "POST", "/orders") {
		failures = append(failures, "gin_integration_router_contract_missing")
	}
	listens := false
	if main != nil && main.Body != nil {
		ast.Inspect(main.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 || !integrationQualified(call.Fun, httpName, "ListenAndServe") {
				return true
			}
			factory, ok := call.Args[1].(*ast.CallExpr)
			if ok {
				name, ok := factory.Fun.(*ast.Ident)
				listens = listens || (ok && name.Name == "buildRouter" && len(factory.Args) == 0 && integrationString(call.Args[0]) == "127.0.0.1:38080")
			}
			return true
		})
	}
	if !listens {
		failures = append(failures, "gin_integration_loopback_server_missing")
	}
	testImports := integrationImports(testFile)
	if !integrationHasCall(testFile, testImports["net/http/httptest"], "NewRecorder") || !integrationHasCall(testFile, testImports["net/http"], "NewRequest") || !integrationHasCall(testFile, "*", "ServeHTTP") {
		failures = append(failures, "gin_integration_request_test_missing")
	}
	// These are minimum observable assertion shapes, not proof of meaningful
	// coverage. The contract and subsequent runtime/manual review check semantics.
	status, body := false, false
	ast.Inspect(testFile, func(n ast.Node) bool {
		binary, ok := n.(*ast.BinaryExpr)
		if !ok || (binary.Op != token.NEQ && binary.Op != token.EQL) {
			return true
		}
		ast.Inspect(binary, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				status = status || s.Sel.Name == "Code"
				body = body || s.Sel.Name == "Body"
			}
			return true
		})
		return true
	})
	if !status || !body {
		failures = append(failures, "gin_integration_response_assertions_missing")
	}
	for _, marker := range []string{"go mod init", "go test ./...", "go run .", "http://127.0.0.1:38080/orders", "进程内", "未验证"} {
		if !strings.Contains(markdown, marker) {
			failures = append(failures, "gin_integration_operation_missing: "+marker)
		}
	}
	return failures
}

func integrationImports(file *ast.File) map[string]string {
	result := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		parts := strings.Split(path, "/")
		name := parts[len(parts)-1]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		result[path] = name
	}
	return result
}
func integrationFunction(file *ast.File, name string) *ast.FuncDecl {
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil && f.Name.Name == name {
			return f
		}
	}
	return nil
}
func integrationEngineResult(f *ast.FuncDecl, ginName string) bool {
	if f.Type.Results == nil || len(f.Type.Results.List) != 1 {
		return false
	}
	p, ok := f.Type.Results.List[0].Type.(*ast.StarExpr)
	return ok && integrationQualified(p.X, ginName, "Engine")
}
func integrationQualified(expr ast.Expr, receiver, name string) bool {
	s, ok := expr.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != name || receiver == "" || receiver == "_" {
		return false
	}
	if receiver == "*" {
		return true
	}
	id, ok := s.X.(*ast.Ident)
	return ok && id.Name == receiver
}
func integrationHasCall(node ast.Node, receiver, name string) bool {
	found := false
	if node != nil {
		ast.Inspect(node, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				found = found || integrationQualified(call.Fun, receiver, name)
			}
			return true
		})
	}
	return found
}
func integrationString(expr ast.Expr) string {
	if value, ok := expr.(*ast.BasicLit); ok && value.Kind == token.STRING {
		s, _ := strconv.Unquote(value.Value)
		return s
	}
	return ""
}
func integrationRoute(f *ast.FuncDecl, method, path string) bool {
	if f == nil || f.Body == nil {
		return false
	}
	found := false
	ast.Inspect(f.Body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && len(c.Args) >= 2 && integrationQualified(c.Fun, "*", method) {
			found = found || integrationString(c.Args[0]) == path
		}
		return true
	})
	return found
}
