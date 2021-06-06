package user_repo

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"

	"asperitas-clone/pkg/items"

	_ "github.com/go-sql-driver/mysql"
)

type UserRepo struct {
	UserDB *sql.DB
}

func (repo *UserRepo) GetUserByID(id int) (*items.User, error) {
	user := &items.User{}
	row := repo.UserDB.QueryRow("SELECT id, username, password FROM users WHERE id= ?", id)
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *UserRepo) GetUserByUsername(username string) (*items.User, error) {
	user := &items.User{}
	row := repo.UserDB.QueryRow("SELECT id, username, password FROM users WHERE username= ?", username)
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *UserRepo) AddUser(user *items.User) (int, error) {
	var username string
	user.Password = HashPassword(user.Password)
	row := repo.UserDB.QueryRow("SELECT username FROM users WHERE username= ?", user.Username)
	err := row.Scan(&username)
	if err == nil {
		return 0, items.ErrUserAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	result, err := repo.UserDB.Exec(
		"INSERT INTO `users` (`username`, `password`) VALUES (?, ?)",
		user.Username,
		user.Password,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func HashPassword(password string) string {
	hashedPassword := md5.Sum([]byte(password))
	return hex.EncodeToString(hashedPassword[:])
}

func (repo *UserRepo) Authorize(username, expPass string) (*items.User, error) {
	u, err := repo.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, items.ErrNoUser
	}
	if HashPassword(expPass) == u.Password {
		return u, nil
	}
	return nil, items.ErrBadPass
}
