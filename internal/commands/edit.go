package commands

import (
	"context"
	"fmt"
	"hellper/internal/app"
	"hellper/internal/model"
	"strconv"
	"time"

	"hellper/internal/bot"
	"hellper/internal/log"

	"github.com/slack-go/slack"
)

// OpenEditIncidentDialog opens a dialog on Slack, so the user can edit an incident
func OpenEditIncidentDialog(ctx context.Context, app *app.App, channelID string, triggerID string) error {
	var (
		startTimestampAsText = ""
	)

	serviceInstances, err := app.ServiceRepository.ListServiceInstances(ctx)
	if err != nil {
		return err
	}

	serviceInstanceList := getDialogOptionsWithServiceInstances(serviceInstances)

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		return err
	}

	if inc.StartTimestamp != nil {
		startTimestampAsText = inc.StartTimestamp.Format(dateLayout)
	}

	incidentTitle := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Incident Title",
			Name:        "incident_title",
			Type:        "text",
			Placeholder: "My Incident Title",
		},
		MaxLength: 100,
		Value:     inc.Title,
	}

	product := &slack.DialogInputSelect{
		DialogInput: slack.DialogInput{
			Label:       "Product / Service",
			Name:        "product",
			Type:        "select",
			Placeholder: "Set the product / service",
			Optional:    false,
		},
		Options:      serviceInstanceList,
		OptionGroups: []slack.DialogOptionGroup{},
		Value:        fmt.Sprintf("%d", inc.ServiceInstanceID),
	}

	commander := &slack.DialogInputSelect{
		DialogInput: slack.DialogInput{
			Label:       "Incident Commander",
			Name:        "incident_commander",
			Type:        "select",
			Placeholder: "Set the Incident commander",
			Optional:    false,
		},
		Value:        inc.Commander.SlackMemberID,
		DataSource:   "users",
		OptionGroups: []slack.DialogOptionGroup{},
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

	meeting := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Meeting URL",
			Name:        "meeting_url",
			Type:        "text",
			Placeholder: "Meeting URL used to discuss the incident eg. Zoom Join URL, Google Meet URL",
			Optional:    true,
		},
		Value: inc.MeetingURL,
	}

	postMortem := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Post Mortem URL",
			Name:        "post_mortem_url",
			Type:        "text",
			Placeholder: "PostMortem URL used to discuss and learn about the incident  eg. Google Docs URL, Wiki URL",
			Optional:    true,
		},
		Value: inc.PostMortemURL,
	}

	startDate := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Start Date",
			Name:        "init_date",
			Type:        "text",
			Placeholder: dateLayout,
			Optional:    true,
		},
		Hint:  "The time is in format " + dateLayout,
		Value: startTimestampAsText,
	}

	rootCause := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Root Cause",
			Name:        "root_cause",
			Type:        "textarea",
			Placeholder: "Incident root cause description.",
			Optional:    true,
		},
		MaxLength: 500,
		Value:     inc.RootCause,
	}

	description := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Incident Description",
			Name:        "incident_description",
			Type:        "textarea",
			Placeholder: "Brief description on what is happening in this incident. eg. We're having issues with the Product X or Service Y",
			Optional:    true,
		},
		MaxLength: 500,
		Value:     inc.DescriptionStarted,
	}

	// Slack force us to have a maximum of 10 fields in the dialog
	dialog := slack.Dialog{
		CallbackID:     "inc-edit",
		Title:          "Edit an Incident",
		SubmitLabel:    "Save",
		NotifyOnCancel: false,
		Elements: []slack.DialogElement{
			incidentTitle,
			product,
			commander,
			severityLevel,
			meeting,
			postMortem,
			startDate,
			rootCause,
			description,
		},
	}

	return app.Client.OpenDialog(triggerID, dialog)
}

// EditIncidentByDialog starts an incident after receiving data from a Slack dialog
func EditIncidentByDialog(
	ctx context.Context,
	app *app.App,
	incidentDetails bot.DialogSubmission,
) error {
	app.Logger.Info(
		ctx,
		"command/open.EditIncidentByDialog",
		log.NewValue("incident_edit_details", incidentDetails),
	)

	var (
		userID                = incidentDetails.User.ID
		channelID             = incidentDetails.Channel.ID
		channelName           = incidentDetails.Channel.Name
		submission            = incidentDetails.Submission
		incidentTitle         = submission["incident_title"]
		serviceInstanceIDText = submission["product"]
		commanderSlackID      = submission["incident_commander"]
		severityLevel         = submission["severity_level"]
		meeting               = submission["meeting_url"]
		postMortem            = submission["post_mortem_url"]
		startTimestampText    = submission["init_date"]
		rootCause             = submission["root_cause"]
		description           = submission["incident_description"]
		startTimestamp        time.Time
	)

	incidentBeforeEdit, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		return err
	}

	commander, err := getSlackUserInfo(ctx, app, commanderSlackID)
	if err != nil {
		return fmt.Errorf("commands.StartIncidentByDialog.get_slack_user_info: incident=%v commanderId=%v error=%v", channelName, commanderSlackID, err)
	}

	severityLevelInt64 := int64(-1)
	if severityLevel != "" {
		severityLevelInt64, err = getStringInt64(severityLevel)
		if err != nil {
			return err
		}
	}

	var serviceInstanceID int
	serviceInstanceID, err = strconv.Atoi(serviceInstanceIDText)
	if err != nil {
		return fmt.Errorf("commands.EditIncidentByDialog.service_instance_id: incident=%v product=%v error=%v",
			channelName, serviceInstanceIDText, err)
	}
	serviceInstanceIDInt64 := int64(serviceInstanceID)

	if startTimestampText != "" {
		startTimestamp, err = time.Parse(dateLayout, startTimestampText)
		if err != nil {
			app.Logger.Error(
				ctx,
				"command.EditIncidentByDialog Parse Start Timestamp ERROR",
				log.NewValue("channelID", channelID),
				log.NewValue("startTimestampText", startTimestampText),
				log.NewValue("error", err),
			)
			PostErrorAttachment(ctx, app, channelID, userID, err.Error())
			return err
		}

		// convert date to timestamp
		startTimestamp = startTimestamp.UTC()
	}

	incident := model.Incident{
		ID:                 incidentBeforeEdit.ID,
		Title:              incidentTitle,
		ServiceInstanceID:  serviceInstanceIDInt64,
		DescriptionStarted: description,
		SeverityLevel:      severityLevelInt64,
		CommanderEmail:     commander.Email,
		MeetingURL:         meeting,
		PostMortemURL:      postMortem,
		RootCause:          rootCause,
		ChannelName:        incidentBeforeEdit.ChannelName,
	}

	if startTimestampText != "" {
		incident.StartTimestamp = &startTimestamp
	}

	err = app.IncidentRepository.UpdateIncident(ctx, &incident)
	if err != nil {
		return err
	}

	incident.ServiceInstance, err = app.ServiceRepository.GetServiceInstance(ctx, serviceInstanceIDInt64)
	if err != nil {
		return err
	}

	incident.Commander = model.Person{
		Email:         commander.Email,
		SlackMemberID: commander.SlackID,
	}

	if incidentBeforeEdit.Commander.SlackMemberID != incident.Commander.SlackMemberID ||
		incidentBeforeEdit.PostMortemURL != incident.PostMortemURL ||
		incidentBeforeEdit.MeetingURL != incident.MeetingURL {
		fillTopic(ctx, app, incident, channelID, meeting, postMortem)
	}

	card := createEditCard(incident, incident.ID)

	postBlockMessage(app, channelID, card)

	return nil
}

func createEditCard(incident model.Incident, incidentID int64) []slack.Block {
	startTimestampAsText := ""

	if incident.StartTimestamp != nil {
		startTimestampAsText = incident.StartTimestamp.Format(dateLayout)
	}

	title := fmt.Sprintf(":white_circle: *Incident #%d - %s* has been edited", incidentID, incident.Title)

	bodySlice := []string{}

	bodySlice = append(bodySlice, fmt.Sprintf("*Product / Service:*\t%s", incident.ServiceInstance.Name))
	bodySlice = append(bodySlice, fmt.Sprintf("*Channel:*\t\t\t\t\t#%s", incident.ChannelName))
	bodySlice = append(bodySlice, fmt.Sprintf("*Commander:*\t\t\t<@%s>", incident.Commander.SlackMemberID))

	if incident.SeverityLevel > 0 {
		bodySlice = append(bodySlice, fmt.Sprintf("*Severity:*\t\t\t\t\t%s", getSeverityLevelText(incident.SeverityLevel)))
	}

	if startTimestampAsText != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("*Start Date:*\t\t\t\t%s", startTimestampAsText))
	}

	if incident.MeetingURL != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("*Meeting:*\t\t\t\t\t<%s|access meeting room>", incident.MeetingURL))
	}

	if incident.PostMortemURL != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("*Post Mortem:*\t\t\t<%s|post mortem link>", incident.PostMortemURL))
	}

	if incident.DescriptionStarted != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("\n*Description:*\n%s", incident.DescriptionStarted))
	}

	if incident.RootCause != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("\n*Root Cause:*\n%s", incident.RootCause))
	}

	return createBaseCard(title, bodySlice)
}
