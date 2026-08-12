// Package session defines durable session and turn identity.
package session

import "time"

type ID string

type TurnID string

type Session struct {
	ID        ID
	CreatedAt time.Time
}

type Turn struct {
	ID             TurnID
	PreviousTurnID TurnID
}
