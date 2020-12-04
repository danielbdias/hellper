package postgres

import (
	"context"
	"errors"
	"fmt"

	"hellper/internal/log"
	"hellper/internal/model"
	"hellper/internal/model/sql"

	_ "github.com/lib/pq"
)

type incidentRepository struct {
	logger log.Logger
	db     sql.DB
}

// NewIncidentRepository creates a new repository to handle the Incident entity
func NewIncidentRepository(logger log.Logger, db sql.DB) model.IncidentRepository {
	return &incidentRepository{
		logger: logger,
		db:     db,
	}
}

func incidentLogValues(inc *model.Incident) []log.Value {
	return []log.Value{
		log.NewValue("title", inc.Title),
		log.NewValue("descriptionStarted", inc.DescriptionStarted),
		log.NewValue("descriptionCancelled", inc.DescriptionCancelled),
		log.NewValue("descriptionResolved", inc.DescriptionResolved),
		log.NewValue("startTime", inc.StartTimestamp),
		log.NewValue("identificationTime", inc.IdentificationTimestamp),
		log.NewValue("endTime", inc.EndTimestamp),
		log.NewValue("snoozedTime", inc.SnoozedUntil),
		log.NewValue("responsibility", inc.Responsibility),
		log.NewValue("functionality", inc.Functionality),
		log.NewValue("rootCause", inc.RootCause),
		log.NewValue("customerImpact", inc.CustomerImpact),
		log.NewValue("meetingURL", inc.MeetingURL),
		log.NewValue("statusPageURL", inc.StatusPageURL),
		log.NewValue("postMortemURL", inc.PostMortemURL),
		log.NewValue("team", inc.Team),
		log.NewValue("product", inc.Product),
		log.NewValue("severityLevel", inc.SeverityLevel),
		log.NewValue("channelName", inc.ChannelName),
		log.NewValue("channelID", inc.ChannelID),
		log.NewValue("commanderID", inc.CommanderID),
		log.NewValue("commanderEmail", inc.CommanderEmail),
	}
}

// InsertIncident inserts a new incident on a database
func (r *incidentRepository) InsertIncident(ctx context.Context, inc *model.Incident) (int64, error) {
	r.logger.Debug(
		ctx,
		"postgres/incident-repository.InsertIncident INFO",
		incidentLogValues(inc)...,
	)

	insertCommand := `INSERT INTO incident
		( title
		, description_started
		, description_cancelled
		, description_resolved
		, start_ts
		, end_ts
		, identification_ts
		, responsibility
		, functionality
		, root_cause
		, customer_impact
		, meeting_url
		, status_page_url
		, post_mortem_url
		, status
		, product
		, severity_level
		, channel_name
		, channel_id
		, commander_id
		, commander_email)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	RETURNING id`

	id := int64(0)

	idResult := r.db.QueryRow(
		insertCommand,
		inc.Title,
		inc.DescriptionStarted,
		inc.DescriptionCancelled,
		inc.DescriptionResolved,
		inc.StartTimestamp,
		inc.EndTimestamp,
		inc.IdentificationTimestamp,
		inc.Responsibility,
		inc.Functionality,
		inc.RootCause,
		inc.CustomerImpact,
		inc.MeetingURL,
		inc.StatusPageURL,
		inc.PostMortemURL,
		inc.Status,
		inc.Product,
		inc.SeverityLevel,
		inc.ChannelName,
		inc.ChannelID,
		inc.CommanderID,
		inc.CommanderEmail)

	switch err := idResult.Scan(&id); err {
	case nil:
		r.logger.Debug(
			ctx,
			"postgres/incident-repository.InsertIncident SUCCESS",
			incidentLogValues(inc)...,
		)
		return id, nil
	default:
		r.logger.Error(
			ctx,
			"postgres/incident-repository.InsertIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return 0, err
	}
}

// UpdateIncident updates the incident on a database
func (r *incidentRepository) UpdateIncident(ctx context.Context, inc *model.Incident) (int64, error) {
	r.logger.Info(
		ctx,
		"postgres/incident-repository.UpdateIncident INFO",
		incidentLogValues(inc)...,
	)

	updateCommand := `UPDATE incident SET
		description_started = $1,
		description_cancelled = $2,
		description_resolved = $3,
		start_ts = $4,
		end_ts = $5,
		identification_ts = $6,
		responsibility = $7,
		functionality = $8,
		root_cause = $9,
		customer_impact = $10,
		meeting_url = $11,
		status_page_url = $12,
		post_mortem_url = $13,
		status = $14,
		product = $15,
		severity_level = $16,
		commander_id = $17,
		commander_email = $18
	WHERE id = $19
	RETURNING id`

	id := int64(0)

	idResult := r.db.QueryRow(
		updateCommand,
		inc.DescriptionStarted,
		inc.DescriptionCancelled,
		inc.DescriptionResolved,
		inc.StartTimestamp,
		inc.EndTimestamp,
		inc.IdentificationTimestamp,
		inc.Responsibility,
		inc.Functionality,
		inc.RootCause,
		inc.CustomerImpact,
		inc.MeetingURL,
		inc.StatusPageURL,
		inc.PostMortemURL,
		inc.Status,
		inc.Product,
		inc.SeverityLevel,
		inc.CommanderID,
		inc.CommanderEmail,
		inc.ID,
	)

	if err := idResult.Scan(&id); err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.UpdateIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)

		return 0, err
	}

	r.logger.Info(
		ctx,
		"postgres/incident-repository.UpdateIncident SUCCESS",
		incidentLogValues(inc)...,
	)
	return id, nil
}

// AddPostMortemUrl adds a PostMortemUrl into an incident registerd on the repository
func (r *incidentRepository) AddPostMortemURL(ctx context.Context, channelName string, postMortemURL string) error {
	logWriter := r.logger.With(
		log.NewValue("channelName", channelName),
		log.NewValue("postMortemURL", postMortemURL),
	)
	logWriter.Debug(
		ctx,
		"postgres/incident-repository.AddPostMortemUrl INFO",
	)

	updateCommand := `UPDATE incident SET post_mortem_url = $1 WHERE channel_name = $2`

	_, err := r.db.Exec(
		updateCommand,
		postMortemURL,
		channelName)

	if err != nil {
		logWriter.Error(
			ctx,
			"postgres/incident-repository.AddPostMortemUrl ERROR",
			log.NewValue("error", err),
		)
	} else {
		logWriter.Debug(
			ctx,
			"postgres/incident-repository.AddPostMortemUrl SUCCESS",
		)
	}

	return err
}

// GetIncident retrieves an incident entity from the repository given a channelID
func (r *incidentRepository) GetIncident(ctx context.Context, channelID string) (inc model.Incident, err error) {
	logWriter := r.logger.With(
		log.NewValue("channelID", channelID),
	)

	logWriter.Debug(
		ctx,
		"postgres/incident-repository.GetIncident INFO",
	)

	rows, err := r.db.Query(
		GetIncidentByChannelID(),
		channelID,
	)
	if err != nil {
		logWriter.Error(
			ctx,
			"postgres/incident-repository.GetIncident Query ERROR",
			log.NewValue("error", err),
		)

		return model.Incident{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		err = errors.New("Incident " + channelID + "not found")
		logWriter.Error(
			ctx,
			"postgres/incident-repository.GetIncident ERROR",
			log.NewValue("error", err),
		)

		return model.Incident{}, err
	}

	rows.Scan(
		&inc.ID,
		&inc.Title,
		&inc.DescriptionStarted,
		&inc.DescriptionCancelled,
		&inc.DescriptionResolved,
		&inc.StartTimestamp,
		&inc.EndTimestamp,
		&inc.IdentificationTimestamp,
		&inc.SnoozedUntil,
		&inc.Responsibility,
		&inc.Functionality,
		&inc.RootCause,
		&inc.CustomerImpact,
		&inc.MeetingURL,
		&inc.StatusPageURL,
		&inc.PostMortemURL,
		&inc.Status,
		&inc.Product,
		&inc.SeverityLevel,
		&inc.ChannelName,
		&inc.ChannelID,
		&inc.CommanderID,
		&inc.CommanderEmail,
	)

	logWriter.Debug(
		ctx,
		"postgres/incident-repository.GetIncident SUCCESS",
	)
	return inc, nil
}

// GetIncidentByChannelID retrieves an Incident given a channelID
func GetIncidentByChannelID() string {
	return `SELECT
		id
		, title
		, CASE WHEN description_started IS NULL THEN '' ELSE description_started END description_started
		, CASE WHEN description_cancelled IS NULL THEN '' ELSE description_cancelled END description_cancelled
		, CASE WHEN description_resolved IS NULL THEN '' ELSE description_resolved END description_resolved
		, start_ts
		, end_ts
		, identification_ts
    , snoozed_until
    , responsibility
		, functionality
		, root_cause
		, customer_impact
		, meeting_url
		, status_page_url
		, post_mortem_url
		, status
		, product
		, CASE WHEN severity_level IS NULL THEN 0 ELSE severity_level END AS severity_level
		, CASE WHEN channel_name IS NULL THEN '' ELSE channel_name END AS channel_name
		, CASE WHEN channel_id IS NULL THEN '' ELSE channel_id END AS channel_id
		, CASE WHEN commander_id IS NULL THEN '' ELSE commander_id END commander_id
		, CASE WHEN commander_email IS NULL THEN '' ELSE commander_email END commander_email
	FROM incident
	WHERE channel_id = $1
	LIMIT 1`
}

func (r *incidentRepository) UpdateIncidentDates(ctx context.Context, inc *model.Incident) error {
	r.logger.Debug(
		ctx,
		"postgres/incident-repository.UpdateIncidentDates INFO",
		incidentLogValues(inc)...,
	)

	result, err := r.db.Exec(
		`UPDATE incident SET
			start_ts = $1,
			identification_ts = $2,
			end_ts = $3
		WHERE channel_id = $4`,
		inc.StartTimestamp,
		inc.IdentificationTimestamp,
		inc.EndTimestamp,
		inc.ChannelID,
	)
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.UpdateIncidentDates Exec ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.UpdateIncidentDates RowsAffected ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	if rowsAffected == 0 {
		err = errors.New("rows not affected")
		r.logger.Error(
			ctx,
			"postgres/incident-repository.UpdateIncidentDates ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	r.logger.Debug(
		ctx,
		"postgres/incident-repository.UpdateIncidentDates SUCCESS",
		incidentLogValues(inc)...,
	)

	return nil
}

func (r *incidentRepository) CancelIncident(ctx context.Context, inc *model.Incident) error {
	logWriter := r.logger.With(
		log.NewValue("channelID", inc.ChannelID),
		log.NewValue("descriptionCancel", inc.DescriptionCancelled),
	)

	logWriter.Debug(
		ctx,
		"postgres/incident-repository.CancelIncident DEBUG",
	)

	result, err := r.db.Exec(
		`UPDATE incident SET status = $1, description_cancelled = $2 WHERE channel_id = $3`,
		model.StatusCancel,
		inc.DescriptionCancelled,
		inc.ChannelID,
	)

	if err != nil {
		logWriter.Error(
			ctx,
			"postgres/incident-repository.CancelIncident ERROR",
			log.NewValue("error", err),
		)
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		logWriter.Error(
			ctx,
			"postgres/incident-repository.CancelIncident ERROR",
			log.NewValue("error", err),
		)

		return err
	}

	if rowsAffected == 0 {
		err = errors.New("rows not affected")
		logWriter.Error(
			ctx,
			"postgres/incident-repository.CancelIncident ERROR",
			log.NewValue("error", err),
		)

		return err
	}

	logWriter.Info(
		ctx,
		"postgres/incident-repository.CancelIncident SUCCESS",
	)
	return nil
}

func (r *incidentRepository) CloseIncident(ctx context.Context, inc *model.Incident) error {
	//TODO: implement team
	r.logger.Info(
		ctx,
		"postgres/incident-repository.CloseIncident INFO",
		incidentLogValues(inc)...,
	)

	result, err := r.db.Exec(
		`UPDATE incident SET
			root_cause = $1,
			functionality = $2,
			team = $3,
			customer_impact = $4,
			severity_level = $5,
			status = $6,
			responsibility = $7
		WHERE channel_id = $8`,
		inc.RootCause,
		inc.Functionality,
		inc.Team,
		inc.CustomerImpact.Int64,
		inc.SeverityLevel,
		model.StatusClosed,
		inc.Responsibility,
		inc.ChannelID,
	)

	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.CloseIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.CloseIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	if rowsAffected == 0 {
		err = errors.New("rows not affected")
		r.logger.Error(
			ctx,
			"postgres/incident-repository.CloseIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}
	r.logger.Info(
		ctx,
		"postgres/incident-repository.CloseIncident SUCCESS",
		incidentLogValues(inc)...,
	)

	return nil
}

func (r *incidentRepository) ResolveIncident(ctx context.Context, inc *model.Incident) error {
	//TODO: implement team
	r.logger.Info(
		ctx,
		"postgres/incident-repository.ResolveIncident INFO",
		incidentLogValues(inc)...,
	)

	result, err := r.db.Exec(
		`UPDATE incident SET
			status_page_url = $1,
			description_resolved = $2,
			start_ts = $3,
			end_ts = $4,
			status = $5
		WHERE channel_id = $6`,
		inc.StatusPageURL,
		inc.DescriptionResolved,
		inc.StartTimestamp,
		inc.EndTimestamp,
		model.StatusResolved,
		inc.ChannelID,
	)

	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.ResolveIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.ResolveIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	if rowsAffected == 0 {
		err = errors.New("rows not affected")
		r.logger.Error(
			ctx,
			"postgres/incident-repository.ResolveIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	r.logger.Info(
		ctx,
		"postgres/incident-repository.ResolveIncident SUCCESS",
		incidentLogValues(inc)...,
	)

	return nil
}

func (r *incidentRepository) ListActiveIncidents(ctx context.Context) ([]model.Incident, error) {
	r.logger.Info(
		ctx,
		"postgres/incident-repository.ListActiveIncidents",
	)
	var (
		incidents    []model.Incident
		logIncidents []log.Value
	)

	rows, err := r.db.Query(
		GetIncidentStatusFilterQuery(),
		model.StatusOpen,
		model.StatusResolved,
	)
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.ListActiveIncidents Query ERROR",
			log.NewValue("error", err),
		)

		return nil, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		i++
		var inc model.Incident
		err := rows.Scan(
			&inc.ID,
			&inc.Title,
			&inc.DescriptionStarted,
			&inc.DescriptionCancelled,
			&inc.DescriptionResolved,
			&inc.StartTimestamp,
			&inc.EndTimestamp,
			&inc.IdentificationTimestamp,
			&inc.SnoozedUntil,
			&inc.Responsibility,
			&inc.Functionality,
			&inc.RootCause,
			&inc.CustomerImpact,
			&inc.StatusPageURL,
			&inc.PostMortemURL,
			&inc.Status,
			&inc.Product,
			&inc.SeverityLevel,
			&inc.ChannelName,
			&inc.ChannelID,
			&inc.CommanderID,
			&inc.CommanderEmail,
		)
		if err != nil {
			r.logger.Error(
				ctx,
				"postgres/incident-repository.ListActiveIncidents Scan ERROR",
				log.NewValue("error", err),
			)

			return nil, err
		}
		logIncidents = append(logIncidents, log.NewValue(fmt.Sprintf("Incident %d", i), incidentLogValues(&inc)))
		incidents = append(incidents, inc)
	}

	r.logger.Info(
		ctx,
		"postgres/incident-repository.ListActiveIncidents SUCCESS",
		logIncidents...,
	)

	return incidents, nil
}

// GetIncidentStatusFilterQuery returns a query to filter incident by status
func GetIncidentStatusFilterQuery() string {
	return `SELECT
		  id
		, title
		, CASE WHEN description_started IS NULL THEN '' ELSE description_started END description_started
		, CASE WHEN description_cancelled IS NULL THEN '' ELSE description_cancelled END description_cancelled
		, CASE WHEN description_resolved IS NULL THEN '' ELSE description_resolved END description_resolved
		, start_ts
		, end_ts
		, identification_ts
    , snoozed_until
		, responsibility
		, functionality
		, root_cause
		, customer_impact
		, status_page_url
		, post_mortem_url
		, status
		, product
		, CASE WHEN severity_level IS NULL THEN 0 ELSE severity_level END AS severity_level
		, CASE WHEN channel_name IS NULL THEN '' ELSE channel_name END AS channel_name
		, CASE WHEN channel_id IS NULL THEN '' ELSE channel_id END AS channel_id
		, CASE WHEN commander_id IS NULL THEN '' ELSE commander_id END commander_id
		, CASE WHEN commander_email IS NULL THEN '' ELSE commander_email END commander_email
	FROM incident
	WHERE status IN ($1, $2)
	LIMIT 100`
}

func (r *incidentRepository) PauseNotifyIncident(ctx context.Context, inc *model.Incident) error {
	r.logger.Info(
		ctx,
		"postgres/incident-repository.PauseNotifyIncident INFO",
		incidentLogValues(inc)...,
	)

	result, err := r.db.Exec(
		`UPDATE incident SET
			snoozed_until = $1
		WHERE channel_id = $2`,
		inc.SnoozedUntil.Time,
		inc.ChannelID,
	)
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.PauseNotifyIncident Exec ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error(
			ctx,
			"postgres/incident-repository.PauseNotifyIncident RowsAffected ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	if rowsAffected == 0 {
		err = errors.New("rows not affected")
		r.logger.Error(
			ctx,
			"postgres/incident-repository.PauseNotifyIncident ERROR",
			append(
				incidentLogValues(inc),
				log.NewValue("error", err),
			)...,
		)
		return err
	}

	r.logger.Info(
		ctx,
		"postgres/incident-repository.PauseNotifyIncident SUCCESS",
		incidentLogValues(inc)...,
	)

	return nil
}
