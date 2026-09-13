package store

import (
	"path/filepath"
	"testing"
)

func prepareUserStore(t *testing.T) {
	t.Helper()

	originalPath := usersFilePath
	originalUsers := Users
	usersFilePath = filepath.Join(t.TempDir(), "users.json")
	Users = nil

	t.Cleanup(func() {
		usersFilePath = originalPath
		Users = originalUsers
	})
}

func TestCreateUserAndLogin(t *testing.T) {
	prepareUserStore(t)

	user, err := CreateUser("alice", "Abcdef12")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.ID != 1 || user.Name != "alice" {
		t.Fatalf("CreateUser() returned %+v", user)
	}

	Users = nil
	LoadUser()
	if len(Users) != 1 || Users[0].Name != "alice" {
		t.Fatalf("LoadUser() loaded %+v", Users)
	}

	loggedIn, err := LoginService("alice", "Abcdef12")
	if err != nil {
		t.Fatalf("LoginService() error = %v", err)
	}
	if loggedIn.ID != user.ID || loggedIn.Name != user.Name {
		t.Fatalf("LoginService() returned %+v", loggedIn)
	}

	if _, err := LoginService("alice", "Wrongpass1"); err == nil {
		t.Fatal("LoginService() accepted an incorrect password")
	}
	if _, err := CreateUser("alice", "Abcdef12"); err == nil {
		t.Fatal("CreateUser() accepted a duplicate username")
	}
}

func TestCreateUserValidation(t *testing.T) {
	prepareUserStore(t)

	testCases := []string{
		"short1A",
		"alllowercase1",
		"ALLUPPERCASE1",
		"OnlyLetters",
	}

	for _, password := range testCases {
		if _, err := CreateUser("alice", password); err == nil {
			t.Fatalf("CreateUser() accepted invalid password %q", password)
		}
	}
}
