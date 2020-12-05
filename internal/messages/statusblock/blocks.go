package statusblock

import (
	"context"
	"fmt"
	"hellper/internal/app"
	"hellper/internal/log"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type Message struct {
	App  *app.App
	Item slack.Item
}

type keyValueEntry struct {
	key   string
	value string
}

func (m Message) CreateLayout(ctx context.Context) []slack.Block {
	date, err := m.getDate(ctx)
	if err != nil {
		return []slack.Block{}
	}

	headerText := fmt.Sprintf("*Date: %s*\n*Reporter*: <@%s>\n\n", date, m.Item.Message.User)
	reporterNameBlock := slack.NewTextBlockObject("mrkdwn", headerText, false, false)
	reporterSection := slack.NewSectionBlock(reporterNameBlock, nil, nil)

	messageSections := m.getMessageSections(ctx)
	divider := slack.NewDividerBlock()

	blocks := make([]slack.Block, 0, len(messageSections)+2)
	blocks = append(blocks, reporterSection)
	for _, section := range messageSections {
		blocks = append(blocks, section)
	}
	blocks = append(blocks, divider)

	return blocks
}

func (m Message) getDate(ctx context.Context) (string, error) {
	timestampSeconds := strings.Split(m.Item.Message.Timestamp, ".")[0]

	timestamp, err := strconv.ParseInt(timestampSeconds, 10, 64)
	if err != nil {
		m.App.Logger.Error(
			ctx,
			"Error while parsing message date",
			log.Action("CreateLayout"),
			log.NewValue("date", timestampSeconds),
			log.NewValue("error", err),
		)
		return "", err
	}

	tm := time.Unix(timestamp, 0)
	return tm.Format("01/02/2006 15:04"), nil
}

func (m Message) getMessageSections(ctx context.Context) []slack.Block {
	// All messages have at least one block. But the ones that uses blocks to be built have more than
	// just one block.
	if len(m.Item.Message.Blocks.BlockSet) > 1 {
		return m.Item.Message.Blocks.BlockSet
	}

	if len(m.Item.Message.Attachments) > 0 {
		fieldsBlocks := m.extractMessageFromFields(ctx)
		return []slack.Block{slack.NewSectionBlock(nil, fieldsBlocks, nil)}
	}

	if len(m.Item.Message.Files) > 0 {
		return m.getMessageWithImage(ctx)
	}

	quotedMessage := m.getQuotedMessage()
	messageBlock := slack.NewTextBlockObject("mrkdwn", quotedMessage, false, false)
	return []slack.Block{slack.NewSectionBlock(messageBlock, nil, nil)}
}

func (m Message) getMessageWithImage(ctx context.Context) []slack.Block {
	quotedMessage := m.getQuotedMessage()

	file := m.Item.Message.Files[0]
	imageBlock := slack.NewImageBlockElement(file.Thumb360, file.Title)

	messageBlock := slack.NewTextBlockObject("mrkdwn", quotedMessage, false, false)
	emptyMessageBlock := slack.NewTextBlockObject("mrkdwn", "______", false, false)

	approveBtnTxt := slack.NewTextBlockObject("plain_text", "Open Image", false, false)
	approveBtn := &slack.ButtonBlockElement{
		Type:     "button",
		Text:     approveBtnTxt,
		URL:      file.URLPrivate,
		Style:    slack.StylePrimary,
		ActionID: "open_image",
	}

	messageAndImageSection := slack.NewSectionBlock(messageBlock, nil, slack.NewAccessory(imageBlock))
	buttonSection := slack.NewSectionBlock(emptyMessageBlock, nil, slack.NewAccessory(approveBtn))

	return []slack.Block{messageAndImageSection, buttonSection}
}

func (m Message) getQuotedMessage() string {
	messageLines := strings.Split(m.Item.Message.Text, "\n")
	quotedMessageLines := make([]string, 0, len(messageLines))
	for _, line := range messageLines {
		quotedMessageLines = append(quotedMessageLines, fmt.Sprintf(">%s", line))
	}
	return strings.Join(quotedMessageLines, "\n")
}

func (m Message) extractMessageFromFields(ctx context.Context) []*slack.TextBlockObject {
	fields := m.Item.Message.Attachments[0].Fields

	fieldsTextBlocks := make([]*slack.TextBlockObject, 0, len(fields))
	for _, field := range fields {
		fieldTextBlock := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*:\t%s", field.Title, field.Value), false, false)
		fieldsTextBlocks = append(fieldsTextBlocks, fieldTextBlock)
	}

	return fieldsTextBlocks
}
