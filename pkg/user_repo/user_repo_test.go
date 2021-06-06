package user_repo

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"asperitas-clone/pkg/items"

	_ "github.com/go-sql-driver/mysql"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// (in directory user_repo:)
// go test -v -coverprofile="../../test/user_repo_cover.out"
// go tool cover -html="../../test/user_repo_cover.out" -o "../../test/user_repo_cover.html"

var (
	ErrDB = errors.New("DB_ERROR")
)

func TestGetUserByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "username", "password"})
	expect := []*items.User{
		{
			Username: "admin",
			ID:       10,
			Password: "f6fdffe48c908deb0f4c3bd36c032e72",
		},
	}
	for _, user := range expect {
		rows.AddRow(user.ID, user.Username, user.Password)
	}

	// Good query
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE id= ?").
		WithArgs(expect[0].ID).
		WillReturnRows(rows)
	repo := &UserRepo{UserDB: db}
	user, err := repo.GetUserByID(expect[0].ID)
	if err != nil {
		t.Errorf("unexpected err: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if !reflect.DeepEqual(user, expect[0]) {
		t.Errorf("results not match, want %v, have %v", expect[0], user)
		return
	}

	// Row not found
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE id= ?").
		WithArgs(9999).
		WillReturnError(sql.ErrNoRows)
	user, err = repo.GetUserByID(9999)
	if err != nil {
		t.Errorf("unexpected err: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if user != nil {
		t.Errorf("results not match, want %v, have %v", nil, user)
		return
	}

	// DB error
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE id= ?").
		WithArgs(expect[0].ID).
		WillReturnError(ErrDB)
	_, err = repo.GetUserByID(expect[0].ID)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, ErrDB) {
		t.Errorf("unexpected error: %v", err.Error())
		return
	}
}

func TestGetUserByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "username", "password"})
	expect := []*items.User{
		{
			Username: "admin",
			ID:       10,
			Password: "f6fdffe48c908deb0f4c3bd36c032e72",
		},
	}
	for _, user := range expect {
		rows.AddRow(user.ID, user.Username, user.Password)
	}

	// Good query
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs(expect[0].Username).
		WillReturnRows(rows)
	repo := &UserRepo{UserDB: db}
	user, err := repo.GetUserByUsername(expect[0].Username)
	if err != nil {
		t.Errorf("unexpected err: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if !reflect.DeepEqual(user, expect[0]) {
		t.Errorf("results not match, want %v, have %v", expect[0], user)
		return
	}

	// Row not found
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs("abacaba").
		WillReturnError(sql.ErrNoRows)
	user, err = repo.GetUserByUsername("abacaba")
	if err != nil {
		t.Errorf("unexpected err: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if user != nil {
		t.Errorf("results not match, want %v, have %v", nil, user)
		return
	}

	// DB error
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs(expect[0].Username).
		WillReturnError(ErrDB)
	_, err = repo.GetUserByUsername(expect[0].Username)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, ErrDB) {
		t.Errorf("unexpected error: %v", err.Error())
		return
	}
}

func TestAddUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "username", "password"})
	expect := []*items.User{
		{
			Username: "admin",
			ID:       10,
			Password: "f6fdffe48c908deb0f4c3bd36c032e72",
		},
	}
	for _, user := range expect {
		rows.AddRow(user.ID, user.Username, user.Password)
	}

	// Good query
	mock.
		ExpectQuery("SELECT username FROM users WHERE").
		WithArgs("abacaba").
		WillReturnError(sql.ErrNoRows)
		//WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("abacaba"))
	mock.
		ExpectExec("INSERT INTO `users`").
		WithArgs("abacaba", "password").
		WillReturnResult(sqlmock.NewResult(1, 1))
	repo := &UserRepo{UserDB: db}
	id, err := repo.AddUser(&items.User{Username: "abacaba", Password: "password"})
	if err != nil {
		t.Errorf("unexpected err: %s", err)
		return
	} else if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	} else if id != 1 {
		t.Errorf("results not match, want %v, have %v", 1, id)
		return
	}

	// Already exsists
	mock.
		ExpectQuery("SELECT username FROM users WHERE").
		WithArgs("abacaba").
		WillReturnRows(sqlmock.NewRows([]string{"username"}).AddRow("abacaba"))
	id, err = repo.AddUser(&items.User{Username: "abacaba", Password: "password"})
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, items.ErrUserAlreadyExists) {
		t.Errorf("unexpected err: %s", err)
		return
	}

	// DB error in SELECT
	mock.
		ExpectQuery("SELECT username FROM users WHERE").
		WithArgs("abacaba").
		WillReturnError(ErrDB)
	id, err = repo.AddUser(&items.User{Username: "abacaba", Password: "password"})
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, ErrDB) {
		t.Errorf("unexpected err: %s", err)
		return
	}

	// DB error in INSERT
	mock.
		ExpectQuery("SELECT username FROM users WHERE").
		WithArgs("abacaba").
		WillReturnError(sql.ErrNoRows)
	mock.
		ExpectExec("INSERT INTO `users`").
		WithArgs("abacaba", "password").
		WillReturnError(ErrDB)
	id, err = repo.AddUser(&items.User{Username: "abacaba", Password: "password"})
	if !errors.Is(err, ErrDB) {
		t.Errorf("unexpected err: %s", err)
		return
	}
}

func TestAuthorize(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("cant create mock: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "username", "password"})
	expect := []*items.User{
		{
			Username: "admin",
			ID:       10,
			Password: "f6fdffe48c908deb0f4c3bd36c032e72",
		},
	}
	for _, user := range expect {
		rows.AddRow(user.ID, user.Username, user.Password)
	}

	// Good query
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs(expect[0].Username).
		WillReturnRows(rows)
	repo := &UserRepo{UserDB: db}
	user, err := repo.Authorize(expect[0].Username, "adminadmin")
	if err != nil {
		t.Errorf("unexpected err: %s", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
	if !reflect.DeepEqual(user, expect[0]) {
		t.Errorf("results not match, want %v, have %v", expect[0], user)
		return
	}

	// No user
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs("abacaba").
		WillReturnError(sql.ErrNoRows)
	user, err = repo.Authorize("abacaba", "123456789")
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, items.ErrNoUser) {
		t.Errorf("unexpected error: %s", err)
		return
	}

	// DB error
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs(expect[0].Username).
		WillReturnError(ErrDB)
	user, err = repo.Authorize(expect[0].Username, "adminadmin")
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, ErrDB) {
		t.Errorf("unexpected error: %s", err)
		return
	}

	// Bad password
	for _, user := range expect {
		rows.AddRow(user.ID, user.Username, user.Password)
	}
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username= ?").
		WithArgs(expect[0].Username).
		WillReturnRows(rows)
	user, err = repo.Authorize(expect[0].Username, "neadminneadmin")
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	} else if !errors.Is(err, items.ErrBadPass) {
		t.Errorf("unexpected error: %s", err)
		return
	}
}
