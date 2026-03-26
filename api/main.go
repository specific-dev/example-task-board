package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

type User struct {
	ID        int    `json:"id"`
	GoogleID  string `json:"google_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type contextKey string

const userIDKey contextKey = "user_id"

var (
	db          *pgxpool.Pool
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	syncURL     string
	syncSecret  string
)

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, int(userID))
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getUserID(r *http.Request) int {
	return r.Context().Value(userIDKey).(int)
}

func generateJWT(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func generateStateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
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

	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET is required")
	}

	webURL := os.Getenv("WEB_URL")
	if webURL == "" {
		log.Fatal("WEB_URL is required")
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientID == "" || googleClientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required")
	}

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		// Derive from PORT for dev
		apiURL = fmt.Sprintf("http://localhost:%s", port)
	}

	syncURL = os.Getenv("DATABASE_SYNC_URL")
	syncSecret = os.Getenv("DATABASE_SYNC_SECRET")

	oauthConfig = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  apiURL + "/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	var err error
	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /auth/google", handleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", handleGoogleCallback(webURL))
	mux.HandleFunc("GET /auth/me", authMiddleware(handleGetMe))

	// Task routes (all authenticated)
	mux.HandleFunc("GET /tasks", authMiddleware(handleGetTasks))
	mux.HandleFunc("POST /tasks", authMiddleware(handleCreateTask))
	mux.HandleFunc("DELETE /tasks/{id}", authMiddleware(handleDeleteTask))

	// Sync proxy (authenticated)
	mux.HandleFunc("GET /sync/tasks", authMiddleware(handleSyncTasks))

	log.Printf("API server listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, cors(mux)))
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateStateToken()
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(webURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		token, err := oauthConfig.Exchange(r.Context(), code)
		if err != nil {
			log.Printf("OAuth exchange error: %v", err)
			http.Error(w, "oauth exchange failed", http.StatusInternalServerError)
			return
		}

		client := oauthConfig.Client(r.Context(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			log.Printf("Failed to get user info: %v", err)
			http.Error(w, "failed to get user info", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "failed to read user info", http.StatusInternalServerError)
			return
		}

		var googleUser GoogleUserInfo
		if err := json.Unmarshal(body, &googleUser); err != nil {
			http.Error(w, "failed to parse user info", http.StatusInternalServerError)
			return
		}

		// Upsert user
		var userID int
		err = db.QueryRow(r.Context(),
			`INSERT INTO users (google_id, email, name, avatar_url)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (google_id) DO UPDATE SET email = $2, name = $3, avatar_url = $4
			 RETURNING id`,
			googleUser.ID, googleUser.Email, googleUser.Name, googleUser.Picture,
		).Scan(&userID)
		if err != nil {
			log.Printf("User upsert error: %v", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		jwtToken, err := generateJWT(userID)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, webURL+"/auth/callback?token="+jwtToken, http.StatusTemporaryRedirect)
	}
}

func handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var user User
	err := db.QueryRow(r.Context(),
		"SELECT id, google_id, email, name, avatar_url FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func handleGetTasks(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	rows, err := db.Query(r.Context(),
		"SELECT id, title, description, status, tags FROM tasks WHERE user_id = $1 ORDER BY id",
		userID,
	)
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
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

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
		"INSERT INTO tasks (title, description, status, tags, user_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		t.Title, t.Description, t.Status, tagsJSON, userID,
	).Scan(&t.ID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		log.Printf("POST /tasks insert error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")

	tag, err := db.Exec(r.Context(), "DELETE FROM tasks WHERE id = $1 AND user_id = $2", id, userID)
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
}

func handleSyncTasks(w http.ResponseWriter, r *http.Request) {
	if syncURL == "" {
		http.Error(w, "sync not configured", http.StatusServiceUnavailable)
		return
	}

	userID := getUserID(r)

	// Build upstream URL with server-controlled params
	upstream := fmt.Sprintf("%s/v1/shape", syncURL)
	params := fmt.Sprintf("table=tasks&where=user_id=%d&secret=%s", userID, syncSecret)

	// Forward only Electric protocol params from client
	for _, key := range []string{"offset", "handle", "live", "cursor"} {
		if val := r.URL.Query().Get(key); val != "" {
			params += "&" + key + "=" + val
		}
	}

	reqURL := upstream + "?" + params
	upstreamReq, err := http.NewRequestWithContext(r.Context(), "GET", reqURL, nil)
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		http.Error(w, "sync engine unavailable", http.StatusBadGateway)
		log.Printf("Sync proxy error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Expose Electric headers, strip upstream CORS/caching
	electricHeaders := []string{
		"electric-handle", "electric-offset", "electric-schema",
		"electric-cursor", "electric-up-to-date", "electric-chunk-last-offset",
	}
	exposeList := strings.Join(electricHeaders, ", ")

	for _, h := range electricHeaders {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}

	// Copy content type
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	// Set our own CORS/caching headers (strip upstream ones)
	w.Header().Set("Access-Control-Expose-Headers", exposeList)
	w.Header().Set("Cache-Control", "no-store")

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
