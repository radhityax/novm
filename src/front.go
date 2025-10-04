package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yuin/goldmark"
	"golang.org/x/crypto/bcrypt"


)

var tmplFuncs = template.FuncMap {
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"add": func(a, b int) int {
		return a+b
	},
	"sub": func(a, b int) int {
		return a-b
	},
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
	renderTemplate(w, r, "login.html", nil)
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
	renderTemplate(w, r, "signup.html", nil)
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

func DashboardPage(w http.ResponseWriter, r *http.Request) {
	author := getusername(w, r)

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if page < 1 {
		page = 1
	}
	limit := 5
	offset := (page - 1) * limit

	rows, err := db.Query(`SELECT id, date, author, title, slug, content FROM 
	posts WHERE author=? ORDER BY id DESC LIMIT ? OFFSET ?`, author, limit, 
	offset)

	if err != nil {
		http.Error(w, "db error", 500)
		return
	}

	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Date, &p.Author, &p.Title, &p.Slug, &p.Content)
		var sb strings.Builder
		if err := goldmark.Convert([]byte(p.Date), &sb); err == nil {
			p.HTML = sb.String()
		}

		posts = append(posts, p)
	}
	data := struct {
		Posts []Post
		Page int
	}{
		Posts: posts,
		Page: page,
	}
	renderTemplate(w, r, "dashboard.html", data)
}

func IndexPage(w http.ResponseWriter, r *http.Request) {

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if page < 1 {
		page = 1
	}

	limit := 5
	offset := (page - 1) * limit

	rows, err := db.Query(`SELECT id, date, author, title, slug, content FROM 
	posts ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)

	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Date, &p.Author, &p.Title, &p.Slug, &p.Content)
		var sb strings.Builder
		if err := goldmark.Convert([]byte(p.Date), &sb); err == nil {
			p.HTML = sb.String()
		}
		posts = append(posts, p)
	}

	data := struct {
		Posts []Post
		Page int
	}{
		Posts: posts,
		Page: page,
	}
	renderTemplate(w, r, "index.html", data)
}

func PostPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/post/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var p Post
	err := db.QueryRow(`SELECT id, date, author, title, content FROM posts WHERE slug=?`, slug).
	Scan(&p.ID, &p.Date, &p.Author, &p.Title, &p.Content)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var sb strings.Builder
	if err := goldmark.Convert([]byte(p.Content), &sb); err == nil {
		p.HTML = sb.String()
	}
	renderTemplate(w, r, "post.html", p)
}

func NewPostPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		title := r.FormValue("title")
		content := r.FormValue("content")
		slug := slugify(title)
		date := time.Now().Format(time.RFC3339)
		if lol, err := r.Cookie("session"); err == nil {

			var userID int
			var userName string
			err := db.QueryRow("SELECT user_id FROM sessions WHERE id =?",
			lol.Value).Scan(&userID)
			if err != nil {
				return
			}

			err = db.QueryRow("SELECT username FROM users where id=?",
			userID).Scan(&userName)

			_, err = db.Exec(`INSERT INTO posts(date, author, title, slug, content) 
			VALUES(?,?,?,?,?)`, date, userName, title, slug, content)

			if err != nil {
				http.Error(w, "db insert error", 500)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	renderTemplate(w, r, "newpost.html", nil)
}

func EditPostPage(w http.ResponseWriter, r *http.Request) {
	username := getusername(w, r)
	var p Post

	slug := strings.TrimPrefix(r.URL.Path, "/edit/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}
	err := db.QueryRow(`SELECT author FROM posts 
	WHERE slug=?`,  slug).Scan(&p.Author)

	if err != nil || p.Author != username {
		http.Error(w, ":(", http.StatusForbidden)
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

	err = db.QueryRow(`SELECT id, date, author, title, slug, content FROM posts WHERE slug=?`, slug).
	Scan(&p.ID, &p.Date, &p.Author, &p.Title, &p.Slug, &p.Content)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, r, "edit.html", p)
}

func DeletePostPage(w http.ResponseWriter, r *http.Request) {
	username := getusername(w, r)
	slug := strings.TrimPrefix(r.URL.Path, "/delete/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var author string
	err := db.QueryRow(`SELECT author FROM posts WHERE slug=?`, slug).Scan(&author)
	if err != nil || author != username {
		http.Error(w, ":(", http.StatusForbidden)
		return
	}

	_, err = db.Exec(`DELETE FROM posts WHERE slug=?`, slug)
	if err != nil {
		http.Error(w, "db delete error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}


func renderTemplate(w http.ResponseWriter, r *http.Request, filename string, data interface{}) {
	tmpl, err := template.New(filename).Funcs(tmplFuncs).ParseFiles(`templates/` + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	head, err := template.ParseFiles(`templates/head.html`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	header, err := template.ParseFiles(`templates/header.html`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logged, err := template.ParseFiles("templates/logged.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	footer, err := template.ParseFiles(`templates/footer.html`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := head.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := r.Cookie("session"); err == nil {
		username := getusername(w, r)
		data := map[string]interface{}{
			"Username": username,
		}
		if err := logged.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := header.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := footer.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}
