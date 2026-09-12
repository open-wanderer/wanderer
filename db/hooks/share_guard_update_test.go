package hooks

import (
	"net/http"
	"testing"
)

func TestUpdateShareHandlerThroughRecordsAPI(t *testing.T) {
	for _, objectField := range []string{"trail", "list"} {
		t.Run(objectField, func(t *testing.T) {
			api := newShareUpdateAPITest(t, objectField)
			api.app.OnRecordUpdateRequest(api.shares.Name).BindFunc(UpdateShareTargetHandler(objectField))
			api.app.OnRecordUpdateRequest(api.shares.Name).BindFunc(UpdateShareHandler(api.objects.Name, objectField))
			api.runCases(t, []shareUpdateAPICase{
				{"private actor-only change to remote", api.privateObject, api.localActor, map[string]string{"actor": api.remoteActor.Id}, api.ownerToken, http.StatusBadRequest},
				{"public actor-only change to remote", api.publicObject, api.localActor, map[string]string{"actor": api.remoteActor.Id}, api.ownerToken, http.StatusOK},
				{"private local permission-only change", api.privateObject, api.localActor, map[string]string{"permission": "edit"}, api.ownerToken, http.StatusOK},
				{"explicit unchanged target with permission change", api.privateObject, api.localActor, map[string]string{objectField: api.privateObject.Id, "permission": "edit"}, api.ownerToken, http.StatusOK},
				{"public remote permission-only change", api.publicObject, api.remoteActor, map[string]string{"permission": "edit"}, api.ownerToken, http.StatusOK},
				{"legacy private remote permission-only change", api.privateObject, api.remoteActor, map[string]string{"permission": "edit"}, api.ownerToken, http.StatusBadRequest},
				{"remote target-only change to private", api.publicObject, api.remoteActor, map[string]string{objectField: api.privateObject.Id}, api.ownerToken, http.StatusBadRequest},
				{"change target and recipient together", api.privateObject, api.localActor, map[string]string{objectField: api.publicObject.Id, "actor": api.remoteActor.Id}, api.ownerToken, http.StatusBadRequest},
				{"repair existing private remote share", api.privateObject, api.remoteActor, map[string]string{"actor": api.localActor.Id}, api.ownerToken, http.StatusOK},
				{"invalid recipient uses record validation", api.privateObject, api.localActor, map[string]string{"actor": "missing00000000"}, api.ownerToken, http.StatusBadRequest},
				{"anonymous cannot update permission", api.publicObject, api.remoteActor, map[string]string{"permission": "edit"}, "", http.StatusNotFound},
				{"recipient cannot update owner's share", api.privateObject, api.localActor, map[string]string{"permission": "edit"}, api.strangerToken, http.StatusNotFound},
				{"stranger cannot repair owner's share", api.privateObject, api.remoteActor, map[string]string{"actor": api.localActor.Id}, api.strangerToken, http.StatusNotFound},
			})
		})
	}
}
