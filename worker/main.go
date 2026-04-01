package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"golang.org/x/image/draw"
)

const (
	TaskQueue       = "thumbnail-tasks"
	ThumbnailMaxDim = 200
)

type ThumbnailInput struct {
	AttachmentID int `json:"attachment_id"`
}

type Activities struct {
	db       *pgxpool.Pool
	s3Client *minio.Client
	s3Bucket string
}

func GenerateThumbnailWorkflow(ctx workflow.Context, input ThumbnailInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})
	var a *Activities
	return workflow.ExecuteActivity(ctx, a.GenerateThumbnail, input).Get(ctx, nil)
}

func (a *Activities) GenerateThumbnail(ctx context.Context, input ThumbnailInput) error {
	logger := activity.GetLogger(ctx)

	// 1. Get attachment from DB
	var s3Key, contentType string
	err := a.db.QueryRow(ctx,
		"SELECT s3_key, content_type FROM attachments WHERE id = $1",
		input.AttachmentID,
	).Scan(&s3Key, &contentType)
	if err != nil {
		return fmt.Errorf("fetch attachment: %w", err)
	}

	if !strings.HasPrefix(contentType, "image/") {
		return nil
	}

	// 2. Download original from S3
	obj, err := a.s3Client.GetObject(ctx, a.s3Bucket, s3Key, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3 get: %w", err)
	}
	defer obj.Close()

	// 3. Decode image
	src, _, err := image.Decode(obj)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// 4. Calculate thumbnail dimensions (fit within ThumbnailMaxDim)
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	newW, newH := ThumbnailMaxDim, ThumbnailMaxDim
	if srcW > srcH {
		newH = srcH * ThumbnailMaxDim / srcW
	} else {
		newW = srcW * ThumbnailMaxDim / srcH
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	// 5. Resize with high-quality interpolation
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Rect, src, bounds, draw.Over, nil)

	// 6. Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return fmt.Errorf("encode thumbnail: %w", err)
	}

	// 7. Upload thumbnail to S3
	thumbKey := fmt.Sprintf("thumbnails/%d.jpg", input.AttachmentID)
	_, err = a.s3Client.PutObject(ctx, a.s3Bucket, thumbKey,
		bytes.NewReader(buf.Bytes()), int64(buf.Len()),
		minio.PutObjectOptions{
			ContentType:          "image/jpeg",
			DisableContentSha256: true,
			SendContentMd5:       true,
		},
	)
	if err != nil {
		return fmt.Errorf("upload thumbnail: %w", err)
	}

	// 8. Update DB with thumbnail key
	_, err = a.db.Exec(ctx,
		"UPDATE attachments SET thumbnail_s3_key = $1 WHERE id = $2",
		thumbKey, input.AttachmentID,
	)
	if err != nil {
		return fmt.Errorf("update attachment: %w", err)
	}

	logger.Info("Generated thumbnail", "attachment_id", input.AttachmentID)
	return nil
}

func main() {
	temporalAddr := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddr == "" {
		log.Fatal("TEMPORAL_ADDRESS is required")
	}
	temporalNS := os.Getenv("TEMPORAL_NAMESPACE")
	if temporalNS == "" {
		temporalNS = "default"
	}
	temporalAPIKey := os.Getenv("TEMPORAL_API_KEY")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3BucketName := os.Getenv("S3_BUCKET")

	ctx := context.Background()

	// Database
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("DB connect: %v", err)
	}
	defer pool.Close()

	// S3
	endpoint := strings.TrimPrefix(strings.TrimPrefix(s3Endpoint, "https://"), "http://")
	useSSL := strings.HasPrefix(s3Endpoint, "https://")
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("S3 client: %v", err)
	}

	// Temporal client
	opts := client.Options{
		HostPort:  temporalAddr,
		Namespace: temporalNS,
	}
	if temporalAPIKey != "" {
		opts.Credentials = client.NewAPIKeyStaticCredentials(temporalAPIKey)
	}
	c, err := client.Dial(opts)
	if err != nil {
		log.Fatalf("Temporal client: %v", err)
	}
	defer c.Close()

	// Start worker
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflow(GenerateThumbnailWorkflow)
	w.RegisterActivity(&Activities{
		db:       pool,
		s3Client: s3Client,
		s3Bucket: s3BucketName,
	})

	log.Println("Thumbnail worker starting...")
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
