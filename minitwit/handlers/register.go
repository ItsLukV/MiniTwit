package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"net/http"
	"text/template"

	"minitwit/models"
	"minitwit/utils"
)

var registerTmpl = template.Must(template.ParseFiles("templates/layout.html", "templates/register.html"))

func RegisterHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if err := registerTmpl.Execute(w, nil); err != nil {
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
			}
		}
		if r.Method == "POST" {
			username := r.FormValue("username")
			email := r.FormValue("email")
			password := r.FormValue("password")
			password2 := r.FormValue("password2")

			// input validation
			if username == "" || email == "" || password == "" {
				http.Error(w, "You must fill out all fields", http.StatusBadRequest)
			}

			// Check if repeated password matches
			if password != password2 {
				http.Error(w, "Passwords do not match", http.StatusBadRequest)
			}

			//check if user already exists
			_, err := models.GetUserByUsername(database, username)
			if err == nil {
				http.Error(w, "User already exists", http.StatusBadRequest)
			}

			// hash the password
			hash := md5.New()
			hash.Write([]byte(password))
			pwHash := hex.EncodeToString(hash.Sum(nil))

			// insert the user into the database
			_, err = database.Exec("INSERT INTO user (username, email, pw_hash) VALUES (?, ?, ?)", username, email, pwHash)

			// redirect to timeline
			utils.AddFlash(w, r, "You were successfully registered and can login now")
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}
