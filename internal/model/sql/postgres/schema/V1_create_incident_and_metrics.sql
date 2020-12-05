-- public.incident definition
-- Drop table
-- DROP TABLE public.incident;
CREATE TABLE IF NOT EXISTS public.incident (
	id serial NOT NULL,
	title text NULL,
    service_instance_id integer NOT NULL REFERENCES public.service_instance (id),
	channel_id varchar(50) NULL,
	channel_name text NULL,
    commander_id text NULL,
    commander_email text NULL,
	status varchar(50) NULL,
	description_started text NULL,
	description_resolved text NULL,
	description_cancelled text NULL,
	root_cause text NULL,
	meeting_url text NULL,
	post_mortem_url text NULL,
	severity_level int4 NULL,
	start_ts timestamptz NULL,
	identification_ts timestamptz NULL,
	end_ts timestamptz NULL,
	updated_at timestamp NOT NULL DEFAULT now(),
	CONSTRAINT firstkey PRIMARY KEY (id)
);