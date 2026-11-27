package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/search"
)

// completeRequestedTrailChildren runs only after the parent view is authorized
// and enriched. PocketBase omits empty reverse expansions, but a missing key
// can also mean expansion failed. Independently prove each requested relation
// empty under its ListRule before making it explicit; errors stay inconclusive.
// This fixed, parent-scoped query avoids the generic filtered-list HTTP timing
// guard without disabling it or accepting arbitrary client filters or sorting.
func completeRequestedTrailChildren(e *core.RequestEvent, trail *core.Record) {
	if trail.Id == "" || e.Request.URL.Query().Has("fields") {
		return
	}
	info, err := e.RequestInfo()
	if err != nil {
		return
	}

	// Match the former SDK getList(1, 1, {fields: "id", skipTotal: true,
	// filter: pb.filter("trail = {:trail}", {trail: trail.id}), query: {share}}).
	// In particular @request.context is "default", not "expand", and unrelated
	// parent query parameters must not affect child ListRules. Never mutate the
	// parent request info used by authorization/enrichment.
	proofInfo := info.Clone()
	proofInfo.Context = core.RequestInfoContextDefault
	proofInfo.Method = http.MethodGet
	proofInfo.Body = map[string]any{}
	proofInfo.Query = map[string]string{
		"page": "1", "perPage": "1", "fields": "id", "skipTotal": "true",
		"filter": "trail = " + strconv.Quote(trail.Id),
	}
	if share, ok := info.Query["share"]; ok {
		proofInfo.Query["share"] = share
	}

	expanded := trail.Expand()
	for _, collection := range []string{"waypoints", "summit_logs"} {
		relation := collection + "_via_trail"
		if _, exists := expanded[relation]; exists {
			continue
		}
		requested := false
		for _, path := range strings.Split(info.Query["expand"], ",") {
			path = strings.TrimSpace(path)
			if path == relation || strings.HasPrefix(path, relation+".") {
				requested = true
				break
			}
		}
		if !requested || !trailChildrenProvablyEmpty(e, collection, trail.Id, proofInfo) {
			continue
		}
		if expanded == nil {
			expanded = map[string]any{}
		}
		expanded[relation] = []*core.Record{}
	}
	trail.SetExpand(expanded)
}

func trailChildrenProvablyEmpty(e *core.RequestEvent, name, trailID string, info *core.RequestInfo) bool {
	collection, err := e.App.FindCachedCollectionByNameOrId(name)
	// Conservatively leave locked collections unknown, even for superusers.
	if err != nil || collection == nil || collection.ListRule == nil {
		return false
	}
	query := e.App.RecordQuery(collection).
		WithContext(e.Request.Context()).
		Select("[[" + collection.Name + ".id]]").
		Limit(1)
	resolver := core.NewRecordFieldResolver(e.App, collection, info, true)
	if !info.HasSuperuserAuth() && *collection.ListRule != "" {
		expr, err := search.FilterData(*collection.ListRule).BuildExpr(resolver)
		if err != nil {
			return false
		}
		query.AndWhere(expr)
	}
	// Follow recordsList: build the rule with hidden fields allowed, then the
	// fixed client predicate with ordinary visibility, and apply resolver joins.
	resolver.SetAllowHiddenFields(info.HasSuperuserAuth())
	expr, err := search.FilterData("trail = {:trail}").BuildExpr(resolver, dbx.Params{"trail": trailID})
	if err != nil {
		return false
	}
	query.AndWhere(expr)
	if err := resolver.UpdateQuery(query); err != nil {
		return false
	}
	var id string
	return errors.Is(query.Row(&id), sql.ErrNoRows)
}
