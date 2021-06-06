package session

import (
	"errors"
	"math/rand"
)

type Session struct {
	ID     string
	UserID int
}

var (
	ErrNoAuth          = errors.New("No session found")
	letterRunes []rune = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	SessionKey  string = "sessionKey"
)

func randomSessionId() string {
	n := 16
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func NewSession(userID int) *Session {
	return &Session{
		ID:     randomSessionId(),
		UserID: userID,
	}
}
