package commands

import (
	"context"
	"errors"
	"fmt"
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

	loggerWritter := app.Logger.With(
		log.NewValue("channelID", channelID),
		log.NewValue("userID", userID),
	)

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		loggerWritter.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	if inc.Status != model.StatusOpen {
		message := "The incident <#" + inc.ChannelID + "> is already `" + inc.Status + "`.\n" +
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
			loggerWritter.Error(
				ctx,
				log.Trace(),
				log.Reason("PostEphemeralContext"),
				log.NewValue("error", err),
			)

			PostErrorAttachment(ctx, app, channelID, userID, err.Error())
			return err
		}

		return errors.New("Incident is not open for cancel. The current incident status is " + inc.Status)
	}

	description := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Cancel Description",
			Name:        "incident_description",
			Type:        "textarea",
			Placeholder: "Brief description on why you are canceling this incident. eg. Opened by accident",
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
	logWriter := app.Logger.With(
		log.NewValue("userID", incidentDetails.User.ID),
		log.NewValue("channelID", incidentDetails.Channel.ID),
		log.NewValue("description", incidentDetails.Submission["incident_description"]),
		log.NewValue("productChannelID", config.Env.ProductChannelID),
	)

	logWriter.Debug(
		ctx,
		log.Trace(),
		log.Action("running"),
		log.NewValue("incidentDetails", incidentDetails),
	)

	var (
		notifyOnCancel   = config.Env.NotifyOnCancel
		productChannelID = config.Env.ProductChannelID
		userID           = incidentDetails.User.ID
		channelID        = incidentDetails.Channel.ID
		description      = incidentDetails.Submission["incident_description"]
		requestCancel    = model.Incident{
			ChannelID:            channelID,
			DescriptionCancelled: description,
		}
	)

	err := app.IncidentRepository.CancelIncident(ctx, &requestCancel)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("CancelIncident"),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("error", err),
		)
		return err
	}

	card := createCancelCard(inc, inc.ID)

	err = postAndPinBlockMessage(app, channelID, card)

	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Action("postCancelMessage"),
			log.Reason("postAndPinBlockMessage"),
			log.NewValue("error", err),
		)
		return err
	}

	if notifyOnCancel {
		logWriter.Debug(ctx, "Notifying incidents channel about the incident cancelation")

		_, _, err := postBlockMessage(app, productChannelID, card)

		if err != nil {
			logWriter.Error(
				ctx,
				log.Trace(),
				log.Action("notifyOnCancel"),
				log.Reason("postBlockMessage"),
				log.NewValue("error", err),
			)
			return err
		}
	}

	err = app.Client.ArchiveConversationContext(ctx, channelID)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("ArchiveConversationContext"),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	return nil
}

func createCancelCard(incident model.Incident, incidentID int64) []slack.Block {
	title := fmt.Sprintf(":no_entry: *Incident #%d - %s* has been canceled", incidentID, incident.Title)

	bodySlice := []string{}

	bodySlice = append(bodySlice, fmt.Sprintf("*Channel:*\t\t\t\t\t#%s", incident.ChannelName))

	if incident.DescriptionCancelled != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("\n*Description:*\n%s", incident.DescriptionCancelled))
	}

	return createBaseCard(title, bodySlice)
}
