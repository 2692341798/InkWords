package parser

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
	"unicode"

	"inkwords-backend/shared/kernel/textbook"
)

// buildGoChunks parses syntax only: imports, init functions and directives are
// source data. Whole declarations keep a function's control flow citeable.
func buildGoChunks(content, documentID, artifactPath, canonicalLocator string) ([]textbook.SourceChunk, error) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, artifactPath, content, goparser.ParseComments|goparser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("无法可靠解析 Go 源码：语法不完整或无效，请检查原文件")
	}
	chunks := make([]textbook.SourceChunk, 0, len(file.Decls)+1)
	physicalFile := fset.File(file.Pos())
	appendChunk := func(start, end int, symbol string) {
		raw := content[start:end]
		text := strings.TrimSpace(raw)
		if text == "" {
			return
		}
		start += len(raw) - len(strings.TrimLeftFunc(raw, unicode.IsSpace))
		end = start + len(text)
		chunk := newStructuredChunk(documentID, len(chunks)+1, 0, start, end,
			fset.PositionFor(physicalFile.Pos(start), false).Line, fset.PositionFor(physicalFile.Pos(end), false).Line,
			artifactPath, canonicalLocator, "go", nil, text)
		chunk.Locator.Symbol = symbol
		chunks = append(chunks, chunk)
	}
	cursor := 0
	for _, decl := range file.Decls {
		startPos, symbol := decl.Pos(), ""
		switch node := decl.(type) {
		case *ast.FuncDecl:
			if node.Doc != nil {
				startPos = node.Doc.Pos()
			}
			symbol = node.Name.Name
			if node.Recv != nil && len(node.Recv.List) == 1 {
				symbol = "(" + goReceiverName(node.Recv.List[0].Type) + ")." + symbol
			}
		case *ast.GenDecl:
			if node.Doc != nil {
				startPos = node.Doc.Pos()
			}
			if len(node.Specs) == 1 {
				if spec, ok := node.Specs[0].(*ast.TypeSpec); ok {
					symbol = spec.Name.Name
				}
			}
		}
		// Ignore //line remapping: evidence always points into the frozen file.
		start := fset.PositionFor(startPos, false).Offset
		end := fset.PositionFor(decl.End(), false).Offset
		appendChunk(cursor, start, "")
		appendChunk(start, end, symbol)
		cursor = end
	}
	appendChunk(cursor, len(content), "")
	return chunks, nil
}

func goReceiverName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return "*" + goReceiverName(node.X)
	case *ast.IndexExpr:
		return goReceiverName(node.X)
	case *ast.IndexListExpr:
		return goReceiverName(node.X)
	case *ast.ParenExpr:
		return goReceiverName(node.X)
	default:
		return ""
	}
}
