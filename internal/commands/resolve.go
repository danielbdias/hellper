package commands

import (
	"context"
	"fmt"
	"hellper/internal/app"
	"hellper/internal/concurrence"
	"sync"
	"time"

	"hellper/internal/bot"
	"hellper/internal/config"
	"hellper/internal/log"
	"hellper/internal/model"

	"github.com/slack-go/slack"
)

// ResolveIncidentDialog opens a dialog on Slack, so the user can resolve an incident
func ResolveIncidentDialog(app *app.App, triggerID string) error {
	description := &slack.TextInputElement{
		DialogInput: slack.DialogInput{
			Label:       "Solution Description",
			Name:        "incident_description",
			Type:        "textarea",
			Placeholder: "Brief description on what was done to solve this incident. eg. The incident was solved in PR #42",
			Optional:    false,
		},
		MaxLength: 500,
	}

	dialogElements := []slack.DialogElement{
		description,
	}

	dialog := slack.Dialog{
		CallbackID:     "inc-resolve",
		Title:          "Resolve an Incident",
		SubmitLabel:    "Resolve",
		NotifyOnCancel: false,
		Elements:       dialogElements,
	}

	return app.Client.OpenDialog(triggerID, dialog)
}

// ResolveIncidentByDialog resolves an incident after receiving data from a Slack dialog
func ResolveIncidentByDialog(
	ctx context.Context,
	app *app.App,
	incidentDetails bot.DialogSubmission,
) error {
	app.Logger.Debug(
		ctx,
		"command/resolve.ResolveIncidentByDialog",
		log.NewValue("incident_resolve_details", incidentDetails),
	)

	var (
		now              = time.Now().UTC()
		channelID        = incidentDetails.Channel.ID
		userID           = incidentDetails.User.ID
		submissions      = incidentDetails.Submission
		description      = submissions["incident_description"]
		notifyOnResolve  = config.Env.NotifyOnResolve
		productChannelID = config.Env.ProductChannelID
	)

	incident := model.Incident{
		ChannelID:           channelID,
		EndTimestamp:        &now,
		DescriptionResolved: description,
	}

	logWriter := app.Logger.With(
		log.NewValue("incident", incident),
		log.NewValue("channelID", channelID),
	)

	logWriter.Debug(
		ctx,
		log.Trace(),
		log.Action("running"),
	)

	err := app.IncidentRepository.ResolveIncident(ctx, &incident)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("ResolveIncident"),
			log.NewValue("error", err),
		)
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

	card := createResolveCard(inc, inc.ID)

	var waitgroup sync.WaitGroup
	defer waitgroup.Wait()

	concurrence.WithWaitGroup(&waitgroup, func() {
		postAndPinBlockMessage(app, channelID, card)
	})

	if notifyOnResolve {
		concurrence.WithWaitGroup(&waitgroup, func() {
			postBlockMessage(app, productChannelID, card)
		})
	}

	postBlockMessage(app, userID, card)

	return nil
}

func createResolveCard(incident model.Incident, incidentID int64) []slack.Block {
	title := fmt.Sprintf(":large_blue_circle: *Incident #%d - %s* has been resolved", incidentID, incident.Title)

	bodySlice := []string{}

	bodySlice = append(bodySlice, fmt.Sprintf("*Channel:*\t\t\t\t\t#%s", incident.ChannelName))

	if incident.PostMortemURL != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("*Post Mortem:*\t\t\t<%s|post mortem link>", incident.PostMortemURL))
	}

	if incident.DescriptionResolved != "" {
		bodySlice = append(bodySlice, fmt.Sprintf("\n*Description:*\n%s", incident.DescriptionResolved))
	}

	return createBaseCard(title, bodySlice)
}
