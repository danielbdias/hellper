package commands

import (
	"context"
	"fmt"
	"hellper/internal/app"
	"hellper/internal/concurrence"
	"strings"
	"sync"
	"time"

	"hellper/internal/bot"
	"hellper/internal/config"
	"hellper/internal/log"
	"hellper/internal/model"

	"github.com/slack-go/slack"
)

// CloseIncidentDialog opens a dialog on Slack, so the user can close an incident
func CloseIncidentDialog(ctx context.Context, app *app.App, channelID, userID, triggerID string) error {
	var (
		startTimestampAsText = ""
	)

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("channelID", channelID),
			log.NewValue("error", err),
		)

		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	if inc.StartTimestamp != nil {
		startTimestampAsText = inc.StartTimestamp.Format(dateLayout)
	}

	rootCause := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Root Cause",
			Name:        "root_cause",
			Type:        "textarea",
			Placeholder: "Incident root cause description.",
			Optional:    false,
		},
		MaxLength: 500,
		Value:     inc.RootCause,
	}

	startDate := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Incident Start Date",
			Name:        "init_date",
			Type:        "text",
			Placeholder: dateLayout,
			Optional:    false,
		},
		Hint:  "The time is in format " + dateLayout,
		Value: startTimestampAsText,
	}

	severityLevel := &slack.DialogInputSelect{
		DialogInput: slack.DialogInput{
			Label:       "Severity Level",
			Name:        "severity_level",
			Type:        "select",
			Placeholder: "Set the severity level",
			Optional:    true,
		},
		Options:      getDialogOptionsWithSeverityLevels(),
		OptionGroups: []slack.DialogOptionGroup{},
		Value:        fmt.Sprintf("%d", inc.SeverityLevel),
	}

	dialogElements := []slack.DialogElement{
		rootCause,
	}

	if inc.StartTimestamp == nil {
		dialogElements = append(dialogElements, startDate)
	}

	dialogElements = append(dialogElements, severityLevel)

	dialog := slack.Dialog{
		CallbackID:     "inc-close",
		Title:          "Close an Incident",
		SubmitLabel:    "Close",
		NotifyOnCancel: false,
		Elements:       dialogElements,
	}

	return app.Client.OpenDialog(triggerID, dialog)
}

// CloseIncidentByDialog closes an incident after receiving data from a Slack dialog
func CloseIncidentByDialog(ctx context.Context, app *app.App, incidentDetails bot.DialogSubmission) error {
	app.Logger.Debug(
		ctx,
		"command/close.CloseIncidentByDialog",
		log.NewValue("incident_close_details", incidentDetails),
	)

	var (
		now              = time.Now().UTC()
		channelID        = incidentDetails.Channel.ID
		userID           = incidentDetails.User.ID
		submissions      = incidentDetails.Submission
		startDateText    = submissions["init_date"]
		severityLevel    = submissions["severity_level"]
		rootCause        = submissions["root_cause"]
		notifyOnClose    = config.Env.NotifyOnClose
		productChannelID = config.Env.ProductChannelID

		startDate time.Time
	)

	logWriter := app.Logger.With(
		log.NewValue("channelID", channelID),
		log.NewValue("userID", userID),
	)

	var err error
	if startDateText != "" {
		startDate, err = time.Parse(dateLayout, startDateText)
		if err != nil {
			logWriter.Error(
				ctx,
				"command/close.CloseIncidentByDialog ParseInLocation start date ERROR",
				log.NewValue("timeZoneString", "UTC"),
				log.NewValue("startDateText", startDateText),
				log.NewValue("error", err),
			)
			PostErrorAttachment(ctx, app, channelID, userID, err.Error())
			return err
		}

		startDate = startDate.UTC()
	}

	severityLevelInt64 := int64(-1)
	if severityLevel != "" {
		severityLevelInt64, err = getStringInt64(severityLevel)
		if err != nil {
			return err
		}
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
		PostErrorAttachment(ctx, app, channelID, userID, err.Error())
		return err
	}

	ownerTeamName, err := app.ServiceRepository.GetServiceInstanceOwnerTeamName(ctx, inc.ServiceInstanceID)
	if err != nil {
		app.Logger.Error(
			ctx,
			log.Trace(),
			log.Reason("GetServiceInstanceOwnerTeamName"),
			log.NewValue("channelID", channelID),
			log.NewValue("error", err),
		)
		return err
	}

	incident := model.Incident{
		RootCause:     rootCause,
		EndTimestamp:  &now,
		SeverityLevel: severityLevelInt64,
		ChannelID:     channelID,
	}

	if startDateText != "" {
		incident.StartTimestamp = &startDate
	}

	err = app.IncidentRepository.CloseIncident(ctx, &incident)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("CloseIncident"),
			log.NewValue("incident", incident),
			log.NewValue("error", err),
		)
		return err
	}

	inc, err = app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("error", err),
		)
		return err
	}

	card := createCloseCard(inc, inc.ID, ownerTeamName)

	privateAttachment := createClosePrivateAttachment(inc)

	var waitgroup sync.WaitGroup
	defer waitgroup.Wait()

	if notifyOnClose {
		concurrence.WithWaitGroup(&waitgroup, func() {
			postBlockMessage(app, productChannelID, card)
		})
	}

	concurrence.WithWaitGroup(&waitgroup, func() {
		postMessage(app, userID, "", privateAttachment)
	})

	postAndPinBlockMessage(app, channelID, card)

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

func createCloseCard(incident model.Incident, incidentID int64, ownerTeamName string) []slack.Block {
	title := fmt.Sprintf(":white_check_mark: *Incident #%d - %s* has been closed", incidentID, incident.Title)

	bodySlice := []string{}

	bodySlice = append(bodySlice, fmt.Sprintf("*Channel:*\t\t\t\t\t#%s", incident.ChannelName))
	bodySlice = append(bodySlice, fmt.Sprintf("*Team:*\t\t\t\t\t\t%s", ownerTeamName))

	if incident.SeverityLevel > 0 {
		bodySlice = append(bodySlice, fmt.Sprintf("*Severity:*\t\t\t\t\t%s", getSeverityLevelText(incident.SeverityLevel)))
	}

	if incident.PostMortemURL != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("*Post Mortem:*\t\t\t<%s|post mortem link>", incident.PostMortemURL))
	}

	if incident.RootCause != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("\n*Root Cause:*\n%s", incident.RootCause))
	}

	return createBaseCard(title, bodySlice)
}

func createClosePrivateAttachment(inc model.Incident) slack.Attachment {
	var privateText strings.Builder
	privateText.WriteString("The Incident <#" + inc.ChannelID + "> has been closed by you\n\n")

	return slack.Attachment{
		Pretext:  "The Incident <#" + inc.ChannelID + "> has been closed by you",
		Fallback: privateText.String(),
		Text:     "",
		Color:    "#FE4D4D",
	}
}
