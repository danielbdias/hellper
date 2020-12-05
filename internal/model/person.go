package model

// Person represents an instance of a person
type Person struct {
	Email         string `db:"email,omitempty"`
	SlackMemberID string `db:"slack_member_id,omitempty"`
}
