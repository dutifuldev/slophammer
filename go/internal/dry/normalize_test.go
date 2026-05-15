package dry

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestNormalizeStatementVariants(t *testing.T) {
	assertTags(t, []tagCase[ast.Stmt]{
		{"block", &ast.BlockStmt{}},
		{"if", &ast.IfStmt{Cond: ast.NewIdent("ok"), Body: &ast.BlockStmt{}}},
		{"for", &ast.ForStmt{Body: &ast.BlockStmt{}}},
		{"range", &ast.RangeStmt{X: ast.NewIdent("xs"), Body: &ast.BlockStmt{}}},
		{"switch", &ast.SwitchStmt{Body: &ast.BlockStmt{}}},
		{"type-switch", &ast.TypeSwitchStmt{Body: &ast.BlockStmt{}}},
		{"select", &ast.SelectStmt{Body: &ast.BlockStmt{}}},
		{"case", &ast.CaseClause{}},
		{"comm", &ast.CommClause{}},
		{"assign/=", &ast.AssignStmt{Tok: token.ASSIGN}},
		{"decl", &ast.DeclStmt{}},
		{"expr-stmt", &ast.ExprStmt{X: ast.NewIdent("x")}},
		{"return", &ast.ReturnStmt{}},
		{"branch/break", &ast.BranchStmt{Tok: token.BREAK}},
		{"go", &ast.GoStmt{Call: &ast.CallExpr{Fun: ast.NewIdent("fn")}}},
		{"defer", &ast.DeferStmt{Call: &ast.CallExpr{Fun: ast.NewIdent("fn")}}},
		{"send", &ast.SendStmt{Chan: ast.NewIdent("ch"), Value: ast.NewIdent("value")}},
		{"incdec/++", &ast.IncDecStmt{Tok: token.INC}},
		{"label", &ast.LabeledStmt{Stmt: &ast.EmptyStmt{}}},
		{"empty", &ast.EmptyStmt{}},
		{"bad-stmt", &ast.BadStmt{}},
	}, normalizeNode)
}

func TestNormalizeExpressionVariants(t *testing.T) {
	assertTags(t, []tagCase[ast.Expr]{
		{"binary/+", &ast.BinaryExpr{X: ast.NewIdent("x"), Y: ast.NewIdent("y"), Op: token.ADD}},
		{"unary/!", &ast.UnaryExpr{X: ast.NewIdent("x"), Op: token.NOT}},
		{"call", &ast.CallExpr{Fun: ast.NewIdent("fn")}},
		{"selector", &ast.SelectorExpr{X: ast.NewIdent("pkg"), Sel: ast.NewIdent("Member")}},
		{"index", &ast.IndexExpr{X: ast.NewIdent("xs"), Index: ast.NewIdent("i")}},
		{"index-list", &ast.IndexListExpr{X: ast.NewIdent("xs")}},
		{"slice", &ast.SliceExpr{X: ast.NewIdent("xs")}},
		{"star", &ast.StarExpr{X: ast.NewIdent("x")}},
		{"paren", &ast.ParenExpr{X: ast.NewIdent("x")}},
		{"composite", &ast.CompositeLit{}},
		{"key-value", &ast.KeyValueExpr{}},
		{"func-lit", &ast.FuncLit{Type: &ast.FuncType{}, Body: &ast.BlockStmt{}}},
		{"type-assert", &ast.TypeAssertExpr{X: ast.NewIdent("x")}},
		{"literal/true", ast.NewIdent("true")},
		{"ident", ast.NewIdent("name")},
		{"literal/INT", &ast.BasicLit{Kind: token.INT}},
		{"array-type", &ast.ArrayType{}},
		{"map-type", &ast.MapType{}},
		{"struct", &ast.StructType{}},
		{"interface", &ast.InterfaceType{}},
		{"chan-type", &ast.ChanType{}},
		{"func-type", &ast.FuncType{}},
		{"ellipsis", &ast.Ellipsis{}},
	}, normalizeNode)
}

func TestNormalizeFallbacks(t *testing.T) {
	if got := normalizeNode(nil).Tag; got != "nil" {
		t.Fatalf("nil tag = %q", got)
	}
	if got := normalizeNode(&ast.Comment{}).Tag; got != "*ast.Comment" {
		t.Fatalf("fallback tag = %q", got)
	}
	if got := normalizeSimpleStmt(nil).Tag; got != "<nil>" {
		t.Fatalf("simple fallback tag = %q", got)
	}
	if got := normalizeTypeExpr(ast.NewIdent("x")).Tag; got != "*ast.Ident" {
		t.Fatalf("type fallback tag = %q", got)
	}
}

type tagCase[T ast.Node] struct {
	tag  string
	node T
}

func assertTags[T ast.Node](t *testing.T, cases []tagCase[T], normalize func(ast.Node) syntaxNode) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.tag, func(t *testing.T) {
			if got := normalize(tc.node).Tag; got != tc.tag {
				t.Fatalf("tag = %q, want %q", got, tc.tag)
			}
		})
	}
}
