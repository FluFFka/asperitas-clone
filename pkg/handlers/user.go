package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"asperitas-clone/pkg/items"
	"asperitas-clone/pkg/session"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	_ "github.com/go-sql-driver/mysql"
)

var (
	SecretKey = []byte("eN3aerUM")
)

// mockgen -source="user.go" -destination="user_mock.go" -package=handlers UserRepositoryInterface

type UserRepositoryInterface interface {
	GetUserByID(int) (*items.User, error)
	GetUserByUsername(string) (*items.User, error)
	AddUser(*items.User) (int, error)
	Authorize(string, string) (*items.User, error)
}

type UserHandler struct {
	PostRepo PostRepositoryInterface
	UserRepo UserRepositoryInterface
	Sessions session.SessionManagerInterface
	Logger   *zap.SugaredLogger
}

func createToken(username string, userID int) ([]byte, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]interface{}{
			"username": username,
			"id":       userID,
		},
	})
	tokenString, err := token.SignedString(SecretKey)
	if err != nil {
		return nil, errors.New(`Token to string transform error`)
	}
	respJSON, err := json.Marshal(
		items.Token{
			Token: tokenString,
		})
	if err != nil {
		return nil, errors.New(`Can't marshall token`)
	}
	return respJSON, nil
}

type privateUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	puser := privateUser{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&puser); err != nil {
		http.Error(w, `Can't decode`, http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	user := items.User{Username: puser.Username, Password: puser.Password}
	userID, err := h.UserRepo.AddUser(&user)
	if errors.Is(err, items.ErrUserAlreadyExists) {
		msg := items.MessageAuthError{
			Location: "body",
			Param:    "username",
			Value:    user.Username,
			Msg:      "already exists",
		}
		resp, err := json.Marshal(
			items.ErrorList{
				Errors: []items.MessageAuthError{msg},
			})
		if err != nil {
			http.Error(w, `Can't marshal errors`, http.StatusInternalServerError)
		}
		http.Error(w, string(resp), 422)
		return
	}
	if err != nil {
		http.Error(w, "Can't add user", http.StatusInternalServerError)
		return
	}

	token, err := createToken(user.Username, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	sess, err := h.Sessions.Create(w, userID)
	if err != nil {
		http.Error(w, `Can't create session`, http.StatusInternalServerError)
		return
	}

	h.Logger.Infof("Created session for %v", sess.UserID)
	w.Write(token)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	pu := privateUser{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&pu); err != nil {
		http.Error(w, `Can't decode`, http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	user, err := h.UserRepo.Authorize(pu.Username, pu.Password)
	if err == items.ErrNoUser {
		jsonError(w, "user not found", http.StatusUnauthorized)
		return
	} else if err == items.ErrBadPass {
		jsonError(w, "invalid password", http.StatusUnauthorized)
		return
	} else if err != nil {
		jsonError(w, "error in DB", http.StatusInternalServerError)
		return
	}
	token, err := createToken(user.Username, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	sess, err := h.Sessions.Create(w, user.ID)
	if err != nil {
		http.Error(w, `Can't create session`, http.StatusInternalServerError)
		return
	}
	w.Write(token)
	h.Logger.Infof("Created session for %v", sess.UserID)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	resp, _ := json.Marshal(map[string]interface{}{
		"message": msg,
	})
	http.Error(w, string(resp), status)
}

func (h *UserHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	username, ok := mux.Vars(r)["USERNAME"]
	if !ok {
		http.Error(w, `Can't get USERNAME`, http.StatusInternalServerError)
		return
	}
	elems, err := h.PostRepo.GetPostsByUsername(username)
	if err != nil {
		http.Error(w, `Can't get posts`, http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(elems)
	if err != nil {
		http.Error(w, `json marshalling error`, http.StatusInternalServerError)
		return
	}
	w.Write(respJSON)
}
