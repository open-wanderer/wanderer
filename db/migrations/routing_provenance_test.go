package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestRoutingProvenanceFieldIsBoundedJSON(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := app.Save(core.NewBaseCollection("trails")); err != nil {
		t.Fatalf("create trails collection: %v", err)
	}
	if err := ensureRoutingProvenanceField(app); err != nil {
		t.Fatalf("apply routing provenance migration: %v", err)
	}
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("find trails collection: %v", err)
	}
	field, ok := trails.Fields.GetByName("routing_provenance").(*core.JSONField)
	if !ok {
		t.Fatalf("routing_provenance is not a JSON field: %T", trails.Fields.GetByName("routing_provenance"))
	}
	if field.MaxSize != 256*1024 {
		t.Fatalf("field max size = %d", field.MaxSize)
	}
}
