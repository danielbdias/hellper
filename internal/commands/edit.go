package commands

import (
	"context"
	"fmt"
	"hellper/internal/app"
	"time"

	"hellper/internal/bot"
	"hellper/internal/config"
	"hellper/internal/log"
	"hellper/internal/model"

	"github.com/slack-go/slack"
)

// OpenEditIncidentDialog opens a dialog on Slack, so the user can edit an incident
func OpenEditIncidentDialog(ctx context.Context, app *app.App, channelID string, triggerID string) error {
	services, err := app.ServiceRepository.ListServiceInstances(ctx)
	if err != nil {
		return err
	}

	serviceList := getDialogOptionsWithServiceInstances(services)

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		return err
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

	severityLevel := &slack.DialogInputSelect{
		DialogInput: slack.DialogInput{
			Label:       "Severity level",
			Name:        "severity_level",
			Type:        "select",
			Placeholder: "Set the severity level",
			Optional:    false,
		},
		SelectedOptions: []slack.DialogSelectOption{
			{
				Label: getSeverityLevelText(inc.SeverityLevel),
				Value: fmt.Sprintf("%d", inc.SeverityLevel),
			},
		},
		Options:      getDialogOptionsWithSeverityLevels(),
		OptionGroups: []slack.DialogOptionGroup{},
	}

	product := &slack.DialogInputSelect{
		DialogInput: slack.DialogInput{
			Label:       "Service",
			Name:        "product",
			Type:        "select",
			Placeholder: "Set the service",
			Optional:    false,
		},
		Options:      serviceList,
		OptionGroups: []slack.DialogOptionGroup{},
	}

	commander := &slack.DialogInputSelect{
		DialogInput: slack.DialogInput{
			Label:       "Incident commander",
			Name:        "incident_commander",
			Type:        "select",
			Placeholder: "Set the Incident commander",
			Optional:    false,
		},
		DataSource:   "users",
		OptionGroups: []slack.DialogOptionGroup{},
	}

	description := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Incident description",
			Name:        "incident_description",
			Type:        "textarea",
			Placeholder: "Incident description eg. We're having issues with the Product X or Service Y",
			Optional:    false,
		},
		MaxLength: 500,
	}

	dialog := slack.Dialog{
		CallbackID:     "inc-edit",
		Title:          "Edits an Incident",
		SubmitLabel:    "Edit",
		NotifyOnCancel: false,
		Elements: []slack.DialogElement{
			incidentTitle,
			inc.ChannelName,
			meeting,
			severityLevel,
			product,
			commander,
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
		now              = time.Now().UTC()
		incidentAuthor   = incidentDetails.User.ID
		submission       = incidentDetails.Submission
		incidentTitle    = submission.IncidentTitle
		channelName      = submission.ChannelName
		incidentRoomURL  = submission.IncidentRoomURL
		severityLevel    = submission.SeverityLevel
		product          = submission.Product
		commander        = submission.IncidentCommander
		description      = submission.IncidentDescription
		environment      = config.Env.Environment
		supportTeam      = config.Env.SupportTeam
		productChannelID = config.Env.ProductChannelID
	)

	user, err := getSlackUserInfo(ctx, app, commander)
	if err != nil {
		return fmt.Errorf("commands.StartIncidentByDialog.get_slack_user_info: incident=%v commanderId=%v error=%v", channelName, commander, err)
	}

	severityLevelInt64, err := getStringInt64(severityLevel)
	if err != nil {
		return err
	}

	incident := model.Incident{
		Title:                   incidentTitle,
		Product:                 product,
		DescriptionStarted:      description,
		Status:                  model.StatusOpen,
		IdentificationTimestamp: &now,
		SeverityLevel:           severityLevelInt64,
		IncidentAuthor:          incidentAuthor,
		CommanderID:             user.SlackID,
		CommanderEmail:          user.Email,
	}

	_, err = app.IncidentRepository.UpdateIncident(ctx, &incident)
	return err
}
