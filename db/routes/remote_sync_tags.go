package routes

import (
	"context"
	"fmt"
	"strings"

	"pocketbase/tagname"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

type remoteSyncTagsKey struct{}

type remoteSyncTagState struct {
	ids      map[string]string
	preserve map[string]bool
}

// Prepare a shared tag set before record writes. A list's trails share one
// collection budget; repeating a tag does not consume another place.
func prepareRemoteSyncTags(app core.App, ctx context.Context, data map[string]any, fallbackAuthor, origin, kind string) (context.Context, error) {
	schema, err := app.FindCachedCollectionByNameOrId("trails")
	if err != nil {
		return ctx, err
	}
	field, ok := schema.Fields.GetByName("tags").(*core.RelationField)
	if !ok {
		return context.WithValue(ctx, remoteSyncTagsKey{}, remoteSyncTagState{ids: map[string]string{}, preserve: map[string]bool{}}), nil
	}
	limit := max(1, field.MaxSelect)
	records := []map[string]any{data}
	if kind == "list" {
		records = nil
		expand, _ := data["expand"].(map[string]any)
		trails, _ := expand["trails"].([]any)
		for _, value := range trails {
			if raw, ok := value.(map[string]any); ok {
				expand, _ := raw["expand"].(map[string]any)
				if tags, _ := expand["tags"].([]any); len(tags) == 0 {
					continue
				}
				iri, _ := raw["iri"].(string)
				if iri == "" {
					id, _ := raw["id"].(string)
					iri = fmt.Sprintf("%s/api/v1/trail/%s", origin, id)
				}
				if util.IsLocalIRI(iri) || !remoteSyncIRIMatchesOrigin(iri, origin) {
					continue
				}
				stored, _ := app.FindFirstRecordByData("trails", "iri", iri)
				if stored != nil && remoteRecordHasLocalAuthor(app, stored) {
					continue
				}
				author, allowed, err := resolveRemoteSyncAuthor(app, ctx, fallbackAuthor, iri, raw)
				if err != nil || !allowed || stored != nil && !remoteSyncRecordCanUpdate(app, stored, author, "") {
					continue
				}
				records = append(records, raw)
			}
		}
		// Tags of another origin's trail come from that origin.
		state := remoteListReferenceStateOf(ctx)
		prepared := map[string]bool{}
		for _, references := range []map[string]remoteListReference{state.confirmed, state.refreshed} {
			for _, reference := range references {
				if !prepared[reference.iri] {
					prepared[reference.iri] = true
					records = append(records, reference.data)
				}
			}
		}
	}
	names := []string{}
	chosen := map[string]bool{}
	skipped := 0
	preserve := map[string]bool{}
	for _, record := range records {
		expand, _ := record["expand"].(map[string]any)
		tags, ok := expand["tags"].([]any)
		if !ok {
			continue
		}
		accepted := make([]any, 0, min(len(tags), limit))
		seen := map[string]bool{}
		for i, value := range tags {
			if len(accepted) >= limit {
				skipped += len(tags) - i
				preserve[remoteSyncTagPlanKey(record)] = true
				break
			}
			raw, ok := value.(map[string]any)
			if !ok {
				continue
			}
			name, ok := raw["name"].(string)
			if !ok {
				continue
			}
			name = tagname.Normalize(name)
			if strings.Trim(name, " ") == "" || seen[name] {
				continue
			}
			if !chosen[name] {
				if len(names) >= remoteSyncChildLimit {
					skipped++
					preserve[remoteSyncTagPlanKey(record)] = true
					continue
				}
				chosen[name] = true
				names = append(names, name)
			}
			seen[name] = true
			accepted = append(accepted, map[string]any{"name": name})
		}
		expand["tags"] = accepted
	}
	if skipped > 0 {
		app.Logger().Warn("remote sync child limit reached", "collection", "tags", "skipped", skipped, "selected", len(names))
	}
	ids := map[string]string{}
	for start := 0; start < len(names); start += remoteSyncWriteBatch {
		if err := ctx.Err(); err != nil {
			return ctx, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		}
		end := min(start+remoteSyncWriteBatch, len(names))
		err := app.RunInTransaction(func(tx core.App) error {
			for _, name := range names[start:end] {
				tag, err := util.ResolveFederatedTag(tx, name)
				if err == nil && tag != nil {
					ids[name] = tag.Id
				}
			}
			return nil
		})
		if err != nil {
			return ctx, err
		}
	}
	return context.WithValue(ctx, remoteSyncTagsKey{}, remoteSyncTagState{ids: ids, preserve: preserve}), nil
}

func remoteSyncTagID(ctx context.Context, name string) string {
	state, _ := ctx.Value(remoteSyncTagsKey{}).(remoteSyncTagState)
	return state.ids[tagname.Normalize(name)]
}

func remoteSyncTagPlanKey(data map[string]any) string {
	if iri, _ := data["iri"].(string); iri != "" {
		return "iri:" + iri
	}
	id, _ := data["id"].(string)
	return "id:" + id
}

func preservePreparedRemoteTags(ctx context.Context, record *core.Record, data map[string]any, ids []string) []string {
	state, prepared := ctx.Value(remoteSyncTagsKey{}).(remoteSyncTagState)
	if !prepared || !state.preserve[remoteSyncTagPlanKey(data)] {
		return ids
	}
	field, ok := record.Collection().Fields.GetByName("tags").(*core.RelationField)
	if !ok {
		return ids
	}
	merged := append([]string(nil), record.GetStringSlice("tags")...)
	seen := map[string]bool{}
	for _, id := range merged {
		seen[id] = true
	}
	for _, id := range ids {
		if !seen[id] && len(merged) < max(1, field.MaxSelect) {
			seen[id] = true
			merged = append(merged, id)
		}
	}
	return merged
}
