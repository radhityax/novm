package main

import (
	"fmt"
	"bufio"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

/* register account */
func CreateAccount() {
	fmt.Println("username:")
	GetUser := bufio.NewScanner(os.Stdin)
	GetUser.Scan()
	username := GetUser.Text()

	fmt.Println("password:")
	GetPass := bufio.NewScanner(os.Stdin)
	GetPass.Scan()
	pass := GetPass.Text()

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("hash error")
		return
	}

	_, err = db.Exec(`INSERT INTO users(username, password_hash) VALUES(?,?)`,
	username, hash)
}
