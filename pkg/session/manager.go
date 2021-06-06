package session

import (
	"database/sql"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// mockgen -source="manager.go" -destination="manager_mock.go" -package=session SessionManagerInterface

type SessionManagerInterface interface {
	Create(http.ResponseWriter, int) (*Session, error)
	Check(*http.Request) (*Session, error)
}

type SessionManager struct {
	SessionDB *sql.DB
}

func (sm *SessionManager) Create(w http.ResponseWriter, userID int) (*Session, error) {
	sess := NewSession(userID)
	_, err := sm.SessionDB.Exec(
		"INSERT INTO `sessions` (`id`, `userid`) VALUES (?, ?)",
		sess.ID,
		sess.UserID,
	)
	if err != nil {
		return nil, err
	}

	cookie := &http.Cookie{
		Name:    "sess_id",
		Value:   sess.ID,
		Path:    "/",
		Expires: time.Now().Add(90 * 24 * time.Hour),
	}
	http.SetCookie(w, cookie)
	return sess, nil
}

func (sm *SessionManager) Check(r *http.Request) (*Session, error) {
	sessionCookie, err := r.Cookie("sess_id")
	if err == http.ErrNoCookie {
		return nil, ErrNoAuth
	}
	row := sm.SessionDB.QueryRow("SELECT id, userid FROM sessions WHERE id= ?", sessionCookie.Value)
	sess := Session{}
	err = row.Scan(&sess.ID, &sess.UserID)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}
