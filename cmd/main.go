package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"asperitas-clone/pkg/handlers"
	"asperitas-clone/pkg/middleware"
	"asperitas-clone/pkg/post_repo"
	"asperitas-clone/pkg/session"
	"asperitas-clone/pkg/user_repo"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"

	_ "github.com/go-sql-driver/mysql"
	_ "go.mongodb.org/mongo-driver/mongo"
)

func main() {
	r := mux.NewRouter()

	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("Error in zap logger")
		return
	}
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	dsn := "root:g9mF7ztS@tcp(localhost:3306)/items"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Can't open mysql db")
		return
	}
	err = db.Ping()
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Can't ping mysql db")
		return
	}

	sess, err := mgo.Dial("mongodb://localhost")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	collection := sess.DB("posts").C("items")
	if collection == nil {
		fmt.Println("Mongo DB is nil")
		return
	}

	sm := &session.SessionManager{SessionDB: db}

	postRepo := &post_repo.PostRepo{PostDB: collection}
	userRepo := &user_repo.UserRepo{UserDB: db}

	userHandler := handlers.UserHandler{
		PostRepo: postRepo,
		UserRepo: userRepo,
		Logger:   logger,
		Sessions: sm,
	}
	postHandler := handlers.PostHandler{
		PostRepo: postRepo,
		UserRepo: userRepo,
		Sessions: sm,
	}

	r.StrictSlash(true)
	r.HandleFunc("/api/login", userHandler.Login).Methods("POST")
	r.HandleFunc("/api/register", userHandler.Register).Methods("POST")

	r.HandleFunc("/api/posts", postHandler.AddPost).Methods("POST")
	r.HandleFunc("/api/posts", postHandler.GetAllPosts).Methods("GET")
	r.HandleFunc("/api/posts/{CATEGORY_NAME}", postHandler.GetPostsByCategory).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}", postHandler.GetPostByID).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}", postHandler.PostComment).Methods("POST")
	r.HandleFunc("/api/post/{POST_ID}/{COMMENT_ID}", postHandler.DeleteComment).Methods("DELETE")
	r.HandleFunc("/api/post/{POST_ID}", postHandler.DeletePost).Methods("DELETE")
	r.HandleFunc("/api/post/{POST_ID}/{VOTE}", postHandler.Vote).Methods("GET")

	r.HandleFunc("/api/user/{USERNAME}", userHandler.GetPosts).Methods("GET")

	r.StrictSlash(false)
	r.PathPrefix("/static").Handler(http.FileServer(http.Dir("./template/")))
	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./template/index.html")
	}))

	auth := middleware.AuthService{
		SessionManager: sm,
		NeedAuth: []middleware.ReqTemplate{
			{
				Method: "POST",
				Reg:    `/api/posts`,
			},
			{
				Method: "POST",
				Reg:    `/api/post/{POST_ID}`,
			},
			{
				Method: "DELETE",
				Reg:    `/api/post/{POST_ID}/{COMMENT_ID}`,
			},
			{
				Method: "GET",
				Reg:    `/api/post/{POST_ID}/{VOTE}`,
			},
			{
				Method: "DELETE",
				Reg:    `/api/post/{POST_ID}`,
			},
		},
	}

	r.Use(auth.Auth)
	reqlog := middleware.ReqLogger{Logger: logger}
	r.Use(reqlog.AccessLog)
	r.Use(middleware.Panic)

	port := ":8080"
	logger.Infow("starting server",
		"type", "START", "port", port,
	)
	http.ListenAndServe(port, r)
}
