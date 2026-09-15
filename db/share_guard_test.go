package main

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestShareGuardThroughRecordsAPI(t *testing.T) {
	for _, target := range []string{"trail", "list"} {
		t.Run(target, func(t *testing.T) {
			api := newShareTestAPI(t, target+"_share", target, "actor")
			t.Run("create", func(t *testing.T) {
				// Stop allowed requests at the save boundary, before indexing/federation.
				// A rejected request must never reach this hook.
				attempts := 0
				id := api.app.OnRecordCreateExecute(api.shares.Name).BindFunc(func(e *core.RecordEvent) error {
					attempts++
					return router.NewApiError(http.StatusTeapot, "test reached persistence", nil)
				})
				defer api.app.OnRecordCreateExecute(api.shares.Name).Unbind(id)
				for _, test := range []struct {
					name          string
					object, actor *core.Record
					status        int
					attempts      int
				}{
					{"private remote rejected before save", api.privateObject, api.remoteActor, http.StatusBadRequest, 0},
					{"private local reaches save", api.privateObject, api.localActor, http.StatusTeapot, 1},
					{"public remote reaches save", api.publicObject, api.remoteActor, http.StatusTeapot, 1},
				} {
					t.Run(test.name, func(t *testing.T) {
						attempts = 0
						body := map[string]string{target: test.object.Id, "actor": test.actor.Id, "permission": "view"}
						shareRequest(t, api.mux, http.MethodPost, "/api/collections/"+api.shares.Name+"/records", body, api.ownerToken, test.status)
						if attempts != test.attempts {
							t.Errorf("save attempts = %d; want %d", attempts, test.attempts)
						}
						if count, err := api.app.CountRecords(api.shares); err != nil || count != 0 {
							t.Fatalf("stored shares = %d, err = %v; want no saved shares", count, err)
						}
					})
				}
			})
			for _, test := range []struct {
				name          string
				object, actor *core.Record
				patch         map[string]string
				status        int
			}{
				{"private remote recipient rejected atomically", api.privateObject, api.localActor, map[string]string{"actor": api.remoteActor.Id, "permission": "edit"}, http.StatusBadRequest},
				{"public remote recipient allowed", api.publicObject, api.localActor, map[string]string{"actor": api.remoteActor.Id}, http.StatusOK},
				{"public remote permission change allowed", api.publicObject, api.remoteActor, map[string]string{"permission": "edit"}, http.StatusOK},
				{"legacy private remote permission change rejected", api.privateObject, api.remoteActor, map[string]string{"permission": "edit"}, http.StatusBadRequest},
				{"legacy private remote share repaired with local recipient", api.privateObject, api.remoteActor, map[string]string{"actor": api.localActor.Id}, http.StatusOK},
			} {
				t.Run(test.name, func(t *testing.T) {
					share := api.newShare(t, test.object, test.actor.Id)
					api.patch(t, share, test.patch, test.status)
				})
			}
		})
	}
}
