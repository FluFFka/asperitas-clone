package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"asperitas-clone/pkg/items"
	"asperitas-clone/pkg/session"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	_ "github.com/go-sql-driver/mysql"
)

// mockgen -source="post.go" -destination="post_mock.go" -package=handlers PostRepositoryInterface

type PostRepositoryInterface interface {
	GetAllPosts() ([]*items.Post, error)
	GetPostsByCategory(string) ([]*items.Post, error)
	GetPostByID(bson.ObjectId) (*items.Post, error)
	AddPost(*items.Post) (bson.ObjectId, error)
	PostComment(*items.Post, *items.Comment) (bson.ObjectId, error)
	DeleteComment(*items.Post, bson.ObjectId, int) error
	DeletePost(bson.ObjectId, *items.User) error
	DeleteUserFromVoteTry(*items.Post, int) error
	Vote(*items.Post, int, int) error
	GetPostsByUsername(string) ([]*items.Post, error)
}

type PostHandler struct {
	PostRepo  PostRepositoryInterface
	UserRepo  UserRepositoryInterface
	Sessions  session.SessionManagerInterface
	SessionDB *sql.DB
}

func (h *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	elems, err := h.PostRepo.GetAllPosts()
	if err != nil {
		http.Error(w, `DB error`, http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(elems)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}

func (h *PostHandler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(r)["POST_ID"]
	if !ok || !bson.IsObjectIdHex(id) {
		http.Error(w, `Can't get post`, http.StatusInternalServerError)
		return
	}
	elem, err := h.PostRepo.GetPostByID(bson.ObjectIdHex(id))
	if err != nil {
		http.Error(w, `DB error`, http.StatusInternalServerError)
		return
	}
	elem.Views++
	respJSON, err := json.Marshal(elem)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}

func (h *PostHandler) GetPostsByCategory(w http.ResponseWriter, r *http.Request) {
	category, ok := mux.Vars(r)["CATEGORY_NAME"]
	if !ok {
		http.Error(w, `Can't get category`, http.StatusInternalServerError)
		return
	}
	elems, err := h.PostRepo.GetPostsByCategory(category)
	if err != nil {
		http.Error(w, `DB error`, http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(elems)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}

func (h *PostHandler) AddPost(w http.ResponseWriter, r *http.Request) {
	post := items.Post{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&post); err != nil {
		http.Error(w, `Can't decode`, http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	sess, err := h.Sessions.Check(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := h.UserRepo.GetUserByID(sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	post.Author = user
	post.Comments = make([]*items.Comment, 0)
	post.Created = time.Now().UTC()
	post.Views = 0
	post.UpvotePercentage = 100
	post.Votes = []items.Vote{
		{
			User: user.ID,
			Vote: 1,
		},
	}
	post.Score = 1
	_, err = h.PostRepo.AddPost(&post)
	if err != nil {
		http.Error(w, `Can't add post`, http.StatusInternalServerError)
		return
	}
	respJSON, err := json.Marshal(post)
	if err != nil {
		http.Error(w, `Can't marshal post`, http.StatusInternalServerError)
		return
	}
	http.Error(w, string(respJSON), http.StatusCreated)
}

func (h *PostHandler) PostComment(w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(r)["POST_ID"]
	if !ok || !bson.IsObjectIdHex(id) {
		http.Error(w, `Can't get post`, http.StatusInternalServerError)
		return
	}
	uid := bson.ObjectIdHex(id)
	message := map[string]string{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&message); err != nil {
		http.Error(w, `Can't decode`, http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	sess, err := h.Sessions.Check(r)
	if err != nil {
		http.Error(w, `Can't get session`, http.StatusInternalServerError)
		return
	}
	if sess == nil {
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}
	user, err := h.UserRepo.GetUserByID(sess.UserID)
	if err != nil {
		http.Error(w, `Can't get user`, http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, `Can't get user`, http.StatusBadRequest)
		return
	}

	comment := items.Comment{
		Created: time.Now().UTC(),
		Author:  user,
		Body:    message["comment"],
	}

	post, err := h.PostRepo.GetPostByID(uid)
	if err != nil {
		http.Error(w, `Can't get post`, http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, `Can't get post`, http.StatusBadRequest)
		return
	}

	_, err = h.PostRepo.PostComment(post, &comment)
	if errors.Is(err, mgo.ErrNotFound) {
		http.Error(w, `Post not found`, http.StatusNoContent)
		return
	} else if err != nil {
		http.Error(w, `Can't post comment`, http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(post)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	postid, ok := mux.Vars(r)["POST_ID"]
	if !ok || !bson.IsObjectIdHex(postid) {
		http.Error(w, `Can't get POST_ID`, http.StatusInternalServerError)
		return
	}
	postuid := bson.ObjectIdHex(postid)
	commentid, ok := mux.Vars(r)["COMMENT_ID"]
	if !ok || !bson.IsObjectIdHex(commentid) {
		http.Error(w, `Can't get COMMENT_ID`, http.StatusInternalServerError)
		return
	}
	commentuid := bson.ObjectIdHex(commentid)
	sess, err := h.Sessions.Check(r)
	if err != nil {
		http.Error(w, `Can't get session`, http.StatusInternalServerError)
		return
	}
	if sess == nil {
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}
	post, err := h.PostRepo.GetPostByID(postuid)
	if err != nil {
		http.Error(w, `Can't get post`, http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, `Can't get post`, http.StatusBadRequest)
		return
	}
	err = h.PostRepo.DeleteComment(post, commentuid, sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respJSON, err := json.Marshal(post)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postid, ok := mux.Vars(r)["POST_ID"]
	if !ok || !bson.IsObjectIdHex(postid) {
		http.Error(w, `Can't get POST_ID`, http.StatusInternalServerError)
		return
	}
	postuid := bson.ObjectIdHex(postid)
	sess, err := h.Sessions.Check(r)
	if err != nil {
		http.Error(w, `Can't get session`, http.StatusInternalServerError)
		return
	}
	if sess == nil {
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}
	user, err := h.UserRepo.GetUserByID(sess.UserID)
	if err != nil {
		http.Error(w, `Can't get user`, http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, `Can't get user`, http.StatusBadRequest)
		return
	}
	err = h.PostRepo.DeletePost(postuid, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Write([]byte(`{"message":"success"}`))
}

func (h *PostHandler) Vote(w http.ResponseWriter, r *http.Request) {
	postid, ok := mux.Vars(r)["POST_ID"]
	if !ok || !bson.IsObjectIdHex(postid) {
		http.Error(w, `Can't get POST_ID`, http.StatusInternalServerError)
		return
	}
	postuid := bson.ObjectIdHex(postid)
	votestring, ok := mux.Vars(r)["VOTE"]
	if !ok {
		http.Error(w, `Can't get VOTE`, http.StatusInternalServerError)
		return
	}
	vote := 0
	switch votestring {
	case "upvote":
		vote = 1
	case "downvote":
		vote = -1
	}
	post, err := h.PostRepo.GetPostByID(postuid)
	if err != nil {
		http.Error(w, `Can't get post`, http.StatusInternalServerError)
		return
	}

	sess, err := h.Sessions.Check(r)
	if err != nil {
		http.Error(w, `Can't get session`, http.StatusInternalServerError)
		return
	}
	if sess == nil {
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}
	err = h.PostRepo.DeleteUserFromVoteTry(post, sess.UserID)
	if err != nil {
		http.Error(w, `Can't delete vote from post`, http.StatusInternalServerError)
		return
	}
	err = h.PostRepo.Vote(post, sess.UserID, vote)
	if err != nil {
		http.Error(w, `Can't vote`, http.StatusInternalServerError)
		return
	}
	respJSON, err := json.Marshal(post)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}
