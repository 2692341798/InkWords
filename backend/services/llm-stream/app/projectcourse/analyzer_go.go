package projectcourse

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"inkwords-backend/shared/kernel/projectcourse"
)

type GoAnalyzer struct{}

func (GoAnalyzer) Supports(file InventoryEntry) bool {
	return strings.HasSuffix(strings.ToLower(file.Path), ".go") && file.Disposition != DispositionExcluded
}

func (a GoAnalyzer) Analyze(ctx context.Context, snapshot projectcourse.SourceSnapshot, files []InventoryEntry) (SemanticFacts, error) {
	if err := snapshot.Validate(); err != nil {
		return SemanticFacts{}, err
	}
	fset := token.NewFileSet()
	var facts SemanticFacts
	for _, file := range files {
		select {
		case <-ctx.Done():
			return SemanticFacts{}, ctx.Err()
		default:
		}
		if !a.Supports(file) || file.Content == "" {
			continue
		}
		tree, err := parser.ParseFile(fset, file.Path, file.Content, parser.ParseComments)
		if err != nil {
			continue
		}
		facts.Symbols = append(facts.Symbols, SymbolRecord{Path: file.Path, Name: tree.Name.Name, Kind: SymbolPackage, StartLine: lineOf(fset, tree.Pos()), EndLine: lineOf(fset, tree.End())})
		for _, importSpec := range tree.Imports {
			facts.Symbols = append(facts.Symbols, SymbolRecord{Path: file.Path, Name: strings.Trim(importSpec.Path.Value, `"`), Kind: SymbolImport, StartLine: lineOf(fset, importSpec.Pos()), EndLine: lineOf(fset, importSpec.End())})
		}
		for _, declaration := range tree.Decls {
			a.collectDeclaration(fset, file, declaration, &facts)
		}
	}
	sortFacts(&facts)
	return facts, nil
}

func (GoAnalyzer) collectDeclaration(fset *token.FileSet, file InventoryEntry, declaration ast.Decl, facts *SemanticFacts) {
	gen, ok := declaration.(*ast.GenDecl)
	if ok {
		for _, spec := range gen.Specs {
			typeSpec, isType := spec.(*ast.TypeSpec)
			if !isType {
				continue
			}
			kind := SymbolType
			if _, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
				kind = SymbolInterface
			}
			facts.Symbols = append(facts.Symbols, SymbolRecord{Path: file.Path, Name: typeSpec.Name.Name, Kind: kind, StartLine: lineOf(fset, typeSpec.Pos()), EndLine: lineOf(fset, typeSpec.End())})
		}
		return
	}
	funcDecl, ok := declaration.(*ast.FuncDecl)
	if !ok || funcDecl.Name == nil {
		return
	}
	kind := SymbolFunction
	name := funcDecl.Name.Name
	if funcDecl.Recv != nil {
		kind = SymbolMethod
	}
	if strings.Contains(strings.ToLower(name), "consume") || strings.Contains(strings.ToLower(name), "worker") {
		kind = SymbolConsumer
	}
	if strings.Contains(strings.ToLower(name), "route") || strings.HasPrefix(name, "Register") {
		kind = SymbolRoute
	}
	facts.Symbols = append(facts.Symbols, SymbolRecord{Path: file.Path, Name: name, Kind: kind, StartLine: lineOf(fset, funcDecl.Pos()), EndLine: lineOf(fset, funcDecl.End())})
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok {
			facts.Relations = append(facts.Relations, RelationRecord{FromPath: file.Path, FromSymbol: name, ToSymbol: selector.Sel.Name, Kind: RelationCalls})
		}
		ident, ok := call.Fun.(*ast.Ident)
		if ok && ident.Name != name {
			facts.Relations = append(facts.Relations, RelationRecord{FromPath: file.Path, FromSymbol: name, ToSymbol: ident.Name, Kind: RelationCalls})
		}
		return true
	})
}

func lineOf(fset *token.FileSet, position token.Pos) int {
	return fset.Position(position).Line
}

func sortFacts(facts *SemanticFacts) {
	sort.Slice(facts.Symbols, func(i, j int) bool {
		left, right := facts.Symbols[i], facts.Symbols[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.StartLine != right.StartLine {
			return left.StartLine < right.StartLine
		}
		return left.Name < right.Name
	})
	sort.Slice(facts.Relations, func(i, j int) bool {
		left, right := facts.Relations[i], facts.Relations[j]
		if left.FromPath != right.FromPath {
			return left.FromPath < right.FromPath
		}
		if left.FromSymbol != right.FromSymbol {
			return left.FromSymbol < right.FromSymbol
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.ToSymbol < right.ToSymbol
	})
}
