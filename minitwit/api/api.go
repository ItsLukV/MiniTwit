package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"minitwit/db"
	"minitwit/gorm_models"
	"minitwit/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

const GORMPATH = "../minitwit_gorm.db"

func notReqFromSimulator(w http.ResponseWriter, r *http.Request) bool {
	from_simulator := r.Header.Get("Authorization")
	if from_simulator != "Basic c2ltdWxhdG9yOnN1cGVyX3NhZmUh" {
		w.WriteHeader(403)
		response := map[string]any{"status": 403, "error_msg": "You are not authorized to access this resource!"}
		json.NewEncoder(w).Encode(response)
		return true
	}
	return false
}

func gormGetUserId(db *gorm.DB, username string) (int, error) {
	// Get first matched record
	user := gorm_models.User{}
	result := db.Select("user_id").Where("username = ?", username).First(&user)
	return user.User_id, result.Error

}

func updateLatest(r *http.Request) {
	// Get arg value associated with 'latest' & convert to int
	parsed_command_id := r.FormValue("latest")
	if parsed_command_id != "-1" && parsed_command_id != "" {
		//f, err := os.OpenFile("./latest_processed_sim_action_id.txt", os.O_WRONLY, os.)//, os.ModeAppend)
		f, err := os.Create("./latest_processed_sim_action_id.txt")
		if err != nil {
			log.Fatalf("Failed to read latest_id file: %v", err)
		}

		//this returns an int as well, but not sure what its used to signal lol
		_, err = f.WriteString(parsed_command_id)
		if err != nil {
			log.Fatalf("Failed to convert write id to file: %v", err)
		}
	}
}

// verified working
func getLatest(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("./latest_processed_sim_action_id.txt")
	if err != nil {
		http.Error(w, "Failed to read the latest ID. Try reloading the page and try again.", http.StatusInternalServerError)
	}

	//we need to convert to int, otherwise tests fail
	latest := string(content)
	latestInt, err := strconv.Atoi(latest)
	if err != nil {
		http.Error(w, "Failed to convert string to int.", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"latest": latestInt})
}

func register(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		//must decode into struct bc data sent as json, which golang bitches abt
		d := json.NewDecoder(r.Body)
		var t models.User
		d.Decode(&t)

		var erro string = ""
		if r.Method == "POST" {
			if t.Username == "" {
				erro = "You have to enter a username"
			} else if t.Email == "" || !strings.ContainsAny(t.Email, "@") {
				erro = "You have to enter a valid email address"
			} else if t.Pwd == "" {
				erro = "You have to enter a password"
			} else if _, err := gormGetUserId(database, t.Username); err == nil {
				erro = "The username is already taken"
			} else {
				// hash the password
				hash := md5.New()
				hash.Write([]byte(r.Form.Get("pwd")))
				pwHash := hex.EncodeToString(hash.Sum(nil))
				// insert the user into the database
				user := gorm_models.User{Username: t.Username, Email: t.Email, Pw_hash: pwHash}
				result := database.Create(&user)
				if result.Error != nil {
					log.Fatalf("Failed to insert in db: %v", err)
				}
			}
		}

		if erro != "" {
			jsonSstring, _ := json.Marshal(map[string]any{
				"status":    400,
				"error_msg": erro,
			})
			w.WriteHeader(400)
			w.Write(jsonSstring)
		} else {
			//.Write() sends header with status OK if .WriteHeader() has not yet been called
			//so we can just send empty message to signal status OK
			w.Write(json.RawMessage(""))
		}
	}
}

func messages(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		if notReqFromSimulator(w, r) {
			return
		}
		// no_msgs = request.args.get("no", type=int, default=100)
		//no_msgs := r.FormValue("no")

		if r.Method == "GET" {
			var users []gorm_models.User
			//same as messages_per_user but without 'where user_id = ?"
			// MISSING ORDER BY and LIMIT!!
			database.Model(&gorm_models.User{}).Preload("Messages", "flagged = 0").Find(&users)
			//fmt.Println(users)

			/*query := "SELECT message.*, user.* FROM message, user WHERE message.flagged = 0 AND message.author_id = user.user_id ORDER BY message.pub_date DESC LIMIT ?"
			messages, err := db.QueryDB(database, query, no_msgs)
			if err != nil {
				print(err.Error())
			}*/

			var filtered_msgs []map[string]any
			for _, user := range users {
				for _, message := range user.Messages {
					filtered_msg := make(map[string]any)
					filtered_msg["content"] = message.Text
					filtered_msg["pub_date"] = message.Pub_date
					filtered_msg["user"] = user.Username
					filtered_msgs = append(filtered_msgs, filtered_msg)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(filtered_msgs)
		}
	}
}

func messages_per_user(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		w.Header().Set("Content-Type", "application/json")
		if notReqFromSimulator(w, r) {
			return
		}

		//get the username
		vars := mux.Vars(r)
		username := vars["username"]

		//no_msgs := r.FormValue("no")
		if r.Method == "GET" {
			//user_id, _ := getUserId(database, username)
			user_id, _ := gormGetUserId(database, username)
			if user_id == -1 {
				fmt.Println("messages per user")
				fmt.Println(username)
				print("user id not found in db!")
				//abort(404)
			}

			var users []gorm_models.User
			database.Model(&gorm_models.User{}).Preload("Messages", "flagged = 0").Where("user_id = ?", user_id).Find(&users)
			//fmt.Println(users)

			/*database.Model(&gorm_models.User{}).Preload("Messages", database.Where(&gorm_models.Message{Flagged: 0})).Where("user_id = ?", user_id).Find(&users).Error*/

			/*query := "SELECT message.*, user.* FROM message, user WHERE message.flagged = 0 AND user.user_id = message.author_id AND user.user_id = ? ORDER BY message.pub_date DESC LIMIT ?"
			messages, err := db.QueryDB(database, query, user_id, no_msgs)
			if err != nil {
				print(err.Error())
			}*/

			var filtered_msgs []map[string]any
			for _, user := range users {
				for _, message := range user.Messages {
					filtered_msg := make(map[string]any)
					filtered_msg["content"] = message.Text
					filtered_msg["pub_date"] = message.Pub_date
					filtered_msg["user"] = user.Username
					filtered_msgs = append(filtered_msgs, filtered_msg)
				}
			}

			/*var filtered_msgs []map[string]any
			for messages.Next() {
				var pubDate string //int64
				var messageID, authorID, flagged, userID int
				var text, username, email, pwHash string

				err := messages.Scan(&messageID, &authorID, &text, &pubDate, &flagged, &userID, &username, &email, &pwHash)
				if err != nil {
					print(err.Error())
				}

				fmt.Println("content: ")
				fmt.Println(text)
				fmt.Println("user: ")
				fmt.Println(username)

				filtered_msg := make(map[string]any)
				filtered_msg["content"] = text
				filtered_msg["pub_date"] = pubDate
				filtered_msg["user"] = username
				filtered_msgs = append(filtered_msgs, filtered_msg)
			}*/

			err := json.NewEncoder(w).Encode(filtered_msgs)
			if err != nil {
				fmt.Println("failed to convert messages to json")
				print(err.Error())
			}
		} else if r.Method == "POST" { // post message as <username>
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			content := req["content"]

			user_id, _ := gormGetUserId(database, username)
			message := gorm_models.Message{Author_id: uint(user_id), Text: content.(string), Pub_date: time.Now().GoString()}

			//database.Model(&gorm_models.User{}).Association("Messages").Append(message)

			result := database.Create(&message)
			if result.Error != nil {
				log.Fatalf("Failed to insert in db: %v", result.Error)
			}

			/*query := "INSERT INTO message (author_id, text, pub_date, flagged) VALUES (?, ?, ?, 0)"
			user_id, _ := getUserId(database, username)
			_, err := database.Exec(query, user_id, content, time.Now())
			if err != nil {
				log.Fatalf("Failed to insert in db: %v", err)
			}*/

			//w.Write([]byte("204"))
			w.WriteHeader(204)
		}
	}
}

func follow(database *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateLatest(r)

		w.Header().Set("Content-Type", "application/json")
		if notReqFromSimulator(w, r) {
			return
		}

		//get the username
		vars := mux.Vars(r)
		username := vars["username"]
		//user_id, _ := getUserId(database, username)
		user_id, _ := gormGetUserId(database, username)
		if user_id == -1 {
			fmt.Println("follow")
			fmt.Println(username)
			print("user id not found in db!")
			//abort(404)
		}
		//no_followers := r.FormValue("no")

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		//content := req["content"]

		if r.Method == "POST" && req["follow"] != "" {
			fmt.Println("POST and follow!")
			follows_username := req["follow"] //r.FormValue("follow")
			//follows_user_id, _ := getUserId(database, follows_username)
			follows_user_id, _ := gormGetUserId(database, follows_username)
			if follows_user_id == -1 {
				// TODO: This has to be another error, likely 500???
				//abort(404)
				print("lol error 404 hehe")
			}
			follower := gorm_models.Follower{Who_id: user_id, Whom_id: follows_user_id}
			result := database.Create(&follower)
			if result.Error != nil {
				log.Fatalf("Failed to insert in db: %v", result.Error)
			}

			/*query := "INSERT INTO follower (who_id, whom_id) VALUES (?, ?)"
			_, err := database.Exec(query, user_id, follows_user_id)
			if err != nil {
				log.Fatalf("Failed to insert in db: %v", err)
			}*/
			w.Write([]byte("204"))
		} else if r.Method == "POST" && req["unfollow"] != "" {
			fmt.Println("POST and UNfollow!")
			unfollows_username := req["unfollow"] //r.FormValue("unfollow")
			unfollows_user_id, _ := gormGetUserId(database, unfollows_username)
			if unfollows_user_id == -1 {
				// TODO: This has to be another error, likely 500???
				//abort(404)
				print("lol error 404 hehe")
			}
			//follower := gorm_models.Follower{Who_id: user_id, Whom_id: follows_user_id}
			result := database.Where("who_id=? AND whom_id=?", user_id, unfollows_user_id)
			if result.Error != nil {
				log.Fatalf("Failed to delete from db: %v", result.Error)
			}
			/*query := "DELETE FROM follower WHERE who_id=? and WHOM_ID=?"
			database.Exec(query, user_id, unfollows_user_id)*/

			w.Write([]byte("204"))
		} else if r.Method == "GET" {
			//no_followers := r.FormValue("no")

			var users []gorm_models.User
			//database.Model(&gorm_models.User{}).Preload("Users").Where("who_id=?", user_id).Find(&users)
			//database.Model(&gorm_models.User{}).Preload("Followers", "user_id=?", user_id).Find(&users)
			database.Model(&gorm_models.User{}).Preload("Followers").Where("user_id=?", user_id).Find(&users)
			//database.Model(&gorm_models.User{}).Preload("Messages", "flagged = 0").Where("user_id = ?", user_id).Find(&users)
			fmt.Println(users)

			//get usernames of users whom given user is following
			/*query := "SELECT user.username FROM user INNER JOIN follower ON follower.whom_id=user.user_id WHERE follower.who_id=? LIMIT ?"
			followers, _ := database.Query(query, user_id, no_followers)*/

			var follower_names []string
			for _, user := range users {
				for _, follows := range user.Followers {
					fmt.Println("username: ")
					fmt.Println(follows.Username)
					follower_names = append(follower_names, follows.Username)
				}
			}

			/*for followers.Next() {
				var username string
				err := followers.Scan(&username)
				if err != nil {
					print(err.Error())
				}
				follower_names = append(follower_names, username)
			}*/
			followers_response := map[string]any{"follows": follower_names}
			fmt.Println(followers_response)
			json.NewEncoder(w).Encode(followers_response)
		}
	}
}

func main() {
	// Db logic
	//this MUST be called, otherwise tests fail
	//seems grom cant read already existing database w/out migration stuff
	db.AutoMigrateDB(GORMPATH)
	gorm_db := db.Gorm_ConnectDB(GORMPATH)

	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/register", register(gorm_db)).Methods("POST")
	r.HandleFunc("/latest", getLatest).Methods("GET")
	r.HandleFunc("/msgs", messages(gorm_db)).Methods("GET")
	r.HandleFunc("/msgs/{username}", messages_per_user(gorm_db)).Methods("GET", "POST")
	r.HandleFunc("/fllws/{username}", follow(gorm_db)).Methods("GET", "POST")

	// Start the server
	fmt.Println("API is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
