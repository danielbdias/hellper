package model

import (
	"time"
)

const (
	StatusOpen     = "open"
	StatusCancel   = "canceled"
	StatusResolved = "resolved"
	StatusClosed   = "closed"
)

// Incident is an entity that represents a system or infrastructure problems ocurring in an environment
type Incident struct {
	ID                      int64      `db:"id,omitempty"`
	Title                   string     `db:"title,omitempty"`
	ServiceInstanceID       int64      `db:"service_instance_id,omitempty"`
	Team                    string     `db:"team,omitempty"`
	ChannelID               string     `db:"channel_id,omitempty"`
	ChannelName             string     `db:"channel_name,omitempty"`
	CommanderEmail          string     `db:"commander_email,omitempty"`
	IncidentAuthor          string     `db:"incident_author_id,omitempty"`
	DescriptionStarted      string     `db:"description_started,omitempty"`
	DescriptionResolved     string     `db:"description_resolved,omitempty"`
	DescriptionCancelled    string     `db:"description_cancelled,omitempty"`
	Status                  string     `db:"status,omitempty"`
	RootCause               string     `db:"root_cause,omitempty"`
	MeetingURL              string     `db:"meeting_url,omitempty"`
	PostMortemURL           string     `db:"post_mortem_url,omitempty"`
	SeverityLevel           int64      `db:"severity_level,omitempty"`
	StartTimestamp          *time.Time `db:"start_ts,omitempty"`
	IdentificationTimestamp *time.Time `db:"identification_ts,omitempty"`
	EndTimestamp            *time.Time `db:"end_ts,omitempty"`
	UpdatedAt               *time.Time `db:"updated_at,omitempty"`
	ServiceInstance         ServiceInstance
	Commander               Person
}
