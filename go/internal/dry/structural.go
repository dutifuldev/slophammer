package dry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type structuralEntry struct {
	location     Range
	nodes        int
	fingerprints map[string]bool
}

type syntaxNode struct {
	Tag      string
	Children []syntaxNode
}

func findStructural(files []sourceFile, options Options) ([]Finding, error) {
	var entries []structuralEntry
	for _, file := range files {
		found, err := structuralEntries(file, options)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}

	var findings []Finding
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			score := structuralSimilarity(entries[i], entries[j])
			if score < options.StructuralThreshold {
				continue
			}
			findings = append(findings, Finding{
				Kind:   "structural-function",
				Left:   entries[i].location,
				Right:  entries[j].location,
				Score:  score,
				Nodes:  min(entries[i].nodes, entries[j].nodes),
				Engine: "structural",
			})
		}
	}
	return findings, nil
}

func structuralEntries(file sourceFile, options Options) ([]structuralEntry, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, file.Path, file.Content, 0)
	if err != nil {
		return nil, err
	}
	var entries []structuralEntry
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start := fileSet.Position(fn.Pos()).Line
		end := fileSet.Position(fn.End()).Line
		normalized := normalizeFunc(fn)
		nodes := syntaxNodeCount(normalized)
		if end-start+1 < options.StructuralMinLines || nodes < options.StructuralMinNodes {
			continue
		}
		entries = append(entries, structuralEntry{
			location:     Range{Path: file.Path, StartLine: start, EndLine: end},
			nodes:        nodes,
			fingerprints: syntaxFingerprints(normalized),
		})
	}
	return entries, nil
}

func normalizeFunc(fn *ast.FuncDecl) syntaxNode {
	children := []syntaxNode{normalizeFieldList("params", fn.Type.Params), normalizeFieldList("results", fn.Type.Results)}
	if fn.Recv != nil {
		children = append(children, normalizeFieldList("receiver", fn.Recv))
	}
	children = append(children, normalizeNode(fn.Body))
	return syntaxNode{Tag: "func", Children: children}
}

func normalizeFieldList(tag string, fields *ast.FieldList) syntaxNode {
	if fields == nil {
		return syntaxNode{Tag: tag}
	}
	children := make([]syntaxNode, 0, len(fields.List))
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			children = append(children, syntaxNode{Tag: "field", Children: []syntaxNode{normalizeNode(field.Type)}})
		}
	}
	return syntaxNode{Tag: tag, Children: children}
}

func normalizeNode(node ast.Node) syntaxNode {
	switch x := node.(type) {
	case nil:
		return syntaxNode{Tag: "nil"}
	case ast.Stmt:
		return normalizeStmt(x)
	case ast.Expr:
		return normalizeExpr(x)
	default:
		return syntaxNode{Tag: fmt.Sprintf("%T", node)}
	}
}

func normalizeStmt(stmt ast.Stmt) syntaxNode {
	switch x := stmt.(type) {
	case *ast.BlockStmt:
		return normalizeList("block", stmtNodes(x.List))
	case *ast.IfStmt:
		return syntaxNode{Tag: "if", Children: []syntaxNode{normalizeNode(x.Init), normalizeNode(x.Cond), normalizeNode(x.Body), normalizeNode(x.Else)}}
	case *ast.ForStmt:
		return syntaxNode{Tag: "for", Children: []syntaxNode{normalizeNode(x.Init), normalizeNode(x.Cond), normalizeNode(x.Post), normalizeNode(x.Body)}}
	case *ast.RangeStmt:
		return syntaxNode{Tag: "range", Children: []syntaxNode{normalizeNode(x.X), normalizeNode(x.Body)}}
	default:
		return normalizeBranchStmt(stmt)
	}
}

func normalizeBranchStmt(stmt ast.Stmt) syntaxNode {
	switch x := stmt.(type) {
	case *ast.SwitchStmt:
		return syntaxNode{Tag: "switch", Children: []syntaxNode{normalizeNode(x.Init), normalizeNode(x.Tag), normalizeNode(x.Body)}}
	case *ast.TypeSwitchStmt:
		return syntaxNode{Tag: "type-switch", Children: []syntaxNode{normalizeNode(x.Init), normalizeNode(x.Assign), normalizeNode(x.Body)}}
	case *ast.SelectStmt:
		return syntaxNode{Tag: "select", Children: []syntaxNode{normalizeNode(x.Body)}}
	case *ast.CaseClause:
		return syntaxNode{Tag: "case", Children: []syntaxNode{normalizeList("case-list", exprNodes(x.List)), normalizeList("case-body", stmtNodes(x.Body))}}
	case *ast.CommClause:
		return syntaxNode{Tag: "comm", Children: []syntaxNode{normalizeNode(x.Comm), normalizeList("comm-body", stmtNodes(x.Body))}}
	default:
		return normalizeSimpleStmt(stmt)
	}
}

func normalizeSimpleStmt(stmt ast.Stmt) syntaxNode {
	switch x := stmt.(type) {
	case *ast.AssignStmt:
		return syntaxNode{Tag: "assign/" + x.Tok.String(), Children: []syntaxNode{normalizeList("lhs", exprNodes(x.Lhs)), normalizeList("rhs", exprNodes(x.Rhs))}}
	case *ast.DeclStmt:
		return syntaxNode{Tag: "decl", Children: []syntaxNode{normalizeDecl(x.Decl)}}
	case *ast.ExprStmt:
		return syntaxNode{Tag: "expr-stmt", Children: []syntaxNode{normalizeNode(x.X)}}
	case *ast.ReturnStmt:
		return normalizeList("return", exprNodes(x.Results))
	case *ast.BranchStmt:
		return syntaxNode{Tag: "branch/" + x.Tok.String()}
	default:
		return normalizeCallStmt(stmt)
	}
}

func normalizeCallStmt(stmt ast.Stmt) syntaxNode {
	switch x := stmt.(type) {
	case *ast.GoStmt:
		return syntaxNode{Tag: "go", Children: []syntaxNode{normalizeNode(x.Call)}}
	case *ast.DeferStmt:
		return syntaxNode{Tag: "defer", Children: []syntaxNode{normalizeNode(x.Call)}}
	case *ast.SendStmt:
		return syntaxNode{Tag: "send", Children: []syntaxNode{normalizeNode(x.Chan), normalizeNode(x.Value)}}
	default:
		return normalizeTinyStmt(stmt)
	}
}

func normalizeTinyStmt(stmt ast.Stmt) syntaxNode {
	switch x := stmt.(type) {
	case *ast.IncDecStmt:
		return syntaxNode{Tag: "incdec/" + x.Tok.String(), Children: []syntaxNode{normalizeNode(x.X)}}
	case *ast.LabeledStmt:
		return syntaxNode{Tag: "label", Children: []syntaxNode{normalizeNode(x.Stmt)}}
	case *ast.EmptyStmt:
		return syntaxNode{Tag: "empty"}
	case *ast.BadStmt:
		return syntaxNode{Tag: "bad-stmt"}
	default:
		return syntaxNode{Tag: fmt.Sprintf("%T", stmt)}
	}
}

func normalizeExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.BinaryExpr:
		return syntaxNode{Tag: "binary/" + x.Op.String(), Children: []syntaxNode{normalizeNode(x.X), normalizeNode(x.Y)}}
	case *ast.UnaryExpr:
		return syntaxNode{Tag: "unary/" + x.Op.String(), Children: []syntaxNode{normalizeNode(x.X)}}
	case *ast.CallExpr:
		return syntaxNode{Tag: "call", Children: append([]syntaxNode{normalizeCallee(x.Fun)}, exprNodes(x.Args)...)}
	case *ast.SelectorExpr:
		return syntaxNode{Tag: "selector", Children: []syntaxNode{normalizeNode(x.X), {Tag: "member"}}}
	default:
		return normalizeIndexExpr(expr)
	}
}

func normalizeIndexExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.IndexExpr:
		return syntaxNode{Tag: "index", Children: []syntaxNode{normalizeNode(x.X), normalizeNode(x.Index)}}
	case *ast.IndexListExpr:
		return syntaxNode{Tag: "index-list", Children: append([]syntaxNode{normalizeNode(x.X)}, exprNodes(x.Indices)...)}
	case *ast.SliceExpr:
		return syntaxNode{Tag: "slice", Children: []syntaxNode{normalizeNode(x.X), normalizeNode(x.Low), normalizeNode(x.High), normalizeNode(x.Max)}}
	default:
		return normalizePointerExpr(expr)
	}
}

func normalizePointerExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.StarExpr:
		return syntaxNode{Tag: "star", Children: []syntaxNode{normalizeNode(x.X)}}
	default:
		return normalizeLiteralOrTypeExpr(expr)
	}
}

func normalizeLiteralOrTypeExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.ParenExpr:
		return syntaxNode{Tag: "paren", Children: []syntaxNode{normalizeNode(x.X)}}
	case *ast.Ident:
		if x.Name == "true" || x.Name == "false" || x.Name == "nil" {
			return syntaxNode{Tag: "literal/" + x.Name}
		}
		return syntaxNode{Tag: "ident"}
	case *ast.BasicLit:
		return syntaxNode{Tag: "literal/" + x.Kind.String()}
	default:
		return normalizeCompositeOrTypeExpr(expr)
	}
}

func normalizeCompositeOrTypeExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.CompositeLit:
		return syntaxNode{Tag: "composite", Children: append([]syntaxNode{normalizeNode(x.Type)}, exprNodes(x.Elts)...)}
	case *ast.KeyValueExpr:
		return syntaxNode{Tag: "key-value", Children: []syntaxNode{normalizeNode(x.Key), normalizeNode(x.Value)}}
	case *ast.FuncLit:
		return syntaxNode{Tag: "func-lit", Children: []syntaxNode{normalizeFieldList("params", x.Type.Params), normalizeFieldList("results", x.Type.Results), normalizeNode(x.Body)}}
	case *ast.TypeAssertExpr:
		return syntaxNode{Tag: "type-assert", Children: []syntaxNode{normalizeNode(x.X), normalizeNode(x.Type)}}
	default:
		return normalizeTypeExpr(expr)
	}
}

func normalizeTypeExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.ArrayType:
		return syntaxNode{Tag: "array-type", Children: []syntaxNode{normalizeNode(x.Elt)}}
	case *ast.MapType:
		return syntaxNode{Tag: "map-type", Children: []syntaxNode{normalizeNode(x.Key), normalizeNode(x.Value)}}
	case *ast.StructType:
		return normalizeFieldList("struct", x.Fields)
	default:
		return normalizeCallableTypeExpr(expr)
	}
}

func normalizeCallableTypeExpr(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.InterfaceType:
		return normalizeFieldList("interface", x.Methods)
	case *ast.ChanType:
		return syntaxNode{Tag: "chan-type", Children: []syntaxNode{normalizeNode(x.Value)}}
	case *ast.FuncType:
		return syntaxNode{Tag: "func-type", Children: []syntaxNode{normalizeFieldList("params", x.Params), normalizeFieldList("results", x.Results)}}
	case *ast.Ellipsis:
		return syntaxNode{Tag: "ellipsis", Children: []syntaxNode{normalizeNode(x.Elt)}}
	default:
		return syntaxNode{Tag: fmt.Sprintf("%T", expr)}
	}
}

func normalizeDecl(decl ast.Decl) syntaxNode {
	switch x := decl.(type) {
	case *ast.GenDecl:
		children := make([]syntaxNode, 0, len(x.Specs))
		for _, spec := range x.Specs {
			children = append(children, normalizeSpec(spec))
		}
		return syntaxNode{Tag: "gen-decl/" + x.Tok.String(), Children: children}
	default:
		return syntaxNode{Tag: "decl"}
	}
}

func normalizeSpec(spec ast.Spec) syntaxNode {
	switch x := spec.(type) {
	case *ast.ValueSpec:
		return syntaxNode{Tag: "value-spec", Children: append([]syntaxNode{normalizeNode(x.Type)}, exprNodes(x.Values)...)}
	case *ast.TypeSpec:
		return syntaxNode{Tag: "type-spec", Children: []syntaxNode{normalizeNode(x.Type)}}
	default:
		return syntaxNode{Tag: "spec"}
	}
}

func normalizeCallee(expr ast.Expr) syntaxNode {
	switch x := expr.(type) {
	case *ast.Ident:
		return syntaxNode{Tag: "callee"}
	case *ast.SelectorExpr:
		return syntaxNode{Tag: "selector-callee", Children: []syntaxNode{normalizeNode(x.X), {Tag: "member"}}}
	default:
		return normalizeNode(x)
	}
}

func stmtNodes(stmts []ast.Stmt) []syntaxNode {
	return syntaxNodes(stmts)
}

func exprNodes(exprs []ast.Expr) []syntaxNode {
	return syntaxNodes(exprs)
}

func syntaxNodes[T ast.Node](nodes []T) []syntaxNode {
	out := make([]syntaxNode, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, normalizeNode(node))
	}
	return out
}

func normalizeList(tag string, children []syntaxNode) syntaxNode {
	return syntaxNode{Tag: tag, Children: children}
}

func syntaxNodeCount(node syntaxNode) int {
	total := 1
	for _, child := range node.Children {
		total += syntaxNodeCount(child)
	}
	return total
}

func syntaxFingerprints(node syntaxNode) map[string]bool {
	out := map[string]bool{}
	var walk func(syntaxNode)
	walk = func(current syntaxNode) {
		out[serializeSyntax(current)] = true
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(node)
	return out
}

func serializeSyntax(node syntaxNode) string {
	var builder strings.Builder
	writeSyntaxNode(&builder, node)
	return builder.String()
}

func writeSyntaxNode(builder *strings.Builder, node syntaxNode) {
	builder.WriteString("(")
	builder.WriteString(node.Tag)
	for _, child := range node.Children {
		builder.WriteByte(' ')
		writeSyntaxNode(builder, child)
	}
	builder.WriteString(")")
}

func structuralSimilarity(left, right structuralEntry) float64 {
	intersection := 0
	for fingerprint := range left.fingerprints {
		if right.fingerprints[fingerprint] {
			intersection++
		}
	}
	union := len(left.fingerprints)
	for fingerprint := range right.fingerprints {
		if !left.fingerprints[fingerprint] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
