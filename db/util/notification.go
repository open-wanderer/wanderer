package util

import (
	"net/mail"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

type NotificationType string

const (
	TrailShare       NotificationType = "trail_share"
	ListShare        NotificationType = "list_share"
	NewFollower      NotificationType = "new_follower"
	TrailComment     NotificationType = "trail_comment"
	TrailLike        NotificationType = "trail_like"
	SummitLogCreate  NotificationType = "summit_log_create"
	CommentMention   NotificationType = "comment_mention"
	TrailMention     NotificationType = "trail_mention"
	SummitLogMention NotificationType = "summit_log_mention"
)

type Notification struct {
	Type     NotificationType  `json:"type"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Seen     bool              `json:"seen"`
	Author   string            `json:"author"`
}

type NotificationSettings struct {
	Web   bool `json:"web"`
	Email bool `json:"email"`
}

func getNotificationPermissions(app core.App, user string, notificationType NotificationType) (*NotificationSettings, error) {
	settings, err := app.FindFirstRecordByFilter("settings", "user={:user}", dbx.Params{"user": user})
	if err != nil {
		return nil, err
	}
	var notificationPreferences map[NotificationType]NotificationSettings

	err = settings.UnmarshalJSONField("notifications", &notificationPreferences)
	if err != nil {
		return nil, err
	}

	settingsForType, exists := notificationPreferences[notificationType]
	if !exists {
		return &NotificationSettings{
			Web:   true,
			Email: true,
		}, nil
	}

	return &settingsForType, nil
}

func SendNotification(app core.App, notification Notification, recipient *core.Record) error {
	if notification.Author == recipient.Id {
		return nil
	}
	if !recipient.GetBool("is_local") {
		return nil
	}
	permissions, err := getNotificationPermissions(app, recipient.GetString("user"), notification.Type)
	if err != nil {
		return err
	}

	notifications, err := app.FindCollectionByNameOrId("notifications")
	if err != nil {
		return err
	}

	if permissions.Web {
		n := core.NewRecord(notifications)
		n.Set("type", string(notification.Type))
		n.Set("metadata", notification.Metadata)
		n.Set("seen", notification.Seen)
		n.Set("recipient", recipient.Id)
		n.Set("author", notification.Author)

		if err := app.Save(n); err != nil {
			return err
		}
	}

	if permissions.Email {
		recipientActor, err := app.FindRecordById("activitypub_actors", recipient.Id)
		if err != nil {
			return err
		}
		recipientUser, err := app.FindRecordById("users", recipientActor.GetString("user"))
		if err != nil {
			return err
		}
		authorActor, err := app.FindRecordById("activitypub_actors", notification.Author)
		if err != nil {
			return err
		}
		html, err := GenerateHTML(app.Settings().Meta.AppURL, recipientActor.GetString("username"), authorActor.GetString("username"), notification.Type, notification.Metadata)
		if err != nil {
			return err
		}

		message := &mailer.Message{
			From: mail.Address{
				Address: app.Settings().Meta.SenderAddress,
				Name:    app.Settings().Meta.SenderName,
			},
			To:      []mail.Address{{Address: recipientUser.Email()}},
			Subject: "wanderer - New Notification",
			HTML:    html,
		}

		return app.NewMailClient().Send(message)
	}
	return nil
}
