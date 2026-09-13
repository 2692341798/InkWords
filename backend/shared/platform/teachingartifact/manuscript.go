package teachingartifact

import (
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	goast "go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ManuscriptGoFiles extracts the exact, explicitly labelled teaching files.
// Both artifact staging and filming guides use this parser to prevent drift.
// It does not execute source or grant permission to stage or run it.
func ManuscriptGoFiles(markdown string) (map[string][]byte, []string, error) {
	source := []byte(markdown)
	document := goldmark.New().Parser().Parse(text.NewReader(source))
	files := make(map[string][]byte)
	limitations := make([]string, 0, 2)
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		block, ok := node.(*ast.FencedCodeBlock)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		// Nested/quoted examples and other languages are not an executable
		// teaching tree. Refuse ambiguity rather than infer a command or path.
		if block.Parent() != document || string(block.Language(source)) != "go" {
			return ast.WalkStop, fmt.Errorf("teaching projection requires top-level Go fences only")
		}
		intro, ok := block.PreviousSibling().(*ast.Paragraph)
		if !ok {
			return ast.WalkStop, fmt.Errorf("teaching file origin paragraph missing")
		}
		label := string(intro.Lines().Value(source))
		filename := ""
		switch {
		case strings.HasPrefix(label, "**代码来源：教学实现（main.go）。**"):
			filename = "main.go"
			if !strings.Contains(label, "非生产用途") || !strings.Contains(label, "省略") {
				return ast.WalkStop, fmt.Errorf("teaching implementation limitations missing")
			}
		case strings.HasPrefix(label, "**代码来源：教学实现测试（main_test.go）。**"):
			filename = "main_test.go"
		default:
			return ast.WalkStop, fmt.Errorf("teaching file origin is not an approved sample filename")
		}
		if _, exists := files[filename]; exists {
			return ast.WalkStop, fmt.Errorf("duplicate teaching file: %s", filename)
		}
		body := append([]byte(nil), block.Lines().Value(source)...)
		if err := validateProjectedGoFile(filename, body); err != nil {
			return ast.WalkStop, err
		}
		files[filename] = body
		limitations = append(limitations, strings.TrimSpace(label))
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(files) != 2 || files["main.go"] == nil || files["main_test.go"] == nil {
		return nil, nil, fmt.Errorf("teaching projection requires main.go and main_test.go")
	}
	return files, limitations, nil
}

func validateProjectedGoFile(filename string, body []byte) error {
	file, err := parser.ParseFile(token.NewFileSet(), filename, body, parser.AllErrors)
	if err != nil || file.Name.Name != "main" {
		return fmt.Errorf("invalid teaching Go source: %s", filename)
	}
	hasMain, hasTest := false, false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*goast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		hasMain = hasMain || function.Name.Name == "main"
		hasTest = hasTest || (strings.HasPrefix(function.Name.Name, "Test") && function.Name.Name != "TestMain")
	}
	if (filename == "main.go" && (!hasMain || hasTest)) || (filename == "main_test.go" && (!hasTest || hasMain)) {
		return fmt.Errorf("teaching Go declarations do not match filename: %s", filename)
	}
	return nil
}
