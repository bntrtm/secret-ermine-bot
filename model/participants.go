package model

// Participant defines a user as they relate to two other participants
// linked to them within an event.
type Participant struct {
	SecretSanta string `json:"secret_santa_id"` // user tasked with getting this participant a gift
	Giftee      string `json:"giftee_id"`       // user this participant is tasked with giving a gift to
}
