package routes

import (
	"net/http"
	"pocketbase/tagname"
	"unicode/utf8"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/search"
)

// TagLookup returns the first visible tag with an exact, already-normalized
// name. The value is bound directly in SQL, including quotes and backslashes.
// Creation remains a separate operation; this lookup does not enforce unique
// tag names or merge historical duplicates.
func TagLookup(e *core.RequestEvent) error {
	collection, err := e.App.FindCachedCollectionByNameOrId("tags")
	if err != nil || collection == nil {
		return e.InternalServerError("Failed to find tags collection.", err)
	}
	info, err := e.RequestInfo()
	if err != nil {
		return e.BadRequestError("Invalid tag lookup request.", err)
	}
	if collection.ListRule == nil && !info.HasSuperuserAuth() {
		return e.ForbiddenError("Only superusers can perform this action.", nil)
	}

	var body struct {
		Name *string `json:"name"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("Invalid tag lookup request.", err)
	}
	if body.Name == nil || tagname.Normalize(*body.Name) != *body.Name || utf8.RuneCountInString(*body.Name) > tagname.MaxLength {
		return e.BadRequestError("Invalid tag name.", nil)
	}

	query := e.App.RecordQuery(collection).
		WithContext(e.Request.Context()).
		AndWhere(dbx.HashExp{collection.Name + ".name": *body.Name}).
		OrderBy("[[" + collection.Name + ".id]]").
		Limit(1)
	if !info.HasSuperuserAuth() && collection.ListRule != nil && *collection.ListRule != "" {
		resolver := core.NewRecordFieldResolver(e.App, collection, info, true)
		expr, err := search.FilterData(*collection.ListRule).BuildExpr(resolver)
		if err != nil {
			return e.InternalServerError("Failed to apply tag list rule.", err)
		}
		query.AndWhere(expr)
		if err := resolver.UpdateQuery(query); err != nil {
			return e.InternalServerError("Failed to apply tag list rule.", err)
		}
	}
	records := []*core.Record{}
	if err := query.All(&records); err != nil {
		return e.InternalServerError("Failed to look up tag.", err)
	}
	if err := apis.EnrichRecords(e, records); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{"items": records})
}
