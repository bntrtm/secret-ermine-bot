package main

// 'sgo' as in "stoat go"
import sgo "github.com/sentinelb51/revoltgo"

// botStore tracks persistent data related to the bot's activity across one or more servers
type botStore struct {
	session *sgo.Session

	Token  string
	Events map[string]SecretSantaEvent // map of servers to secret-santa events (limited to one active SSE/server)
}

type Participant struct {
	Username    string `json:"username"`     // Stoat username of this participant in the server
	SecretSanta string `json:"secret_santa"` // user tasked with getting this participant a gift
	Giftee      string `json:"giftee"`       // user this participant is tasked with giving a gift to
	About       string `json:"about"`        // a short message from this participant addressing gift ideas for them
}

type SecretSantaEvent struct {
	Organizer        *sgo.User     `json:"organizer"`         // user that started the session
	OrganizationDate string        `json:"organization_date"` // timestring referring to the date & time the session began
	DistributionDate string        `json:"distribution_date"` // timestring referring to the date & time gifts will be distributed
	Participants     []Participant `json:"participants"`      // list of users participating in the Session

	// Spend limit not enforced with Int parsing and such because currency validation
	// for something this simple just seems unnecessary
	SpendLimit string // a user-input string detailing monetary spending limits for the Secret Santa event
}
