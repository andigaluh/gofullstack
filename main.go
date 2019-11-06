package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/kataras/go-sessions"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var err error

type user struct {
	ID        int
	Username  string
	FirstName string
	LastName  string
	Password  string
	Group_id  string
}

type M map[string]interface{}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("assets"))))

	connectdb()
	routes()
	defer db.Close()

	fmt.Println("Server running on port :4000")
	http.ListenAndServe(":4000", nil)
}

func connectdb() {
	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error getting env, not comming through %v", err)
	} else {
		fmt.Println("We are getting the env values")
	}

	DBURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	db, err = sql.Open(os.Getenv("DB_DRIVER"), DBURL)

	if err != nil {
		log.Fatalln(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}
}

func routes() {
	http.HandleFunc("/", home)
	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/edit", edit)
	http.HandleFunc("/delete", Delete)
}

func QueryUser(username string) user {
	var users = user{}
	err = db.QueryRow(`
		SELECT id, 
		username, 
		first_name, 
		last_name, 
		password, 
		group_id
		FROM users WHERE username=?
		`, username).
		Scan(
			&users.ID,
			&users.Username,
			&users.FirstName,
			&users.LastName,
			&users.Password,
			&users.Group_id,
		)
	return users
}

func home(w http.ResponseWriter, r *http.Request) {

	session := sessions.Start(w, r)
	if len(session.GetString("username")) == 0 {
		http.Redirect(w, r, "/login", 301)
	}

	var users user
	var arruser []user
	groupID := session.GetString("group_id")

	if groupID == "1" {
		rows, errordb := db.Query("Select id,username,first_name,last_name,password, group_id from users")
		if errordb != nil {
			log.Print(errordb)
		}

		for rows.Next() {
			if errrows := rows.Scan(&users.ID, &users.Username, &users.FirstName, &users.LastName, &users.Password, &users.Group_id); errrows != nil {
				log.Fatal(errrows.Error())

			} else {
				arruser = append(arruser, users)
			}
		}
	} else {
		err = db.QueryRow(`
			SELECT id, 
			username, 
			first_name, 
			last_name, 
			password, 
			group_id
			FROM users WHERE group_id=?
			`, groupID).
			Scan(
				&users.ID,
				&users.Username,
				&users.FirstName,
				&users.LastName,
				&users.Password,
				&users.Group_id,
			)
		arruser = append(arruser, users)
	}

	var data = M{"title": "Learning web Go", "name": session.GetString("username"), "data": arruser}

	var tmpl = template.Must(template.ParseFiles(
		"views/index.html",
		"views/_header.html",
		"views/_message.html",
		"views/_footer.html",
	))

	var err = tmpl.ExecuteTemplate(w, "index", data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	session := sessions.Start(w, r)
	if len(session.GetString("username")) != 0 && checkErr(w, r, err) {
		http.Redirect(w, r, "/", 302)
	}

	if r.Method != "POST" {
		http.ServeFile(w, r, "views/login.html")
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	users := QueryUser(username)

	var passwordtes = bcrypt.CompareHashAndPassword([]byte(users.Password), []byte(password))

	if passwordtes == nil {
		session := sessions.Start(w, r)
		session.Set("username", users.Username)
		session.Set("name", users.FirstName)
		session.Set("group_id", users.Group_id)
		session.Set("id", users.ID)
		http.Redirect(w, r, "/", 302)
	} else {
		http.Redirect(w, r, "/login", 302)
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.ServeFile(w, r, "views/register.html")
		return
	}

	username := r.FormValue("email")
	firstname := r.FormValue("first_name")
	lastname := r.FormValue("last_name")
	password := r.FormValue("password")

	users := QueryUser(username)

	if (user{}) == users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		if len(hashedPassword) != 0 && checkErr(w, r, err) {
			stmt, err := db.Prepare("INSERT INTO users SET username=?, password=?, firstname=?, lastname=?")
			if err == nil {
				_, err := stmt.Exec(&username, &hashedPassword, &firstname, &lastname)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}
	} else {
		http.Redirect(w, r, "/register", 302)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session := sessions.Start(w, r)
	session.Clear()
	sessions.Destroy(w, r)
	http.Redirect(w, r, "/", 302)
}

func checkErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err != nil {

		fmt.Println(r.Host + r.URL.Path)

		http.Redirect(w, r, r.Host+r.URL.Path, 301)
		return false
	}

	return true
}

func edit(w http.ResponseWriter, r *http.Request) {
	session := sessions.Start(w, r)
	if len(session.GetString("username")) == 0 {
		http.Redirect(w, r, "/login", 301)
	}

	if r.Method != "POST" {

		nID := r.URL.Query().Get("id")
		selDB, errselDB := db.Query("SELECT * FROM users WHERE id=?", nID)
		if errselDB != nil {
			panic(errselDB.Error())
		}
		users := user{}
		for selDB.Next() {
			var id int
			var username, firstname, lastname, password, group_id string
			errdbscan := selDB.Scan(&id, &username, &firstname, &lastname, &password, &group_id)
			if errdbscan != nil {
				panic(errdbscan.Error())
			}
			users.ID = id
			users.Username = username
			users.FirstName = firstname
			users.LastName = lastname
			users.Password = password
			users.Group_id = group_id
		}

		var data = M{"title": "Learning web Go", "name": session.GetString("username"), "data": users}

		var tmpl = template.Must(template.ParseFiles(
			"views/edit.html",
			"views/_header.html",
			"views/_message.html",
			"views/_footer.html",
		))

		var err = tmpl.ExecuteTemplate(w, "edit", data)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	username := r.FormValue("email")
	password := r.FormValue("password")
	firstname := r.FormValue("first_name")
	lastname := r.FormValue("last_name")
	id := r.FormValue("uid")

	if len(password) > 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		insForm, errupd := db.Prepare("UPDATE users SET username=?, password=?, first_name=?, last_name=? WHERE id=?")
		if errupd != nil {
			panic(errupd.Error())
		}
		insForm.Exec(username, hashedPassword, firstname, lastname, id)
		log.Println("UPDATE: Username: " + username + " | password: " + password + " | first_name: " + firstname + " | last_name: " + lastname)
	} else {
		insForm, errupd := db.Prepare("UPDATE users SET username=?, first_name=?, last_name=? WHERE id=?")
		if errupd != nil {
			panic(errupd.Error())
		}
		insForm.Exec(username, firstname, lastname, id)
		log.Println("UPDATE: Username: " + username + " | first_name: " + firstname + " | last_name: " + lastname)
	}

	http.Redirect(w, r, "/", 301)

}

func Delete(w http.ResponseWriter, r *http.Request) {
	emp := r.URL.Query().Get("id")
	delForm, err := db.Prepare("DELETE FROM users WHERE id=?")
	if err != nil {
		panic(err.Error())
	}
	delForm.Exec(emp)
	log.Println("DELETE")
	http.Redirect(w, r, "/", 301)
}
