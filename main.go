// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

type Todo struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	IsCompleted bool   `json:"isCompleted"`
}

var templates map[string]*template.Template
var conn *pgx.Conn

func init() {
	godotenv.Load() // Load .env file

	// Build the connection string
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Note: Be cautious with direct string manipulation for connection strings in production code to avoid issues with special characters.
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	conn, err = pgx.Connect(context.Background(), connectionString)
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}

	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["index.html"] = template.Must(template.ParseFiles("index.html"))
	templates["todo.html"] = template.Must(template.ParseFiles("todo.html"))
}

// fetch
func fetchTodos() ([]Todo, error) {
	var todos []Todo
	rows, err := conn.Query(context.Background(), "SELECT id, name, is_completed FROM todos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.Id, &t.Name, &t.IsCompleted); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, nil
}

// handlers
func submitTodoHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	completed := r.PostFormValue("completed") == "true"
	var lastInsertId int
	err := conn.QueryRow(context.Background(), "INSERT INTO todos (name, is_completed) VALUES ($1, $2) RETURNING id", name, completed).Scan(&lastInsertId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	todo := Todo{Id: lastInsertId, Name: name, IsCompleted: completed}
	tmpl := templates["todo.html"]
	tmpl.ExecuteTemplate(w, "todo.html", todo)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	todos, err := fetchTodos()
	if err != nil {
		log.Fatal(err)
	}
	jsonTodos, err := json.Marshal(todos)
	if err != nil {
		log.Fatal(err)
	}

	tmpl := templates["index.html"]
	tmpl.ExecuteTemplate(w, "index.html", map[string]template.JS{"Todos": template.JS(jsonTodos)})
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/submit-todo/", submitTodoHandler)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
