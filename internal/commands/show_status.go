package commands

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"hellper/internal/app"
	"hellper/internal/log"
	"hellper/internal/messages/statusblock"
	"hellper/internal/model"

	"github.com/slack-go/slack"
)

const slackMaxBlocksPerMessage = 50

func createDateFields(inc model.Incident) (fields []slack.AttachmentField) {
	dateLayout := time.RFC1123

	if startTime := inc.StartTimestamp; startTime != nil {
		timeMessage := startTime.Format(dateLayout)

		field := slack.AttachmentField{
			Title: "Incident Initial Time:",
			Value: timeMessage,
		}
		fields = append(fields, field)
	}

	if identificationTime := inc.IdentificationTimestamp; identificationTime != nil {
		timeMessage := identificationTime.Format(dateLayout)

		field := slack.AttachmentField{
			Title: "Incident Identification Time:",
			Value: timeMessage,
		}
		fields = append(fields, field)
	}

	if endTime := inc.EndTimestamp; endTime != nil {
		timeMessage := endTime.Format(dateLayout)

		field := slack.AttachmentField{
			Title: "Incident End Time:",
			Value: timeMessage,
		}
		fields = append(fields, field)
	}

	return fields
}

func createDatesAttachment(ctx context.Context, app *app.App, channelID string) (slack.Attachment, error) {
	logWriter := app.Logger.With(
		log.NewValue("channelID", channelID),
	)

	logWriter.Debug(ctx, log.Trace())

	inc, err := app.IncidentRepository.GetIncident(ctx, channelID)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("GetIncident"),
			log.NewValue("error", err),
		)

		return slack.Attachment{}, err
	}

	fields := createDateFields(inc)
	attach := slack.Attachment{
		Pretext:  "Incident Dates:",
		Fallback: "Incident Dates",
		Text:     "",
		Color:    "#f2b12e",
		Fields:   fields,
	}

	return attach, nil
}

func createStatusAttachment(ctx context.Context, app *app.App, channelID string) (slack.Attachment, error) {
	var (
		attach     slack.Attachment
		fields     []slack.AttachmentField
		attachText string
	)

	logWriter := app.Logger.With(
		log.NewValue("channelID", channelID),
	)

	logWriter.Debug(
		ctx,
		log.Trace(),
		log.Action("createStatusAttachment"),
		log.Reason("AttachmentField"),
	)

	items, _, err := app.Client.ListPins(channelID)
	if err != nil {
		logWriter.Error(
			ctx,
			log.Trace(),
			log.Reason("ListPins"),
			log.NewValue("error", err),
		)

		return slack.Attachment{}, err
	}

	sort.Slice(
		items,
		func(i, j int) bool {
			return items[i].Message.Timestamp < items[j].Message.Timestamp
		},
	)

	if len(items) > 0 {
		for _, item := range items {
			attachText = ""

			timeMessage, err := convertTimestamp(item.Message.Timestamp)
			if err != nil {
				logWriter.Error(
					ctx,
					log.Trace(),
					log.Reason("convertTimestamp"),
					log.NewValue("error", err),
				)

				return slack.Attachment{}, err
			}

			if item.Message.User != "" {
				user, err := app.Client.GetUserInfoContext(ctx, item.Message.User)
				if err != nil {
					logWriter.Error(
						ctx,
						log.Trace(),
						log.Reason("GetUserInfoContext"),
						log.NewValue("error", err),
					)

					return slack.Attachment{}, err
				}

				msg, err := treatMessage(ctx, app, item.Message.Text)
				if err != nil {
					return slack.Attachment{}, err
				}

				attachText = msg + " - @" + user.Name
			} else {
				attachText = item.Message.Attachments[0].Pretext + " - @Hellper"
			}

			field := slack.AttachmentField{
				Value: "```" +
					timeMessage.Format(time.RFC1123) +
					"\n" +
					attachText +
					"```",
			}
			fields = append(fields, field)
		}

		attach = slack.Attachment{
			Pretext:  "Incident Status:",
			Fallback: "Incident Status",
			Text:     "",
			Color:    "#f2b12e",
			Fields:   fields,
		}
	} else {
		field := slack.AttachmentField{
			Title: "Incident Timeline is empty",
		}
		fields = append(fields, field)

		attach = slack.Attachment{
			Pretext:  "Incident Status:",
			Fallback: "Incident Status",
			Text:     "",
			Color:    "#999999",
			Fields:   fields,
		}
	}
	return attach, nil
}

func treatMessage(ctx context.Context, app *app.App, msg string) (string, error) {
	msg = treatHere(msg)
	msg, err := treatUsersMentions(ctx, app, msg)
	if err != nil {
		return "", err
	}

	return msg, nil
}

func treatHere(msg string) string {
	x := []string{
		"here",
		"channel",
	}

	for _, w := range x {
		msg = strings.Replace(msg, "<!"+w+">", "@"+w, -1)
	}

	return msg
}

func treatUsersMentions(ctx context.Context, app *app.App, msg string) (string, error) {
	re := regexp.MustCompile(`<@(\w+)>`)
	userIDs := re.FindAllStringSubmatch(msg, -1)

	for _, id := range userIDs {
		user, err := app.Client.GetUserInfoContext(ctx, id[1])
		if err != nil {
			app.Logger.Error(
				ctx,
				log.Trace(),
				log.Reason("GetUserInfoContext"),
				log.NewValue("message", msg),
				log.NewValue("error", err),
			)
			return "", err
		}

		msg = strings.Replace(msg, id[0], "@"+user.Name, -1)
	}

	return msg, nil
}

// ShowStatus posts an attachment on the channel, with each pinned message from it
func ShowStatus(
	ctx context.Context,
	app *app.App,
	channelID string,
	userID string,
) error {
	err := postLoadingMessage(ctx, app, channelID, userID)

	go func(ctx context.Context) {
		postStatus(ctx, app, channelID, userID)
	}(context.Background())

	return err
}

func postLoadingMessage(ctx context.Context, app *app.App, channelID string, userID string) error {
	return postMessageVisibleOnlyForUser(
		ctx,
		app,
		channelID,
		userID,
		"I will fetch the status for you, this might take a few seconds.",
	)
}

func postStatus(ctx context.Context, app *app.App, channelID string, userID string) error {
	logWriter := app.Logger.With(
		log.NewValue("channelID", channelID),
	)

	logWriter.Debug(
		ctx,
		log.Trace(),
		log.Action("running"),
	)

	statusMessages, err := getStatusMessagesByChannel(ctx, app, channelID)
	if err != nil {
		logWriter.Error(ctx, "Could not load status messages", log.NewValue("error", err))
		return err
	}

	sections := make([]slack.Block, 0, slackMaxBlocksPerMessage)
	for _, statusMessage := range statusMessages {
		logWriter.Debug(ctx, "Processing slack item", log.NewValue("item", statusMessage))
		message := statusblock.Message{App: app, Item: statusMessage}
		messageBlocks := message.CreateLayout(ctx)

		// Slack limits a message to 50 blocks, so we need to split the message in case
		// we have more blocks than that
		if len(sections)+len(messageBlocks) > slackMaxBlocksPerMessage {
			_, _, err = app.Client.PostMessage(channelID, slack.MsgOptionBlocks(sections...))

			if err != nil {
				logWriter.Error(ctx, "Error while sending status message", log.NewValue("error", err))
				return err
			}

			sections = make([]slack.Block, 0, slackMaxBlocksPerMessage)
		}

		sections = append(sections, message.CreateLayout(ctx)...)
	}

	_, _, err = app.Client.PostMessage(channelID, slack.MsgOptionBlocks(sections...))

	if err != nil {
		logWriter.Error(ctx, "Error while sending status message", log.NewValue("error", err))
	}

	return err
}

func getStatusMessagesByChannel(ctx context.Context, app *app.App, channelID string) ([]slack.Item, error) {
	pins, _, err := app.Client.ListPins(channelID)

	if err != nil {
		return []slack.Item{}, err
	}

	sort.Slice(
		pins,
		func(i, j int) bool {
			return pins[i].Message.Timestamp < pins[j].Message.Timestamp
		},
	)

	return pins, nil
}
