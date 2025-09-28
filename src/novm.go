package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yuin/goldmark"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

var tmplFuncs = template.FuncMap{
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
}

type Post struct {
	ID      int
	Title   string
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
		title TEXT,
		slug TEXT UNIQUE,
		content TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", IndexPage)
	http.HandleFunc("/register", SignupPage)
	http.HandleFunc("/login", LoginPage)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/dashboard", AuthMiddleware(DashboardPage))
	http.HandleFunc("/post/", PostPage)
	http.HandleFunc("/new", AuthMiddleware(NewPostPage))
	http.HandleFunc("/edit/", AuthMiddleware(EditPostPage))
	http.HandleFunc("/delete/", AuthMiddleware(DeletePostPage))

	http.Handle("/static/", http.StripPrefix("/static/", 
	http.FileServer(http.Dir("static"))))

	fmt.Println("novm started at http://localhost:1112")
	http.ListenAndServe(":1112", nil)
}

func SignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "hash error", 500)
			return
		}
		_, err = db.Exec(`INSERT INTO users(username, password_hash) VALUES(?, ?)`, 
		username, hash)
		if err != nil {
			http.Error(w, "username already used", 400)
			return
		}
		var userID int
		db.QueryRow("SELECT id FROM users WHERE username=?", username).Scan(&userID)
		createSession(w, userID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "signup.html", nil)
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		var hash string
		var userID int
		err := db.QueryRow(`SELECT id, password_hash FROM users WHERE username=?`, 
		username).Scan(&userID, &hash)
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
	renderTemplate(w, "login.html", nil)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		db.Exec("DELETE FROM sessions WHERE id=?", cookie.Value)
		http.SetCookie(w, &http.Cookie{
			Name:    "session",
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
		})
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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

func DashboardPage(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, slug, content FROM posts ORDER BY id DESC`)
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content)
		posts = append(posts, p)
	}
	renderTemplate(w, "dashboard.html", posts)
}


func IndexPage(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, slug, content FROM posts ORDER BY id DESC`)
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content)
		//var sb strings.Builder
		//if err := goldmark.Convert([]byte(p.Content), &sb); err == nil {
	//		p.HTML = sb.String()
	//	}
		posts = append(posts, p)
	}
	renderTemplate(w, "index.html", posts)
}


func PostPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/post/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var p Post
	err := db.QueryRow(`SELECT id, title, content FROM posts WHERE slug=?`, slug).
	Scan(&p.ID, &p.Title, &p.Content)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var sb strings.Builder
	if err := goldmark.Convert([]byte(p.Content), &sb); err == nil {
		p.HTML = sb.String()
	}
	renderTemplate(w, "post.html", p)
}


func NewPostPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		content := r.FormValue("content")
		slug := slugify(title)

		_, err := db.Exec(`INSERT INTO posts(title, slug, content) VALUES(?,?,?)`,
		title, slug, content)

		if err != nil {
			http.Error(w, "db insert error", 500)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "newpost.html", nil)
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b)
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

func EditPostPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/edit/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		content := r.FormValue("content")
		newSlug := slugify(title)

		_, err := db.Exec(`UPDATE posts SET title=?, slug=?, content=? WHERE slug=?`,
		title, newSlug, content, slug)
		if err != nil {
			http.Error(w, "db update error", 500)
			return
		}
		http.Redirect(w, r, "/post/"+newSlug, http.StatusSeeOther)
		return
	}

	var p Post
	err := db.QueryRow(`SELECT id, title, slug, content FROM posts WHERE slug=?`, slug).
	Scan(&p.ID, &p.Title, &p.Slug, &p.Content)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "edit.html", p)
}

func DeletePostPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/delete/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	_, err := db.Exec(`DELETE FROM posts WHERE slug=?`, slug)
	if err != nil {
		http.Error(w, "db delete error", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func renderTemplate(w http.ResponseWriter, filename string, data interface{}) {
	tmpl, err := template.New(filename).Funcs(tmplFuncs).ParseFiles(`templates/` + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}
