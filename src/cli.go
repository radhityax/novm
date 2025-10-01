package main

import (
	"fmt"
	"bufio"
	"strings"
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
)

/* register account */
func RegistAcc() {
	scanner := bufio.NewScanner(strings
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("hash error")
		return
	}

	_, err = db.Exec(`INSERT INTO users(username, password_hash) VALUES(?,?)`,
	username, hash)
}
