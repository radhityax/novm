package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"
	"os"

	_ "github.com/mattn/go-sqlite3"

)

// i'm not sure about implementing the configure file,
// so i had to do that in the source code directly.

// register in the browser: 0 for no, 1 for yes
var register_browser_mode int = 0

var db *sql.DB

type Post struct {
	ID      int
	Date	string
	Title   string
	Author	string
	Content string
	Slug	string
	HTML    string
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./novm.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password_hash TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER,
		expiry DATETIME
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT,
		author TEXT,
		title TEXT,
		slug TEXT UNIQUE,
		content TEXT
	)`)

	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) == 1 {
		fmt.Println("you need another option")
		return
	} else if os.Args[1] == "-r" {
		http.HandleFunc("/", IndexPage)

		http.HandleFunc("/login", func (w http.ResponseWriter, r *http.Request) {
			if _, err := r.Cookie("session"); err == nil {
				http.Redirect(w, r, "/dashboard", 302)
			} else {
				LoginPage(w, r)
			}})

			http.HandleFunc("/register", func (w http.ResponseWriter, r *http.Request) {
				if _, err := r.Cookie("session"); err == nil {
					http.Redirect(w, r, "/dashboard", 302)
				} else {
					if (register_browser_mode > 0) {
					SignupPage(w, r)
				} else {
					http.Redirect(w, r, "/", 302)
				}
				}})

				http.HandleFunc("/logout", LogoutHandler)
				http.HandleFunc("/dashboard", AuthMiddleware(DashboardPage))
				http.HandleFunc("/post/", PostPage)
				http.HandleFunc("/new", AuthMiddleware(NewPostPage))
				http.HandleFunc("/edit/", AuthMiddleware(EditPostPage))
				http.HandleFunc("/delete/", AuthMiddleware(DeletePostPage))

				http.Handle("/static/", http.StripPrefix("/static/", 
				http.FileServer(http.Dir("static"))))

				fmt.Println("novm started at http://localhost:1112")
				err := http.ListenAndServe(":1112", nil)
				if err != nil {
					log.Fatal(err)
					return
				}
				return
			} else if os.Args[1] == "-h" {
				fmt.Println("c - create account")
				fmt.Println("h - help")
				fmt.Println("r - run the blog")
				fmt.Println("v - about")
				return
			} else if os.Args[1] == "-v" {
				fmt.Println("novm - wannabe blog system written in golang")
				fmt.Println("github.com/radhityax/novm")
				return
			} else if os.Args[1] == "-c" {
				CreateAccount()
				return
			} else {
				fmt.Println("wrong option. see -h flag for help")
				return
			}
		}


		func createSession(w http.ResponseWriter, userID int) {
			sessionID := generateSessionID()
			expiry := time.Now().Add(1 * time.Hour)
			_, err := db.Exec(`INSERT INTO sessions(id, user_id, expiry) VALUES (?, ?, ?)`, 
			sessionID, userID, expiry)

			if err != nil {
				log.Println("create session error:", err)
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				Expires:  expiry,
			})
		}

		func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie("session")
				if err != nil || cookie.Value == "" {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				var userID int
				var expiry time.Time
				err = db.QueryRow("SELECT user_id, expiry FROM sessions WHERE id=?", 
				cookie.Value).Scan(&userID, &expiry)
				if err != nil || time.Now().After(expiry) {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				next.ServeHTTP(w, r)
			}
		}


		func generateSessionID() string {
			b := make([]byte, 32)
			_, err := rand.Read(b)
			if err != nil {
				log.Fatal(err)
			}
			return hex.EncodeToString(b)
		}

func getusername(w http.ResponseWriter, r *http.Request) (string) {
	if xxx, err := r.Cookie("session"); err == nil {
		var id int
		var user string
		err := db.QueryRow("SELECT user_id FROM sessions WHERE id=?", xxx.Value).Scan(&id)
		if err != nil {
			return ""
		}

		err = db.QueryRow("SELECT username FROM users WHERE id=?",
		id).Scan(&user)
		return user
	} 
	return ""
}
