package config

import (
	"encoding/json"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/paked/configure"
)

//Env contains operating system environment variables.
var Env = newEnvironment()

type meetingConfig struct {
	ProviderName   string
	ProviderConfig map[string]string
}

type environment struct {
	// Service
	BindAddress string
	Environment string
	// Dependencies
	Logger      string
	Client      string
	Database    string
	FileStorage string
	Calendar    string
	// Dependencies Configuration
	OAuthToken          string
	SlackSigningSecret  string
	ProductChannelID    string
	SupportTeam         string
	DSN                 string
	GoogleCredentials   string
	GoogleDriveFileID   string
	GoogleDriveToken    string
	GoogleCalendarID    string
	GoogleCalendarToken string
	// Incident Management
	PostmortemGapDays             int
	SLAHoursToClose               int
	ReminderOpenStatusSeconds     int
	ReminderResolvedStatusSeconds int
	ReminderOpenNotifyMsg         string
	ReminderResolvedNotifyMsg     string
	MeetingConfig                 meetingConfig
	NotifyOnResolve               bool
	NotifyOnClose                 bool
	NotifyOnCancel                bool
	Timezone                      string
	InvitationStrategy            string
}

func newEnvironment() environment {
	var (
		vars                  = configure.New(configure.NewEnvironment())
		env                   environment
		meetingProviderConfig = ""
	)

	// Dependencies Configuration
	_ = godotenv.Load("development.env")

	env.MeetingConfig = meetingConfig{}

	// Service
	vars.StringVar(&env.BindAddress, "hellper_bind_address", ":8080", "Hellper local bind address")
	vars.StringVar(&env.Environment, "hellper_environment", "", "Hellper current environment")

	// Dependencies
	vars.StringVar(&env.Logger, "hellper_logger", "zap", "Hellper log provider")
	vars.StringVar(&env.Client, "hellper_client", "slack", "Hellper bot client tool")
	vars.StringVar(&env.Database, "hellper_database", "postgres", "Hellper database provider")
	vars.StringVar(&env.MeetingConfig.ProviderName, "hellper_meeting", "matrix", "Name of meeting provider that will create a War room on incident start.")
	vars.StringVar(&env.FileStorage, "hellper_file_storage", "none", "Hellper file storage tool for postmortem document")
	vars.StringVar(&env.Calendar, "hellper_calendar", "none", "Hellper calendar tool for postmortem meeting")

	// Dependencies Configuration
	vars.StringVar(&env.OAuthToken, "hellper_oauth_token", "", "Token to execute oauth actions")
	vars.StringVar(&env.SlackSigningSecret, "hellper_slack_signing_secret", "", "Slack signs the requests confirm that each request comes from Slack by verifying its unique signature")
	vars.StringVar(&env.ProductChannelID, "hellper_product_channel_id", "", "The Product channel id")
	vars.StringVar(&env.SupportTeam, "hellper_support_team", "", "Support team identifier")
	vars.StringVar(&env.DSN, "hellper_dsn", "", "Hellper database provider")
	vars.StringVar(&meetingProviderConfig, "hellper_meeting_provider_config", "{}", "Specific config of meeting provider that will create a War room on incident start.")
	vars.StringVar(&env.GoogleCredentials, "hellper_google_credentials", "", "Google Credentials")
	vars.StringVar(&env.GoogleDriveFileID, "hellper_google_drive_file_id", "", "Google Drive FileId")
	vars.StringVar(&env.GoogleDriveToken, "hellper_google_drive_token", "", "Google Drive Token")
	vars.StringVar(&env.GoogleCalendarID, "hellper_google_calendar_id", "", "Calendar ID to create a event")
	vars.StringVar(&env.GoogleCalendarToken, "hellper_google_calendar_token", "", "Google Calendar Token")

	// Incident Management
	vars.IntVar(&env.PostmortemGapDays, "hellper_postmortem_gap_days", 2, "Gap in days between resolve and postmortem event")
	vars.IntVar(&env.SLAHoursToClose, "hellper_sla_hours_to_close", 168, "SLA hours to close")
	vars.IntVar(&env.ReminderOpenStatusSeconds, "hellper_reminder_open_status_seconds", 7200, "Contains the time for the stat reminder to be triggered when status is open, by default the time is 2 hours if there is no variable")
	vars.IntVar(&env.ReminderResolvedStatusSeconds, "hellper_reminder_resolved_status_seconds", 86400, "Contains the time for the stat reminder to be triggered when status is resolved, by default the time is 24 hours if there is no variable")
	vars.StringVar(&env.ReminderOpenNotifyMsg, "hellper_reminder_open_notify_msg", "Incident Status: Open - Update the status of this incident, just pin a message with status on the channel.", "Notify message when status is open")
	vars.StringVar(&env.ReminderResolvedNotifyMsg, "hellper_reminder_resolved_notify_msg", "Incident Status: Resolved - Update the status of this incident, just pin a message with status on the channel.", "Notify message when status is resolved")
	vars.BoolVar(&env.NotifyOnResolve, "hellper_notify_on_resolve", true, "Notify the Product channel when resolve the incident")
	vars.BoolVar(&env.NotifyOnClose, "hellper_notify_on_close", true, "Notify the Product channel when close the incident")
	vars.BoolVar(&env.NotifyOnCancel, "hellper_notify_on_cancel", true, "Notify the Product channel when cancel the incident")
	vars.StringVar(&env.Timezone, "hellper_timezone", "America/Sao_Paulo", "The local time of a region or a country used to create a event.")
	vars.StringVar(&env.InvitationStrategy, "hellper_invitation_strategy", "invite_all", "Strategy to be used when inviting stakeholders to slack channel")

	vars.Parse()

	env.MeetingConfig.ProviderConfig = getStringMapFromJSON(meetingProviderConfig)

	return env
}

func getStringMapFromJSON(mapAsJSONString string) map[string]string {
	stringMap := make(map[string]string)

	err := json.Unmarshal([]byte(mapAsJSONString), &stringMap)
	if err != nil {
		fmt.Printf("Config with invalid json format. Config: %s", mapAsJSONString)
	}

	return stringMap
}
