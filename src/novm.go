package main

import (
	"fmt"
	_ "html/template"
	"net/http"
)

var Port string = ":1112"
var Address string = "localhost"

func main() {
	http.HandleFunc("/", LoginPage)
	http.HandleFunc("/login", LoginPage)
	http.HandleFunc("/dashboard", WelcomePage)

	http.Handle("/static/", http.StripPrefix("/static/",
				http.FileServer(http.Dir("static"))))
	
	fmt.Printf("novm started on %s%s\n", Address, Port)
	http.ListenAndServe(Port, nil)
}

func SignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")

			fmt.Printf("signup alert\nusername: %s\npassword:%s",
			username, password)
	}
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
}

func WelcomePage(w http.ResponseWriter, r *http.Request) {
}
