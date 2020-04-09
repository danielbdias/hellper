package commands

import (
	"context"
	"hellper/internal/bot"
	"hellper/internal/config"
	"hellper/internal/log"
	"hellper/internal/model"
	"strings"

	"github.com/slack-go/slack"
)

//ListOpenIncidents get the currently opened incidents and return the channel_name of each one of them.
func ListOpenIncidents(ctx context.Context, client bot.Client, logger log.Logger, repository model.Repository, event TriggerEvent) {

	incidents, err := repository.ListActiveIncidents(ctx)
	if err != nil {
		logger.Error(
			ctx,
			"command/list_open.ListOpenIncidents ListActiveIncidents error",
			log.NewValue("event", event),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, client, logger, event.Channel, event.User, err.Error())
	}

	logger.Info(
		ctx,
		"command/list_open.ListOpenIncidents",
		log.NewValue("event", event),
		log.NewValue("incidents", incidents),
	)

	if len(incidents) == 0 {
		postMessage(client, event.Channel, config.Env.Messages.NoListOpenIncidents)
	} else {
		attachment := createListOpenAttachment(incidents)
		postMessage(client, event.Channel, "", attachment)
	}
}

func createListOpenAttachment(incidents []model.Incident) slack.Attachment {
	var messageText strings.Builder
	messageText.WriteString("Current open incidents:")

	var fields []slack.AttachmentField

	for _, inc := range incidents {
		messageText.WriteString("- <#" + inc.ChannelId + ">\n")

		fields = append(
			fields,
			slack.AttachmentField{
				Value: "- <#" + inc.ChannelId + ">",
			},
		)
	}

	return slack.Attachment{
		Pretext:  "Current open incidents:",
		Fallback: messageText.String(),
		Text:     "",
		Color:    "#000000",
		Fields:   fields,
	}
}
