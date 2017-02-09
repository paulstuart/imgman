package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/paulstuart/secrets"
)

var (
	errEmptyCookie = fmt.Errorf("user cookie is empty")
)

type userLevel struct {
	ID   int
	Name string
}

var userLevels = []userLevel{{0, "User"}, {1, "Editor"}, {2, "Admin"}}

func userByID(id interface{}) (user, error) {
	return getUser("where usr=?", id)
}

func userLogin(id string) string {
	if len(id) == 0 {
		return ""
	}
	if id == "0" {
		return ""
	}
	u, err := userByID(id)
	if err != nil {
		return err.Error()
	}
	if u.Login == nil {
		return ""
	}
	return *u.Login
}

func userByLogin(login string) (user, error) {
	return getUser("where login=?", login)
}

func userByEmail(email string) (user, error) {
	return getUser("where email=?", email)
}

func (user *user) Cookie() string {
	text, err := json.Marshal(user)
	fmt.Println("Marshal user", string(text))
	if err != nil {
		fmt.Println("Marshal user", user, "Error", err)
		return ""
	}
	secret, e2 := secrets.EncryptString(string(text))
	if e2 != nil {
		fmt.Println("Encrypt text", text, "Error", e2)
		return ""
	}
	return secret
}

func (user *user) login() string {
	if user.Login == nil {
		return ""
	}
	return *user.Login
}

func (user *user) FromCookie(cookie string) error {
	if len(cookie) == 0 {
		return errEmptyCookie
	}
	plain, err := secrets.DecryptString(cookie)
	if err != nil {
		return fmt.Errorf("Decrypt text: %s error: %s", cookie, err)
	}
	if err = json.Unmarshal([]byte(plain), &user); err != nil {
		return fmt.Errorf("unmarshal text: %s error: %s", plain, err)
	}
	return nil
}

func userCookie(username string) string {
	u, err := userByEmail(username)
	if err != nil {
		fmt.Println("User error:", err)
		return ""
	}
	return u.Cookie()
}

func userFromCookie(cookie string) user {
	u := &user{}
	// ignore errors -- just return blank user if no cookie set
	u.FromCookie(cookie)
	return *u
}

func userAuth(username, password string) (*user, error) {
	user, err := userByEmail(username)
	if err != nil {
		log.Println("user error:", err)
		return nil, fmt.Errorf("%s is not authorized for access", username)
	}
	if authenticate(username, password) {
		return &user, nil
	}
	return nil, fmt.Errorf("invalid credentials for %s", username)
}

// TODO: should probably cache results in a safe map
func userFromAPIKey(key string) (user, error) {
	return getUser("where apikey=?", key)
}

func getUser(where string, args ...interface{}) (user, error) {
	u := user{}
	err := dbObjectLoad(&u, where, args...)
	if err != nil {
		err = fmt.Errorf("invalid user")
	}
	return u, err
}
