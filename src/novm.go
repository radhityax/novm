package main

import (
	"fmt"
	"database/sql"
	"log" 
	"html/template"
	"encoding/hex"
	"net/http"
	"time"
	"crypto/rand"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var (
	port = ":1112"
	address = "localhost"
)
func main() {
	var err error
	db, err = sql.Open("sqlite3", "./novm.db")
	
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password_hash TEXT
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER,
		expiry DATETIME
	)`)

	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", LoginPage)
	http.HandleFunc("/register", SignupPage)
	http.HandleFunc("/login", LoginPage)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/dashboard", AuthMiddleware(WelcomePage))

	http.Handle("/static/", http.StripPrefix("/static/",
				http.FileServer(http.Dir("static"))))
	
	fmt.Printf("novm started on %s%s\n", address, port)
	http.ListenAndServe(port, nil)
}

func SignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")

			hash, err := bcrypt.GenerateFromPassword([]byte(password),
			bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "hash error", 500)
				return 
			}

			_, err = db.Exec(`INSERT INTO users(username, password_hash)
			VALUES(?,?)`, username, hash)
			if err != nil {
				http.Error(w, "the username has been used", 400)
				return
			}
			var userID int
			db.QueryRow("SELECT id FROM users WHERE username=?", username).Scan(&userID)
			createSession(w, userID)
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
	}

	tmpl, err := template.ParseFiles("templates/signup.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var hash string
		var userID int
		err := db.QueryRow(`SELECT id, password_hash FROM users WHERE username=?
		`, username).Scan(&userID, &hash)

		if err != nil {
			http.Error(w, "username not found", 401)
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			http.Error(w, "wrong password", 401)
			return
		}
		createSession(w, userID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		db.Exec("DELETE FROM sessions WHERE id=?", cookie.Value)

		http.SetCookie(w, &http.Cookie{
			Name: "session",
			Value: "",
			Path: "/",
			HttpOnly: true,
			Expires: time.Unix(0, 0),
		})
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func createSession(w http.ResponseWriter, userID int) {
	sessionID := generateSessionID()
	expiry := time.Now().Add(1*time.Hour)

	_, err := db.Exec(`INSERT INTO sessions(id,user_id,expiry) VALUES (?,?,?)`,
	sessionID, userID, expiry)
	if err != nil {
		log.Println("createsession error:", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name: "session",
		Value: sessionID,
		Path: "/",
		HttpOnly: true,
		Secure: false,
		Expires: expiry,
		SameSite: http.SameSiteStrictMode,
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
		next.ServeHTTP(w,r)
	}
}
func WelcomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "welkom")
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b)
}
