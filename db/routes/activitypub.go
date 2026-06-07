package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"pocketbase/federation"
	"pocketbase/util"
	"strconv"
	"strings"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func ActivitypubActor(e *core.RequestEvent) error {
	resource := e.Request.URL.Query().Get("resource")
	resource = strings.TrimPrefix(resource, "acct:")

	iri := e.Request.URL.Query().Get("iri")
	follows := e.Request.URL.Query().Get("follows") == "true"

	var userActor *core.Record
	var err error
	if e.Auth != nil {
		userActor, err = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}
	}
	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	var actor *core.Record
	if resource != "" {
		actor, err = federation.GetActorByHandle(e.App, ctx, resource, follows)
	} else {
		actor, err = federation.GetActorByIRI(e.App, ctx, iri, follows)
	}
	if err != nil && actor == nil {
		if strings.HasPrefix(err.Error(), "webfinger") {
			return e.NotFoundError("Not found", err)
		}
		return err
	} else if err != nil && actor != nil {
		if errors.Is(err, federation.ErrProfilePrivate) {
			// this is our own profile
			if e.Auth != nil && actor.GetString("user") == e.Auth.Id {
				return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})
			} else {
				return e.JSON(http.StatusNotFound, map[string]any{"error": "profile is private"})
			}
		}
		// we could not fetch the remote actor so we return our local cached copy
		return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})
}

func ActivitypubActivityProcess(e *core.RequestEvent) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	var activity pub.Activity
	err = activity.UnmarshalJSON(body)
	if err != nil {
		return err
	}

	inbox := fmt.Sprintf("%s%s", origin, e.Request.Header.Get("X-Forwarded-Path"))

	recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)
	if err != nil {
		return err
	}

	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", activity.Actor.GetID().String())
	if err != nil {
		if err == sql.ErrNoRows {
			ctx, err := util.GetSafeActorContext(e.Request, recipient)
			if err != nil {
				return err
			}
			actor, err = federation.GetActorByIRI(e.App, ctx, activity.Actor.GetID().String(), false)
			if err != nil {
				return err
			}
		} else {
			return err

		}
	}

	verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))
	if err != nil || !verified {
		e.App.Logger().Error(err.Error())
		return e.UnauthorizedError("Invalid http signature", err)
	}

	switch activity.Type {
	case pub.FollowType:
		err = federation.ProcessFollowActivity(e.App, actor, activity)
	case pub.AcceptType:
		err = federation.ProcessAcceptActivity(e.App, actor, activity)
	case pub.UndoType:
		err = federation.ProcessUndoActivity(e.App, actor, activity)
	case pub.UpdateType:
		fallthrough
	case pub.CreateType:
		err = federation.ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity)
	case pub.DeleteType:
		err = federation.ProcessDeleteActivity(e.App, actor, activity)
	case pub.AnnounceType:
		err = federation.ProcessAnnounceActivity(e.App, actor, activity)
	case pub.LikeType:
		err = federation.ProcessLikeActivity(e.App, actor, activity)
	}
	return e.JSON(http.StatusOK, err)
}

func ActivitypubActorFollow(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	followType := e.Request.PathValue("follow")
	page := e.Request.URL.Query().Get("page")
	intPage := 0

	if page != "" {
		var err error
		intPage, err = strconv.Atoi(page)
		if err != nil {
			return err
		}
	}

	actor, err := e.App.FindRecordById("activitypub_actors", id)
	if err != nil {
		return err
	}

	var userActor *core.Record
	if e.Auth != nil {
		userActor, err = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	url := actor.GetString(followType)

	if url == "" {
		return e.BadRequestError("unknown type: "+followType, nil)
	}
	collection, err := federation.FetchCollection(e.App, ctx, fmt.Sprintf("%s?page=%d", url, intPage))
	if err != nil {
		if errors.Is(err, federation.ErrProfilePrivate) {
			return e.JSON(http.StatusNotFound, map[string]any{"error": "profile is private"})
		} else if errors.Is(err, util.ErrRateLimited) {
			return e.TooManyRequestsError("Too many requests", err)
		}
		return err
	}
	return e.JSON(http.StatusOK, collection)
}

func ActivitypubTrail(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	trail, err := e.App.FindRecordById("trails", id)
	if err != nil {
		return err
	}

	trailObject, err := util.ObjectFromTrail(e.App, trail, nil)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, trailObject)
}

func ActivitypubComment(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	comment, err := e.App.FindRecordById("comments", id)
	if err != nil {
		return err
	}

	commentObject, err := util.ObjectFromComment(e.App, comment, nil)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, commentObject)
}
