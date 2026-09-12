package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bntrtm/secret-ermine-bot/db"
	"github.com/bntrtm/secret-ermine-bot/model"
)

type Repo struct {
	sebDB   *sql.DB
	queries *db.Queries
}

func New(database *sql.DB) (*Repo, error) {
	if database == nil {
		return nil, fmt.Errorf("nil database provided")
	}

	return &Repo{
		sebDB:   database,
		queries: db.New(database),
	}, nil
}

func (r *Repo) SaveEvent(serverID string, e model.SecretSantaEvent) error {
	ctx := context.Background()
	tx, err := r.sebDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	if _, err := qtx.UpsertEvent(ctx, db.UpsertEventParams{
		ServerID:             serverID,
		OrganizationDate:     e.OrganizationDate,
		DistributionDate:     e.DistributionDate,
		JoinMessageID:        e.JoinMessageID,
		JoinMessageChannelID: e.JoinMessageChannelID,
		Notes:                e.Notes,
		OrganizerID:          e.OrganizerID,
	}); err != nil {
		return err
	}
	for k, v := range e.Participants {
		if _, err := qtx.UpsertParticipant(ctx, db.UpsertParticipantParams{
			ID:            k,
			ServerID:      serverID,
			SecretSantaID: v.SecretSanta,
			GifteeID:      v.Giftee,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repo) GetEvents() (map[string]model.SecretSantaEvent, error) {
	ctx := context.Background()

	dbEvents, err := r.queries.GetEvents(ctx)
	if err != nil {
		return nil, err
	}

	events := map[string]model.SecretSantaEvent{}
	for _, e := range dbEvents {
		participants, err := r.getEventParticipants(e.ServerID)
		if err != nil {
			return nil, fmt.Errorf("could not get participants for event on server %s: %v", e.ServerID, err)
		}
		events[e.ServerID] = model.SecretSantaEvent{
			Participants:         participants,
			OrganizationDate:     e.OrganizationDate,
			DistributionDate:     e.DistributionDate,
			JoinMessageID:        e.JoinMessageID,
			JoinMessageChannelID: e.JoinMessageChannelID,
			Notes:                e.Notes,
			OrganizerID:          e.OrganizerID,
		}
	}

	return events, err
}

func (r *Repo) getEventParticipants(serverID string) (map[string]model.Participant, error) {

	dbParticipants, err := r.queries.GetParticipantsByEvent(context.Background(), serverID)
	if err != nil {
		return nil, err
	}

	participants := map[string]model.Participant{}
	for _, p := range dbParticipants {
		participants[p.ID] = model.Participant{
			SecretSanta: p.SecretSantaID,
			Giftee:      p.GifteeID,
		}
	}

	return participants, nil
}
