package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Tag struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Tags        []Tag  `json:"tags"`
}

var db *pgxpool.Pool

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	var err error
	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(r.Context(), "SELECT id, title, description, status, tags FROM tasks ORDER BY id")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			log.Printf("GET /tasks query error: %v", err)
			return
		}
		defer rows.Close()

		tasks := []Task{}
		for rows.Next() {
			var t Task
			var tagsJSON []byte
			if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &tagsJSON); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				log.Printf("GET /tasks scan error: %v", err)
				return
			}
			if err := json.Unmarshal(tagsJSON, &t.Tags); err != nil {
				t.Tags = []Tag{}
			}
			tasks = append(tasks, t)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	})

	mux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		var t Task
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if t.Title == "" || t.Status == "" {
			http.Error(w, "title and status are required", http.StatusBadRequest)
			return
		}
		if t.Tags == nil {
			t.Tags = []Tag{}
		}

		tagsJSON, err := json.Marshal(t.Tags)
		if err != nil {
			http.Error(w, "invalid tags", http.StatusBadRequest)
			return
		}

		err = db.QueryRow(r.Context(),
			"INSERT INTO tasks (title, description, status, tags) VALUES ($1, $2, $3, $4) RETURNING id",
			t.Title, t.Description, t.Status, tagsJSON,
		).Scan(&t.ID)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			log.Printf("POST /tasks insert error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)
	})

	mux.HandleFunc("DELETE /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")

		tag, err := db.Exec(r.Context(), "DELETE FROM tasks WHERE id = $1", id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			log.Printf("DELETE /tasks/%s error: %v", id, err)
			return
		}
		if tag.RowsAffected() == 0 {
			http.Error(w, fmt.Sprintf("task %s not found", id), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	log.Printf("API server listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, cors(mux)))
}
