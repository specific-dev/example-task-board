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
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

type Attachment struct {
	ID          int       `json:"id"`
	TaskID      int       `json:"task_id"`
	UserID      int       `json:"user_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	S3Key       string    `json:"s3_key"`
	CreatedAt   time.Time `json:"created_at"`
}

type contextKey string

const userIDKey contextKey = "user_id"

var (
	db          *pgxpool.Pool
	s3Client    *minio.Client
	s3Bucket    string
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	syncURL     string
	syncSecret  string
)

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
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

	// S3 setup
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Bucket = os.Getenv("S3_BUCKET")

	if s3Endpoint != "" {
		// Strip protocol for minio client
		endpoint := strings.TrimPrefix(strings.TrimPrefix(s3Endpoint, "https://"), "http://")
		useSSL := strings.HasPrefix(s3Endpoint, "https://")
		var err error
		s3Client, err = minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			log.Fatalf("Failed to create S3 client: %v", err)
		}
		// Ensure bucket exists
		ctx := context.Background()
		exists, err := s3Client.BucketExists(ctx, s3Bucket)
		if err != nil {
			log.Fatalf("Failed to check bucket: %v", err)
		}
		if !exists {
			if err := s3Client.MakeBucket(ctx, s3Bucket, minio.MakeBucketOptions{}); err != nil {
				log.Fatalf("Failed to create bucket: %v", err)
			}
		}
		log.Printf("S3 storage configured: %s/%s", s3Endpoint, s3Bucket)
	}

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
	mux.HandleFunc("PATCH /tasks/{id}", authMiddleware(handleUpdateTask))
	mux.HandleFunc("DELETE /tasks/{id}", authMiddleware(handleDeleteTask))

	// Attachment routes (all authenticated)
	mux.HandleFunc("POST /tasks/{id}/attachments", authMiddleware(handleUploadAttachment))
	mux.HandleFunc("DELETE /attachments/{id}", authMiddleware(handleDeleteAttachment))
	mux.HandleFunc("GET /attachments/{id}/download", authMiddleware(handleDownloadAttachment))

	// Sync proxy (authenticated)
	mux.HandleFunc("GET /sync/tasks", authMiddleware(handleSyncTasks))
	mux.HandleFunc("GET /sync/attachments", authMiddleware(handleSyncAttachments))

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

func handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id := r.PathValue("id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	tag, err := db.Exec(r.Context(),
		"UPDATE tasks SET status = $1 WHERE id = $2 AND user_id = $3",
		body.Status, id, userID,
	)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		log.Printf("PATCH /tasks/%s error: %v", id, err)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, fmt.Sprintf("task %s not found", id), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	if s3Client == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	userID := getUserID(r)
	taskID := r.PathValue("id")

	// Verify the task belongs to the user
	var exists bool
	err := db.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2)", taskID, userID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	// 10 MB max
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "file too large (max 10MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	s3Key := fmt.Sprintf("attachments/%s/%d_%s", taskID, time.Now().UnixNano(), header.Filename)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s3Client.PutObject(r.Context(), s3Bucket, s3Key, file, header.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		log.Printf("S3 upload error: %v", err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	var att Attachment
	err = db.QueryRow(r.Context(),
		`INSERT INTO attachments (task_id, user_id, filename, content_type, size, s3_key)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, task_id, user_id, filename, content_type, size, s3_key, created_at`,
		taskID, userID, header.Filename, contentType, header.Size, s3Key,
	).Scan(&att.ID, &att.TaskID, &att.UserID, &att.Filename, &att.ContentType, &att.Size, &att.S3Key, &att.CreatedAt)
	if err != nil {
		log.Printf("Insert attachment error: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(att)
}

func handleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	if s3Client == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	userID := getUserID(r)
	id := r.PathValue("id")

	// Get the attachment to find its S3 key, and verify ownership via task
	var s3Key string
	err := db.QueryRow(r.Context(),
		`SELECT a.s3_key FROM attachments a
		 JOIN tasks t ON t.id = a.task_id
		 WHERE a.id = $1 AND t.user_id = $2`,
		id, userID,
	).Scan(&s3Key)
	if err != nil {
		http.Error(w, "attachment not found", http.StatusNotFound)
		return
	}

	// Delete from S3
	if err := s3Client.RemoveObject(r.Context(), s3Bucket, s3Key, minio.RemoveObjectOptions{}); err != nil {
		log.Printf("S3 delete error: %v", err)
	}

	// Delete from DB
	_, err = db.Exec(r.Context(), "DELETE FROM attachments WHERE id = $1", id)
	if err != nil {
		log.Printf("Delete attachment DB error: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	if s3Client == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	userID := getUserID(r)
	id := r.PathValue("id")

	var att Attachment
	err := db.QueryRow(r.Context(),
		`SELECT a.id, a.filename, a.content_type, a.s3_key FROM attachments a
		 JOIN tasks t ON t.id = a.task_id
		 WHERE a.id = $1 AND t.user_id = $2`,
		id, userID,
	).Scan(&att.ID, &att.Filename, &att.ContentType, &att.S3Key)
	if err != nil {
		http.Error(w, "attachment not found", http.StatusNotFound)
		return
	}

	obj, err := s3Client.GetObject(r.Context(), s3Bucket, att.S3Key, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("S3 download error: %v", err)
		http.Error(w, "download failed", http.StatusInternalServerError)
		return
	}
	defer obj.Close()

	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Filename))
	io.Copy(w, obj)
}

func proxySyncShape(w http.ResponseWriter, r *http.Request, table string, where string) {
	if syncURL == "" {
		http.Error(w, "sync not configured", http.StatusServiceUnavailable)
		return
	}

	// Build upstream URL with server-controlled params
	upstream := fmt.Sprintf("%s/v1/shape", syncURL)
	params := fmt.Sprintf("table=%s&where=%s&secret=%s", table, where, syncSecret)

	// Forward only Electric protocol params from client
	for _, key := range []string{"offset", "handle", "live", "live_sse", "cursor", "expired_handle", "replica", "log"} {
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

func handleSyncTasks(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	proxySyncShape(w, r, "tasks", fmt.Sprintf("user_id=%d", userID))
}

func handleSyncAttachments(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	proxySyncShape(w, r, "attachments", fmt.Sprintf("user_id=%d", userID))
}
