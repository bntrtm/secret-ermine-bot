-- +goose Up
CREATE TABLE events (
	server_id text PRIMARY KEY,
	organization_date text NOT NULL,
	distribution_date text NOT NULL,
	join_message_id text NOT NULL,
	join_message_channel_id text NOT NULL,
	notes text NOT NULL,
	organizer_id text NOT NULL
);

CREATE TABLE participants (
	id text NOT NULL,
	server_id text NOT NULL,
	secret_santa_id text NOT NULL,
	giftee_id text NOT NULL,

	PRIMARY KEY (id, server_id),

	FOREIGN KEY (server_id) REFERENCES events(server_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE participants;
DROP TABLE events;
