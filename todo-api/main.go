// todo-api/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

type Todo struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	IsCompleted bool   `json:"isCompleted"`
}

var conn *pgx.Conn

func init() {
	godotenv.Load() // Load .env file

	// Build the connection string
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	connectionString := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"

	var err error
	conn, err = pgx.Connect(context.Background(), connectionString)
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
}

// fetch
func fetchTodos(w http.ResponseWriter, r *http.Request) {
	var todos []Todo
	rows, err := conn.Query(context.Background(), "SELECT id, name, is_completed FROM todos")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.Id, &t.Name, &t.IsCompleted); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		todos = append(todos, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

// handlers
func submitTodoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received request: %+v", r)
	name := r.FormValue("name")
	completed := r.FormValue("completed") == "true"
	var lastInsertId int
	err := conn.QueryRow(context.Background(), "INSERT INTO todos (name, is_completed) VALUES ($1, $2) RETURNING id", name, completed).Scan(&lastInsertId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	todo := Todo{Id: lastInsertId, Name: name, IsCompleted: completed}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/todos", fetchTodos).Methods("GET")
	r.HandleFunc("/submit-todo", submitTodoHandler).Methods("POST")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080"},
		AllowCredentials: true,
		AllowedMethods:   []string{"POST", "GET", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
	})
	handler := c.Handler(r)

	log.Fatal(http.ListenAndServe(":8000", handler))
}
