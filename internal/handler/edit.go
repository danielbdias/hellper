package handler

import (
	"bytes"
	"net/http"

	"hellper/internal/app"
	"hellper/internal/commands"
	"hellper/internal/log"
)

type handlerEdit struct {
	app *app.App
}

func newHandlerEdit(
	app *app.App,
) *handlerEdit {
	return &handlerEdit{
		app: app,
	}
}

func (h *handlerEdit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		logger = h.app.Logger

		formValues []log.Value
		buf        bytes.Buffer
	)

	r.ParseForm()
	buf.ReadFrom(r.Body)
	body := buf.String()
	logger.Info(
		ctx,
		"handler/edit.ServeHTTP",
		log.NewValue("requestbody", body),
	)

	for key, value := range r.Form {
		formValues = append(formValues, log.NewValue(key, value))
	}
	logger.Info(
		ctx,
		"handler/edit.ServeHTTP Form",
		formValues...,
	)

	channelID := r.FormValue("channel_id")
	triggerID := r.FormValue("trigger_id")

	err := commands.OpenEditIncidentDialog(ctx, h.app, channelID, triggerID)
	if err != nil {
		logger.Error(
			ctx,
			log.Trace(),
			log.Reason("OpenEditIncidentDialog"),
			log.NewValue("triggerID", triggerID),
			log.NewValue("error", err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
