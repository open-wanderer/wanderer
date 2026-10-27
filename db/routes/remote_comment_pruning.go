package routes

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

func omittedRemoteCommentReferences(app core.App, trail *core.Record, listed map[string]bool) []canonicalCommentReference {
	stored := []*core.Record{}
	if err := app.RecordQuery("comments").Select("id", "iri", "author", "trail").AndWhere(dbx.HashExp{"trail": trail.Id}).All(&stored); err != nil {
		return nil
	}
	result := []canonicalCommentReference{}
	for _, comment := range stored {
		iri := comment.GetString("iri")
		if listed[iri] || remoteSyncIRIMatchesOrigin(iri, trail.GetString("iri")) || util.RecordHasLocalIdentity(app, comment) || util.IRIHasRecordIdentity(iri, "comments", comment.Id) {
			continue
		}
		result = append(result, canonicalCommentReference{iri: iri, updated: "canonical-absence", omitted: true})
	}
	return result
}

func removeConfirmedGoneComment(app core.App, trail *core.Record, comment *core.Record, trusted canonicalRemoteComment) error {
	if comment == nil || comment.GetString("trail") != trail.Id || comment.GetString("author") != trusted.author || util.RecordHasLocalIdentity(app, comment) || util.IRIHasRecordIdentity(comment.GetString("iri"), "comments", comment.Id) {
		return nil
	}
	if !remoteSyncRecordCanUpdate(app, comment, trusted.author, trail.Id) {
		return nil
	}
	canonicalRemoteLogCache.invalidateAfterRollback(app, trusted.cacheKey)
	return app.Delete(comment)
}
