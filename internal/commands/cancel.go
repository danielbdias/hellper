package commands

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"hellper/internal/app"
	"hellper/internal/bot"
	"hellper/internal/config"
	"hellper/internal/log"
	"hellper/internal/model"

	"github.com/slack-go/slack"
)

// OpenCancelIncidentDialog opens a dialog on Slack, so the user can cancel an incident
func OpenCancelIncidentDialog(
	ctx context.Context,
	app *app.App,
	channelID string,
	userID string,
	triggerID string,
) error {

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("channelID", channelID),
			log.NewValue("userID", userID),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	if inc.Status != model.StatusOpen {
		message := "The incident <#" + inc.ChannelId + "> is already `" + inc.Status + "`.\n" +
			"Only a `open` incident can be canceled."

		var messageText strings.Builder
		messageText.WriteString(message)

		attch := slack.Attachment{
			Pretext:  "",
			Fallback: messageText.String(),
			Text:     message,
			Color:    "#ff8c00",
			Fields:   []slack.AttachmentField{},
		}

		_, err = app.Client.PostEphemeralContext(ctx, channelID, userID, slack.MsgOptionAttachments(attch))
		if err != nil {
			app.Logger.Error(
				ctx,
				log.Trace(),
				log.Reason("PostEphemeralContext"),
				log.NewValue("channelID", channelID),
				log.NewValue("userID", userID),
				log.NewValue("error", err),
			)

			PostErrorAttachment(ctx, app, channelID, userID, err.Error())
			return err
		}

		return errors.New("Incident is not open for cancel. The current incident status is " + inc.Status)
	}

	description := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Description",
			Name:        "incident_description",
			Type:        "textarea",
			Placeholder: "Description eg. We are canceling the suspected incident",
			Optional:    false,
		},
		MaxLength: 500,
	}

	dialog := slack.Dialog{
		CallbackID:     "inc-cancel",
		Title:          "Cancel an Incident",
		SubmitLabel:    "Ok",
		NotifyOnCancel: false,
		Elements: []slack.DialogElement{
			description,
		},
	}

	return app.Client.OpenDialog(triggerID, dialog)
}

// CancelIncidentByDialog cancels an incident after receiving data from a Slack dialog
func CancelIncidentByDialog(
	ctx context.Context,
	app *app.App,
	incidentDetails bot.DialogSubmission,
) error {
	app.Logger.Info(
		ctx,
		log.Trace(),
		log.Action("running"),
		log.NewValue("incidentDetails", incidentDetails),
	)

	var (
		supportTeam      = config.Env.SupportTeam
		notifyOnCancel   = config.Env.NotifyOnCancel
		productChannelID = config.Env.ProductChannelID
		userID           = incidentDetails.User.ID
		channelID        = incidentDetails.Channel.ID
		description      = incidentDetails.Submission.IncidentDescription
		requestCancel    = model.Incident{
			ChannelId:            channelID,
			DescriptionCancelled: description,
		}
	)

	err := app.IncidentRepository.CancelIncident(ctx, &requestCancel)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("CancelIncident"),
			log.NewValue("channelID", channelID),
			log.NewValue("userID", userID),
			log.NewValue("description", description),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("channelID", channelID),
			log.NewValue("error", err),
		)
		return err
	}

	attachment := createCancelAttachment(inc, userID)
	message := "An Incident has been canceled by <@" + userID + "> *cc:* <!subteam^" + supportTeam + ">"

	err = postAndPinMessage(
		app,
		channelID,
		message,
		attachment,
	)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("postAndPinMessage"),
			log.NewValue("channelID", channelID),
			log.NewValue("userID", userID),
			log.NewValue("attachment", attachment),
			log.NewValue("error", err),
		)
		return err
	}

	if notifyOnCancel {
		err := postAndPinMessage(
			app,
			productChannelID,
			message,
			attachment,
		)
		if err != nil {
			app.Logger.Error(
				ctx,
				log.Trace(),
				log.Reason("postAndPinMessage"),
				log.NewValue("channelID", channelID),
				log.NewValue("productChannelID", productChannelID),
				log.NewValue("userID", userID),
				log.NewValue("attachment", attachment),
				log.NewValue("error", err),
			)
			return err
		}
	}

	err = app.Client.ArchiveConversationContext(ctx, channelID)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("ArchiveConversationContext"),
			log.NewValue("channelID", channelID),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	return nil
}

func createCancelAttachment(inc model.Incident, userID string) slack.Attachment {
	var messageText strings.Builder

	messageText.WriteString("An Incident has been canceled by <@" + userID + ">\n\n")
	messageText.WriteString("*Channel:* <#" + inc.ChannelId + ">\n")
	messageText.WriteString("*Description:* `" + inc.DescriptionCancelled + "`\n\n")

	return slack.Attachment{
		Pretext:  "",
		Fallback: messageText.String(),
		Text:     "",
		Color:    "#EDA248",
		Fields: []slack.AttachmentField{
			{
				Title: "Incident ID",
				Value: strconv.FormatInt(inc.Id, 10),
			},
			{
				Title: "Incident Channel",
				Value: "<#" + inc.ChannelId + ">",
			},
			{
				Title: "Incident Title",
				Value: inc.Title,
			},
			{
				Title: "Description",
				Value: "```" + inc.DescriptionCancelled + "```",
			},
		},
	}
}
