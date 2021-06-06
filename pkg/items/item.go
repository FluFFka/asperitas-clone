package items

import (
	"errors"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Post struct {
	ID               bson.ObjectId `json:"id"`
	Author           *User         `json:"author"`
	Category         string        `json:"category"`
	Comments         []*Comment    `json:"comments"`
	Created          time.Time     `json:"created"`
	Score            int           `json:"score"`
	Title            string        `json:"title"`
	Type             string        `json:"type"`
	Text             string        `json:"text,omitempty"`
	URL              string        `json:"url,omitempty"`
	UpvotePercentage int           `json:"upvotePercentage"`
	Views            int           `json:"views"`
	Votes            []Vote        `json:"votes"`
}

type Vote struct {
	User int `json:"user"`
	Vote int `json:"vote"`
}

type Comment struct {
	ID      bson.ObjectId `json:"id"`
	Created time.Time     `json:"created"`
	Author  *User         `json:"author"`
	Body    string        `json:"body"`
}

type User struct {
	Username string `json:"username"`
	ID       int    `json:"id"`
	Password string `json:"-"`
}

var (
	ErrNoUser            = errors.New("No user found")
	ErrBadPass           = errors.New("Invalid password")
	ErrUserAlreadyExists = errors.New("Username already exists")
	ErrPermissionDenied  = errors.New("Permission denied")
	ErrCommentNotFound   = errors.New("Comment is not found")
)

type MessageAuthError struct {
	Location string      `json:"location"`
	Param    string      `json:"param"`
	Value    interface{} `json:"value"`
	Msg      string      `json:"msg"`
}

type Token struct {
	Token string `json:"token"`
}

type ErrorList struct {
	Errors []MessageAuthError `json:"errors"`
}
