package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

var tasks = []Task{
	{
		ID:          "1",
		Title:       "Design system tokens",
		Description: "Define color palette, typography scale, and spacing tokens for the component library.",
		Status:      "todo",
		Tags:        []Tag{{Label: "Design", Color: "#7c3aed"}, {Label: "Foundation", Color: "#0891b2"}},
	},
	{
		ID:          "2",
		Title:       "Set up CI pipeline",
		Description: "Configure GitHub Actions for linting, type checking, and running tests on every PR.",
		Status:      "todo",
		Tags:        []Tag{{Label: "DevOps", Color: "#ea580c"}},
	},
	{
		ID:          "3",
		Title:       "User authentication flow",
		Description: "Implement sign-up, login, and password reset screens with form validation.",
		Status:      "todo",
		Tags:        []Tag{{Label: "Feature", Color: "#2563eb"}, {Label: "Auth", Color: "#dc2626"}},
	},
	{
		ID:          "4",
		Title:       "API rate limiting",
		Description: "Add middleware to enforce per-user rate limits on all public API endpoints.",
		Status:      "in-progress",
		Tags:        []Tag{{Label: "Backend", Color: "#16a34a"}, {Label: "Security", Color: "#dc2626"}},
	},
	{
		ID:          "5",
		Title:       "Dashboard layout",
		Description: "Build the responsive grid layout for the main dashboard with sidebar navigation.",
		Status:      "in-progress",
		Tags:        []Tag{{Label: "Feature", Color: "#2563eb"}, {Label: "UI", Color: "#7c3aed"}},
	},
	{
		ID:          "6",
		Title:       "Database indexing",
		Description: "Optimize slow queries by adding composite indexes on frequently filtered columns.",
		Status:      "done",
		Tags:        []Tag{{Label: "Backend", Color: "#16a34a"}, {Label: "Performance", Color: "#ca8a04"}},
	},
	{
		ID:          "7",
		Title:       "Onboarding tooltip tour",
		Description: "Create a guided tooltip walkthrough for first-time users after sign-up.",
		Status:      "done",
		Tags:        []Tag{{Label: "Feature", Color: "#2563eb"}, {Label: "UX", Color: "#7c3aed"}},
	},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(tasks)
	})

	mux.HandleFunc("OPTIONS /tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
	})

	log.Printf("API server listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
