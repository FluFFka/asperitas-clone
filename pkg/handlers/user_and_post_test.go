package handlers

import (
	"asperitas-clone/pkg/items"
	"asperitas-clone/pkg/post_repo"
	"asperitas-clone/pkg/session"
	"asperitas-clone/pkg/user_repo"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// (in directory handlers:)
// go test -v -coverprofile="../../test/user_and_post_cover.out"
// go tool cover -html="../../test/user_and_post_cover.out" -o "../../test/user_and_post_cover.html"

var (
	ErrDB = errors.New("DB_ERROR")
)

type CustomPostMatcher struct {
	check *items.Post
}
type CustomCommentMatcher struct {
	check *items.Comment
}

func (cm CustomPostMatcher) Matches(x interface{}) bool {
	post, ok := x.(*items.Post)
	if !ok {
		return false
	}
	if cm.check.Category == post.Category && cm.check.Text == post.Text && cm.check.Title == post.Title && cm.check.Type == post.Type {
		return true
	}
	return false
}
func (cm CustomPostMatcher) String() string {
	return "*items.Post"
}
func (cm CustomCommentMatcher) Matches(x interface{}) bool {
	comment, ok := x.(*items.Comment)
	if !ok {
		return false
	}
	if cm.check.Body == comment.Body && cm.check.Author.Username == comment.Author.Username {
		return true
	}
	return false
}
func (cm CustomCommentMatcher) String() string {
	return "*items.Comment"
}

func TestUserHandlerGetPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	userService := &UserHandler{
		PostRepo: postSt,
		UserRepo: &user_repo.UserRepo{},
		Sessions: &session.SessionManager{},
		Logger:   zap.NewNop().Sugar(),
	}
	user := &items.User{
		Username: "admin",
		Password: "adminadmin",
	}
	posts := []*items.Post{
		{
			ID:       "0",
			Author:   user,
			Category: "funny",
			Title:    "abacaba",
			Type:     "text",
			Text:     "text2",
		},
	}

	// Good request
	postSt.EXPECT().GetPostsByUsername(user.Username).Return(posts, nil)
	r := httptest.NewRequest("GET", "/api/user/admin", nil)
	r = mux.SetURLVars(r, map[string]string{"USERNAME": "admin"})
	w := httptest.NewRecorder()
	userService.GetPosts(w, r)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	bodyTrue, _ := json.Marshal(posts)
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	} else if string(body) != string(bodyTrue) {
		t.Errorf("expected %s\ngot %s", string(bodyTrue), string(body))
		return
	}

	// No vars
	r = httptest.NewRequest("GET", "/api/user/admin", nil)
	w = httptest.NewRecorder()
	userService.GetPosts(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// DB error
	postSt.EXPECT().GetPostsByUsername(user.Username).Return(nil, ErrDB)
	r = httptest.NewRequest("GET", "/api/user/admin", nil)
	r = mux.SetURLVars(r, map[string]string{"USERNAME": "admin"})
	w = httptest.NewRecorder()
	userService.GetPosts(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestUserHandlerRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	userService := &UserHandler{
		PostRepo: &post_repo.PostRepo{},
		UserRepo: userSt,
		Sessions: managerSt,
		Logger:   zap.NewNop().Sugar(),
	}
	user := &items.User{
		Username: "admin",
		Password: "adminadmin",
	}

	// Good request
	userSt.EXPECT().AddUser(user).Return(1, nil)
	managerSt.EXPECT().Create(gomock.Any(), 1).Return(&session.Session{}, nil)
	bodyString := fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body := strings.NewReader(bodyString)
	r := httptest.NewRequest("POST", "/api/register", body)
	w := httptest.NewRecorder()
	userService.Register(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	}

	// No body
	r = httptest.NewRequest("POST", "/api/register", nil)
	w = httptest.NewRecorder()
	userService.Register(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// User already exists
	userSt.EXPECT().AddUser(user).Return(0, items.ErrUserAlreadyExists)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/register", body)
	w = httptest.NewRecorder()
	userService.Register(w, r)
	resp = w.Result()
	if resp.StatusCode != 422 {
		t.Errorf("expected code 422, got %d", resp.StatusCode)
		return
	}

	// Can't add user
	userSt.EXPECT().AddUser(user).Return(0, ErrDB)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/register", body)
	w = httptest.NewRecorder()
	userService.Register(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// Can't create sessions
	userSt.EXPECT().AddUser(user).Return(1, nil)
	managerSt.EXPECT().Create(gomock.Any(), 1).Return(nil, ErrDB)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/register", body)
	w = httptest.NewRecorder()
	userService.Register(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestUserHandlerLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	userService := &UserHandler{
		PostRepo: &post_repo.PostRepo{},
		UserRepo: userSt,
		Sessions: managerSt,
		Logger:   zap.NewNop().Sugar(),
	}
	user := &items.User{
		ID:       1,
		Username: "admin",
		Password: "adminadmin",
	}

	// Good request
	userSt.EXPECT().Authorize(user.Username, user.Password).Return(user, nil)
	managerSt.EXPECT().Create(gomock.Any(), user.ID).Return(&session.Session{}, nil)
	bodyString := fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body := strings.NewReader(bodyString)
	r := httptest.NewRequest("POST", "/api/login", body)
	w := httptest.NewRecorder()
	userService.Login(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	}

	// No body
	r = httptest.NewRequest("POST", "/api/login", nil)
	w = httptest.NewRecorder()
	userService.Login(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// No user
	userSt.EXPECT().Authorize(user.Username, user.Password).Return(nil, items.ErrNoUser)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/login", body)
	w = httptest.NewRecorder()
	userService.Login(w, r)
	resp = w.Result()
	if resp.StatusCode != 401 {
		t.Errorf("expected code 401, got %d", resp.StatusCode)
		return
	}

	// Wrong password
	userSt.EXPECT().Authorize(user.Username, user.Password).Return(nil, items.ErrBadPass)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/login", body)
	w = httptest.NewRecorder()
	userService.Login(w, r)
	resp = w.Result()
	if resp.StatusCode != 401 {
		t.Errorf("expected code 401, got %d", resp.StatusCode)
		return
	}

	// User db error
	userSt.EXPECT().Authorize(user.Username, user.Password).Return(nil, ErrDB)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/login", body)
	w = httptest.NewRecorder()
	userService.Login(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// Session db error
	userSt.EXPECT().Authorize(user.Username, user.Password).Return(user, nil)
	managerSt.EXPECT().Create(gomock.Any(), user.ID).Return(&session.Session{}, ErrDB)
	bodyString = fmt.Sprintf(`{"username":"%s","password":"%s"}`, user.Username, user.Password)
	body = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", "/api/login", body)
	w = httptest.NewRecorder()
	userService.Login(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerGetAllPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  nil,
		Sessions:  nil,
		SessionDB: nil,
	}
	user := &items.User{
		Username: "admin",
		Password: "adminadmin",
	}
	posts := []*items.Post{
		{
			ID:       "0",
			Author:   user,
			Category: "funny",
			Title:    "abacaba",
			Type:     "text",
			Text:     "text2",
		},
	}

	//Good request
	postSt.EXPECT().GetAllPosts().Return(posts, nil)
	r := httptest.NewRequest("GET", "/api/posts", nil)
	w := httptest.NewRecorder()
	postService.GetAllPosts(w, r)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	bodyTrue, _ := json.Marshal(posts)
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	} else if string(body) != string(bodyTrue) {
		t.Errorf("expected %s\ngot %s", string(bodyTrue), string(body))
		return
	}

	//DB error
	postSt.EXPECT().GetAllPosts().Return(nil, ErrDB)
	r = httptest.NewRequest("GET", "/api/posts", nil)
	w = httptest.NewRecorder()
	postService.GetAllPosts(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerGetPostByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  nil,
		Sessions:  nil,
		SessionDB: nil,
	}
	user := &items.User{
		Username: "admin",
		Password: "adminadmin",
	}
	posts := []*items.Post{
		{
			ID:       bson.NewObjectId(),
			Author:   user,
			Category: "funny",
			Title:    "abacaba",
			Type:     "text",
			Text:     "text2",
		},
	}

	//Good request
	oldViews := posts[0].Views
	postSt.EXPECT().GetPostByID(posts[0].ID).Return(posts[0], nil)
	url := "/api/post/" + posts[0].ID.Hex()
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": posts[0].ID.Hex()})
	w := httptest.NewRecorder()
	postService.GetPostByID(w, r)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	bodyTrue, _ := json.Marshal(posts[0])
	bodyUnmarshalled := items.Post{}
	err := json.Unmarshal(body, &bodyUnmarshalled)
	if err != nil {
		t.Errorf("Can't unmarshall body")
		return
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	} else if string(body) != string(bodyTrue) {
		t.Errorf("expected %s\ngot %s", string(bodyTrue), string(body))
		return
	} else if bodyUnmarshalled.Views-oldViews != 1 {
		t.Errorf("views did't change")
	}

	// Bad ID
	url = "/api/post/-1"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": "-1"})
	w = httptest.NewRecorder()
	postService.GetPostByID(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// DB error
	postSt.EXPECT().GetPostByID(posts[0].ID).Return(posts[0], ErrDB)
	url = "/api/post/" + posts[0].ID.Hex()
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": posts[0].ID.Hex()})
	w = httptest.NewRecorder()
	postService.GetPostByID(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerGetPostsByCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  nil,
		Sessions:  nil,
		SessionDB: nil,
	}
	user := &items.User{
		Username: "admin",
		Password: "adminadmin",
	}
	posts := []*items.Post{
		{
			ID:       bson.NewObjectId(),
			Author:   user,
			Category: "funny",
			Title:    "abacaba",
			Type:     "text",
			Text:     "text2",
		},
	}

	//Good request
	postSt.EXPECT().GetPostsByCategory(posts[0].Category).Return(posts, nil)
	url := "/api/posts/" + posts[0].Category
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"CATEGORY_NAME": posts[0].Category})
	w := httptest.NewRecorder()
	postService.GetPostsByCategory(w, r)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	bodyTrue, _ := json.Marshal(posts)
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	} else if string(body) != string(bodyTrue) {
		t.Errorf("expected %s\ngot %s", string(bodyTrue), string(body))
		return
	}

	// No category
	url = "/api/posts/"
	r = httptest.NewRequest("GET", url, nil)
	w = httptest.NewRecorder()
	postService.GetPostsByCategory(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	// DB error
	postSt.EXPECT().GetPostsByCategory(posts[0].Category).Return(nil, ErrDB)
	url = "/api/posts/" + posts[0].Category
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"CATEGORY_NAME": posts[0].Category})
	w = httptest.NewRecorder()
	postService.GetPostsByCategory(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerAddPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  userSt,
		Sessions:  managerSt,
		SessionDB: nil,
	}
	user := &items.User{
		ID:       1,
		Username: "admin",
		Password: "adminadmin",
	}
	post := &items.Post{
		Category: "funny",
		Title:    "abacaba",
		Type:     "text",
		Text:     "text2",
	}

	//Good request
	url := "/api/posts"
	bodyByteSl, _ := json.Marshal(post)
	bodyInp := strings.NewReader(string(bodyByteSl))
	r := httptest.NewRequest("POST", url, bodyInp)
	w := httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().AddPost(CustomPostMatcher{post}).Return(post.ID, nil)
	postService.AddPost(w, r)
	resp := w.Result()
	if resp.StatusCode != 201 {
		t.Errorf("expected code 201, got %d", resp.StatusCode)
		return
	}

	// No body
	r = httptest.NewRequest("POST", url, nil)
	w = httptest.NewRecorder()
	postService.AddPost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Session DB error
	bodyByteSl, _ = json.Marshal(post)
	bodyInp = strings.NewReader(string(bodyByteSl))
	r = httptest.NewRequest("POST", url, bodyInp)
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, ErrDB)
	postService.AddPost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//User DB error
	bodyByteSl, _ = json.Marshal(post)
	bodyInp = strings.NewReader(string(bodyByteSl))
	r = httptest.NewRequest("POST", url, bodyInp)
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(nil, ErrDB)
	postService.AddPost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Post DB error
	bodyByteSl, _ = json.Marshal(post)
	bodyInp = strings.NewReader(string(bodyByteSl))
	r = httptest.NewRequest("POST", url, bodyInp)
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().AddPost(CustomPostMatcher{post}).Return(post.ID, ErrDB)
	postService.AddPost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerPostComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  userSt,
		Sessions:  managerSt,
		SessionDB: nil,
	}
	user := &items.User{
		ID:       1,
		Username: "admin",
		Password: "adminadmin",
	}
	comment := &items.Comment{
		ID:     bson.NewObjectId(),
		Author: user,
		Body:   "comment",
	}
	post := &items.Post{
		ID:       bson.NewObjectId(),
		Category: "funny",
		Title:    "abacaba",
		Type:     "text",
		Text:     "text2",
	}

	//Good request
	url := "/api/posts/" + post.ID.Hex()
	bodyString := `{"comment":"comment"}`
	bodyInp := strings.NewReader(bodyString)
	r := httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w := httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	postSt.EXPECT().PostComment(post, CustomCommentMatcher{comment}).Return(comment.ID, nil)
	postService.PostComment(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	}

	//Wrong POST_ID
	url = "/api/posts/-1"
	r = httptest.NewRequest("POST", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": "-1"})
	w = httptest.NewRecorder()
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Empty body
	url = "/api/posts/" + post.ID.Hex()
	r = httptest.NewRequest("POST", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Session DB error
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, ErrDB)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No session
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, nil)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 401 {
		t.Errorf("expected code 401, got %d", resp.StatusCode)
		return
	}

	//User DB Error
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(nil, ErrDB)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No user
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(nil, nil)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 400 {
		t.Errorf("expected code 400, got %d", resp.StatusCode)
		return
	}

	//Post DB error (get post)
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(nil, ErrDB)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No post (get post)
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(nil, nil)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 400 {
		t.Errorf("expected code 400, got %d", resp.StatusCode)
		return
	}

	//No post (post comment)
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	postSt.EXPECT().PostComment(post, CustomCommentMatcher{comment}).Return(comment.ID, mgo.ErrNotFound)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 204 {
		t.Errorf("expected code 204, got %d", resp.StatusCode)
		return
	}

	//Post DB error (post comment)
	url = "/api/posts/" + post.ID.Hex()
	bodyString = `{"comment":"comment"}`
	bodyInp = strings.NewReader(bodyString)
	r = httptest.NewRequest("POST", url, bodyInp)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	postSt.EXPECT().PostComment(post, CustomCommentMatcher{comment}).Return(comment.ID, ErrDB)
	postService.PostComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerDeleteComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  userSt,
		Sessions:  managerSt,
		SessionDB: nil,
	}
	user := &items.User{
		ID:       1,
		Username: "admin",
		Password: "adminadmin",
	}
	comment := &items.Comment{
		ID:     bson.NewObjectId(),
		Author: user,
		Body:   "comment",
	}
	post := &items.Post{
		ID:       bson.NewObjectId(),
		Author:   user,
		Category: "funny",
		Title:    "abacaba",
		Type:     "text",
		Text:     "text2",
		Comments: []*items.Comment{
			comment,
		},
	}

	//Good request
	url := "/api/posts/" + post.ID.Hex() + "/" + comment.ID.Hex()
	r := httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": comment.ID.Hex()})
	w := httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().DeleteComment(post, comment.ID, user.ID).Return(nil)
	postService.DeleteComment(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	}

	//Bad POST_ID
	url = "/api/posts/" + "-1" + "/" + comment.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": "-1", "COMMENT_ID": comment.ID.Hex()})
	w = httptest.NewRecorder()
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Bad COMMENT_ID
	url = "/api/posts/" + post.ID.Hex() + "/" + "-1"
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": "-1"})
	w = httptest.NewRecorder()
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Session DB error
	url = "/api/posts/" + post.ID.Hex() + "/" + comment.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": comment.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, ErrDB)
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No session
	url = "/api/posts/" + post.ID.Hex() + "/" + comment.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": comment.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, nil)
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 401 {
		t.Errorf("expected code 401, got %d", resp.StatusCode)
		return
	}

	//Post DB error (get post)
	url = "/api/posts/" + post.ID.Hex() + "/" + comment.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": comment.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(nil, ErrDB)
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No post (get post)
	url = "/api/posts/" + post.ID.Hex() + "/" + comment.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": comment.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(nil, nil)
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 400 {
		t.Errorf("expected code 400, got %d", resp.StatusCode)
		return
	}

	//Post DB error (delete comment)
	url = "/api/posts/" + post.ID.Hex() + "/" + comment.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "COMMENT_ID": comment.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	postSt.EXPECT().DeleteComment(post, comment.ID, user.ID).Return(ErrDB)
	postService.DeleteComment(w, r)
	resp = w.Result()
	if resp.StatusCode != 400 {
		t.Errorf("expected code 400, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerDeletePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  userSt,
		Sessions:  managerSt,
		SessionDB: nil,
	}
	user := &items.User{
		ID:       1,
		Username: "admin",
		Password: "adminadmin",
	}
	post := &items.Post{
		ID:       bson.NewObjectId(),
		Author:   user,
		Category: "funny",
		Title:    "abacaba",
		Type:     "text",
		Text:     "text2",
	}

	//Good request
	url := "/api/posts/" + post.ID.Hex()
	r := httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w := httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().DeletePost(post.ID, user).Return(nil)
	postService.DeletePost(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	}

	//Wrong POST_ID
	url = "/api/posts/" + "-1"
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": "-1"})
	w = httptest.NewRecorder()
	postService.DeletePost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Session DB error
	url = "/api/posts/" + post.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, ErrDB)
	postService.DeletePost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No session
	url = "/api/posts/" + post.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(nil, nil)
	postService.DeletePost(w, r)
	resp = w.Result()
	if resp.StatusCode != 401 {
		t.Errorf("expected code 401, got %d", resp.StatusCode)
		return
	}

	//User DB error
	url = "/api/posts/" + post.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(nil, ErrDB)
	postService.DeletePost(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No user
	url = "/api/posts/" + post.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(nil, nil)
	postService.DeletePost(w, r)
	resp = w.Result()
	if resp.StatusCode != 400 {
		t.Errorf("expected code 400, got %d", resp.StatusCode)
		return
	}

	//Post DB error
	url = "/api/posts/" + post.ID.Hex()
	r = httptest.NewRequest("DELETE", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	userSt.EXPECT().GetUserByID(user.ID).Return(user, nil)
	postSt.EXPECT().DeletePost(post.ID, user).Return(ErrDB)
	postService.DeletePost(w, r)
	resp = w.Result()
	if resp.StatusCode != 400 {
		t.Errorf("expected code 400, got %d", resp.StatusCode)
		return
	}
}

func TestPostHandlerVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	postSt := NewMockPostRepositoryInterface(ctrl)
	userSt := NewMockUserRepositoryInterface(ctrl)
	managerSt := session.NewMockSessionManagerInterface(ctrl)
	postService := &PostHandler{
		PostRepo:  postSt,
		UserRepo:  userSt,
		Sessions:  managerSt,
		SessionDB: nil,
	}
	user := &items.User{
		ID:       1,
		Username: "admin",
		Password: "adminadmin",
	}
	post := &items.Post{
		ID:       bson.NewObjectId(),
		Author:   user,
		Category: "funny",
		Title:    "abacaba",
		Type:     "text",
		Text:     "text2",
		Votes: []items.Vote{
			{
				User: user.ID,
				Vote: 1,
			},
		},
	}

	//Good request
	url := "/api/posts/" + post.ID.Hex() + "/" + "downvote"
	r := httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "VOTE": "downvote"})
	w := httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().DeleteUserFromVoteTry(post, user.ID).Return(nil)
	postSt.EXPECT().Vote(post, user.ID, -1).Return(nil)
	postService.Vote(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected code 200, got %d", resp.StatusCode)
		return
	}

	//Wrong POST_ID
	url = "/api/posts/" + "-1" + "/" + "upvote"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": "-1", "VOTE": "downvote"})
	w = httptest.NewRecorder()
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Wrong VOTE
	url = "/api/posts/" + post.ID.Hex() + "/"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex()})
	w = httptest.NewRecorder()
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Post DB Error
	url = "/api/posts/" + post.ID.Hex() + "/" + "upvote"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "VOTE": "upvote"})
	w = httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(nil, ErrDB)
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Session DB error
	url = "/api/posts/" + post.ID.Hex() + "/" + "upvote"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "VOTE": "upvote"})
	w = httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	managerSt.EXPECT().Check(r).Return(nil, ErrDB)
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//No session
	url = "/api/posts/" + post.ID.Hex() + "/" + "upvote"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "VOTE": "upvote"})
	w = httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	managerSt.EXPECT().Check(r).Return(nil, nil)
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 401 {
		t.Errorf("expected code 401, got %d", resp.StatusCode)
		return
	}

	//Post DB error (DeleteUserFromVoteTry)
	url = "/api/posts/" + post.ID.Hex() + "/" + "upvote"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "VOTE": "upvote"})
	w = httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().DeleteUserFromVoteTry(post, user.ID).Return(ErrDB)
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}

	//Post DB error (Vote)
	url = "/api/posts/" + post.ID.Hex() + "/" + "upvote"
	r = httptest.NewRequest("GET", url, nil)
	r = mux.SetURLVars(r, map[string]string{"POST_ID": post.ID.Hex(), "VOTE": "upvote"})
	w = httptest.NewRecorder()
	postSt.EXPECT().GetPostByID(post.ID).Return(post, nil)
	managerSt.EXPECT().Check(r).Return(&session.Session{ID: "1", UserID: user.ID}, nil)
	postSt.EXPECT().DeleteUserFromVoteTry(post, user.ID).Return(nil)
	postSt.EXPECT().Vote(post, user.ID, 1).Return(ErrDB)
	postService.Vote(w, r)
	resp = w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected code 500, got %d", resp.StatusCode)
		return
	}
}
