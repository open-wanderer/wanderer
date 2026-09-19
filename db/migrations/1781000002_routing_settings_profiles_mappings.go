package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		for _, raw := range []string{
			routingSettingsCollectionJSON,
			routingProfilesCollectionJSON,
			routingProfileMappingsCollectionJSON,
		} {
			if err := createRoutingCollection(app, raw); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{
			"routing_profile_mappings",
			"routing_profiles",
			"routing_settings",
		} {
			if collection, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func createRoutingCollection(app core.App, raw string) error {
	var collection core.Collection
	if err := json.Unmarshal([]byte(raw), &collection); err != nil {
		return err
	}
	existing, err := app.FindCollectionByNameOrId(collection.Name)
	if err == nil {
		return fmt.Errorf(
			"routing collection %q already exists with id %q",
			collection.Name,
			existing.Id,
		)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check routing collection %q: %w", collection.Name, err)
	}
	return app.Save(&collection)
}

const routingSettingsCollectionJSON = `{
	"id": "rt_settings001",
	"name": "routing_settings",
	"type": "base",
	"system": false,
	"listRule": "scope = 'builtin' || scope = 'admin' || user = @request.auth.id",
	"viewRule": "scope = 'builtin' || scope = 'admin' || user = @request.auth.id",
	"createRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"updateRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"deleteRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"fields": [
		{"id":"rts_id","name":"id","type":"text","system":true,"required":true,"primaryKey":true,"autogeneratePattern":"[a-z0-9]{15}","min":15,"max":15,"pattern":"^[a-z0-9]+$"},
		{"id":"rts_scope","name":"scope","type":"select","required":true,"maxSelect":1,"values":["builtin","admin","user"]},
		{"id":"rts_user","name":"user","type":"relation","required":false,"collectionId":"_pb_users_auth_","cascadeDelete":true,"maxSelect":1},
		{"id":"rts_config","name":"config","type":"json","required":false,"maxSize":2000000},
		{"id":"rts_created","name":"created","type":"autodate","onCreate":true,"onUpdate":false},
		{"id":"rts_updated","name":"updated","type":"autodate","onCreate":true,"onUpdate":true}
	],
	"indexes": [
		"CREATE INDEX ` + "`" + `idx_routing_settings_scope_user` + "`" + ` ON ` + "`" + `routing_settings` + "`" + ` (` + "`" + `scope` + "`" + `, ` + "`" + `user` + "`" + `)"
	]
}`

const routingProfilesCollectionJSON = `{
	"id": "rt_profiles01",
	"name": "routing_profiles",
	"type": "base",
	"system": false,
	"listRule": "scope = 'admin' || user = @request.auth.id",
	"viewRule": "scope = 'admin' || user = @request.auth.id",
	"createRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"updateRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"deleteRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"fields": [
		{"id":"rtp_id","name":"id","type":"text","system":true,"required":true,"primaryKey":true,"autogeneratePattern":"[a-z0-9]{15}","min":15,"max":15,"pattern":"^[a-z0-9]+$"},
		{"id":"rtp_scope","name":"scope","type":"select","required":true,"maxSelect":1,"values":["admin","user"]},
		{"id":"rtp_user","name":"user","type":"relation","required":false,"collectionId":"_pb_users_auth_","cascadeDelete":true,"maxSelect":1},
		{"id":"rtp_plugin","name":"plugin_id","type":"text","required":true,"min":1,"max":96,"pattern":"^[a-z0-9][a-z0-9_-]*$"},
		{"id":"rtp_key","name":"key","type":"text","required":true,"min":1,"max":128},
		{"id":"rtp_name","name":"name","type":"text","required":true,"min":1,"max":256},
		{"id":"rtp_kind","name":"kind","type":"select","required":true,"maxSelect":1,"values":["custom_file","generated","native_config"]},
		{"id":"rtp_mode","name":"mode","type":"select","required":true,"maxSelect":1,"values":["foot","bike","motor","mixed","other"]},
		{"id":"rtp_content","name":"content_base64","type":"text","required":false,"max":349528},
		{"id":"rtp_ctype","name":"content_type","type":"text","required":false,"max":128},
		{"id":"rtp_meta","name":"metadata","type":"json","required":false,"maxSize":2000000},
		{"id":"rtp_native","name":"native_config","type":"json","required":false,"maxSize":2000000},
		{"id":"rtp_enabled","name":"enabled","type":"bool","required":false},
		{"id":"rtp_created","name":"created","type":"autodate","onCreate":true,"onUpdate":false},
		{"id":"rtp_updated","name":"updated","type":"autodate","onCreate":true,"onUpdate":true}
	],
	"indexes": [
		"CREATE INDEX ` + "`" + `idx_routing_profiles_scope_user_plugin` + "`" + ` ON ` + "`" + `routing_profiles` + "`" + ` (` + "`" + `scope` + "`" + `, ` + "`" + `user` + "`" + `, ` + "`" + `plugin_id` + "`" + `)"
	]
}`

const routingProfileMappingsCollectionJSON = `{
	"id": "rt_mappings01",
	"name": "routing_profile_mappings",
	"type": "base",
	"system": false,
	"listRule": "scope = 'admin' || user = @request.auth.id",
	"viewRule": "scope = 'admin' || user = @request.auth.id",
	"createRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"updateRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"deleteRule": "@request.auth.id != '' && scope = 'user' && user = @request.auth.id",
	"fields": [
		{"id":"rtm_id","name":"id","type":"text","system":true,"required":true,"primaryKey":true,"autogeneratePattern":"[a-z0-9]{15}","min":15,"max":15,"pattern":"^[a-z0-9]+$"},
		{"id":"rtm_scope","name":"scope","type":"select","required":true,"maxSelect":1,"values":["admin","user"]},
		{"id":"rtm_user","name":"user","type":"relation","required":false,"collectionId":"_pb_users_auth_","cascadeDelete":true,"maxSelect":1},
		{"id":"rtm_category","name":"category","type":"text","required":true,"min":1,"max":256},
		{"id":"rtm_subcategory","name":"subcategory","type":"text","required":false,"max":256},
		{"id":"rtm_plugin","name":"plugin_id","type":"text","required":true,"min":1,"max":96,"pattern":"^[a-z0-9][a-z0-9_-]*$"},
		{"id":"rtm_instance","name":"instance_id","type":"text","required":false,"max":64},
		{"id":"rtm_pkey","name":"native_profile_key","type":"text","required":false,"max":128},
		{"id":"rtm_profile","name":"profile","type":"relation","required":false,"collectionId":"rt_profiles01","cascadeDelete":false,"maxSelect":1},
		{"id":"rtm_prefs","name":"preferences","type":"json","required":false,"maxSize":2000000},
		{"id":"rtm_native","name":"native_config","type":"json","required":false,"maxSize":2000000},
		{"id":"rtm_created","name":"created","type":"autodate","onCreate":true,"onUpdate":false},
		{"id":"rtm_updated","name":"updated","type":"autodate","onCreate":true,"onUpdate":true}
	],
	"indexes": [
		"CREATE INDEX ` + "`" + `idx_routing_mappings_scope_user_plugin_instance_category` + "`" + ` ON ` + "`" + `routing_profile_mappings` + "`" + ` (` + "`" + `scope` + "`" + `, ` + "`" + `user` + "`" + `, ` + "`" + `plugin_id` + "`" + `, ` + "`" + `instance_id` + "`" + `, ` + "`" + `category` + "`" + `, ` + "`" + `subcategory` + "`" + `)"
	]
}`
