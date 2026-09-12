-- name: GetParticipantsByEvent :many
SELECT *
FROM participants
WHERE server_id = ?;

-- name: GetEvents :many
SELECT *
FROM events;

-- name: UpsertParticipant :one
INSERT INTO participants (
	id,
	server_id,
	secret_santa_id,
	giftee_id
) VALUES (
    ?1, ?2, ?3, ?4
)
ON CONFLICT(id, server_id) DO UPDATE SET
	secret_santa_id = EXCLUDED.secret_santa_id,
	giftee_id = EXCLUDED.giftee_id
RETURNING *;

-- name: UpsertEvent :one
INSERT INTO events (
	server_id,
	organization_date,
	distribution_date,
	join_message_id,
	join_message_channel_id,
	notes,
	organizer_id
) VALUES (
    ?1, ?2, ?3, ?4, ?5, ?6, ?7
)
ON CONFLICT(server_id) DO UPDATE SET
  organization_date = EXCLUDED.organization_date,
  distribution_date = EXCLUDED.distribution_date,
  join_message_id = EXCLUDED.join_message_id,
  join_message_channel_id = EXCLUDED.join_message_channel_id,
  notes = EXCLUDED.notes,
  organizer_id = EXCLUDED.organizer_id
RETURNING *;

-- name: DeleteEventParticipants :exec
DELETE FROM participants
WHERE server_id = ?1;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE server_id = ?1;
