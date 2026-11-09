package util

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// These are memory-only bookkeeping models, not simulated database/App calls.
// Their statement ASTs are bound below to the actual production statements.
func photoCorrectionModel(existing []*core.Record, i int, record, correctedAsset *core.Record, correctedAssets map[string]*core.Record) *core.Record {
	if record.Id == correctedAsset.Id {
		existing[i] = correctedAsset
	}
	record = correctedAsset
	correctedAssets[record.Id] = record
	return record
}

func photoCorrectionRefreshModel(existing []*core.Record, correctedAssets map[string]*core.Record) {
	for i, record := range existing {
		if fresh := correctedAssets[record.Id]; fresh != nil {
			existing[i] = fresh
		}
	}
}

func assertPhotoCorrectionSourceBinding(t *testing.T) {
	t.Helper()
	_, testPath, _, _ := runtime.Caller(0)
	productionPath := filepath.Join(filepath.Dir(testPath), "federated_assets.go")
	source, err := os.ReadFile(productionPath)
	if os.IsNotExist(err) {
		// The no-start isolated export pins the full production source as
		// read-only test data, not a compiled App-dependent Go source file.
		source, err = os.ReadFile(productionPath + ".txt")
	}
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	production, err := parser.ParseFile(fset, productionPath, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	models, err := parser.ParseFile(fset, testPath, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	find := func(file *ast.File, name string) *ast.FuncDecl {
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Name.Name == name {
				return fn
			}
		}
		t.Fatalf("missing source function %s", name)
		return nil
	}
	render := func(statements []ast.Stmt) string {
		var result bytes.Buffer
		for _, statement := range statements {
			if err := format.Node(&result, fset, statement); err != nil {
				t.Fatal(err)
			}
			result.WriteByte('\n')
		}
		return result.String()
	}
	reconcile := find(production, "ReconcileFederatedPhotoAssets")
	var correction []ast.Stmt
	ast.Inspect(reconcile.Body, func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, statement := range block.List {
			assign, ok := statement.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				continue
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			fn, ok := call.Fun.(*ast.Ident)
			if ok && fn.Name == "correctFederatedTargetPhotoAsset" {
				// Skip only the call and its error-return guard. Bind every
				// following statement in this production correction block.
				if i+2 > len(block.List) {
					t.Fatal("missing correction error guard")
				}
				correction = block.List[i+2:]
			}
		}
		return true
	})
	model := find(models, "photoCorrectionModel").Body.List
	if len(correction) == 0 || render(correction) != render(model[:len(model)-1]) {
		t.Fatalf("correction model is not bound to production:\n%s", render(correction))
	}
	var refresh ast.Stmt
	existingLoops := 0
	// Both loops now live inside the first operation scope. Bind the actual
	// final loop without replacing or weakening any bookkeeping assertions.
	ast.Inspect(reconcile.Body, func(node ast.Node) bool {
		if loop, ok := node.(*ast.RangeStmt); ok {
			if name, ok := loop.X.(*ast.Ident); ok && name.Name == "existing" {
				existingLoops++
				refresh = loop
			}
		}
		return true
	})
	if existingLoops != 2 || refresh == nil || render([]ast.Stmt{refresh}) != render(find(models, "photoCorrectionRefreshModel").Body.List) {
		t.Fatal("refresh model is not bound to production")
	}
}

func TestPhotoCorrectionBookkeepingKeepsKeeperOwnPosition(t *testing.T) {
	assertPhotoCorrectionSourceBinding(t)
	const identity = "https://remote.example/api/files/trails/id/photo.jpg"
	for _, sourceFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "source-before-keeper", false: "keeper-before-source"}[sourceFirst], func(t *testing.T) {
			source := photoIndexRecord("wrong-author", "federation", "photo.jpg", identity)
			source.Id = "source"
			keeper := photoIndexRecord("author", "federation", "photo.jpg", identity)
			keeper.Id = "keeper"
			fresh := keeper.Fresh() // memory snapshot only, NOT a DB reload
			fresh.Set("metadata", map[string]any{"source_file": "refreshed.jpg"})
			existing, sourcePosition, keeperPosition := []*core.Record{source, keeper}, 0, 1
			if !sourceFirst {
				existing, sourcePosition, keeperPosition = []*core.Record{keeper, source}, 1, 0
			}
			corrected := map[string]*core.Record{}
			local := photoCorrectionModel(existing, sourcePosition, source, fresh, corrected)
			photoCorrectionRefreshModel(existing, corrected)
			if local != fresh || corrected[keeper.Id] != fresh {
				t.Fatal("managed-map local binding or keeper freshness lost")
			}
			if existing[sourcePosition] != source || existing[keeperPosition] != fresh {
				t.Error("different-ID merge duplicated/moved keeper into source position")
			}
			p1 := FederatedPhoto{CanonicalID: "https://remote.example/api/v1/assets/p1", SourceName: "photo.jpg", IsThumbnail: true}
			p2 := FederatedPhoto{CanonicalID: "https://remote.example/api/v1/assets/p2", SourceName: p1.SourceName}
			index := NewFederatedPhotoAssetIndex(existing, "author")
			position, match := index.FirstMatch(p1)
			if position != keeperPosition || match != fresh {
				t.Errorf("first adoption position=%d, want keeper's own position %d", position, keeperPosition)
			}
			if match == nil {
				t.Fatal("missing keeper")
			}
			adopted := match.Fresh()
			adopted.Set("external_id", p1.CanonicalID)
			index.Replace(position, adopted)
			if secondPosition, second := index.FirstMatch(p2); second != nil {
				t.Errorf("P2 matched stale duplicate keeper at %d; P1 could be overwritten/merged", secondPosition)
			}
			// The DB identity-holder/link/cover behavior is deliberately NOT
			// modeled here; a separate integration test is compile-only.
		})
	}
}

func TestPhotoCorrectionBookkeepingSameIDAndOriginalOrder(t *testing.T) {
	assertPhotoCorrectionSourceBinding(t)
	source := photoIndexRecord("wrong-author", "federation", "", "https://remote.example/api/files/id/photo.jpg")
	source.Id = "source"
	fresh := source.Fresh()
	fresh.Set("author", "author")
	existing := []*core.Record{source}
	corrected := map[string]*core.Record{}
	photoCorrectionModel(existing, 0, source, fresh, corrected)
	if existing[0] != fresh {
		t.Fatal("same-ID correction did not refresh its own position")
	}
	keeper := fresh.Fresh()
	keeper.Id = "keeper"
	independent := photoIndexRecord("author", "", "photo.jpg", "")
	existing = []*core.Record{source, independent, keeper}
	corrected = map[string]*core.Record{}
	photoCorrectionModel(existing, 0, source, keeper.Fresh(), corrected)
	photoCorrectionRefreshModel(existing, corrected)
	photo := FederatedPhoto{CanonicalID: "https://remote.example/api/v1/assets/p1", SourceName: "photo.jpg"}
	if position, asset := NewFederatedPhotoAssetIndex(existing, "author").FirstMatch(photo); position != 1 || asset != independent {
		t.Fatal("different-ID merge changed original first-match order")
	}
}
