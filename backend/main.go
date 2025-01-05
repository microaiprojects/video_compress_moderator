package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"bytes"
	"io"
	"encoding/json"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Video struct {
	ID           string  `json:"id"`
	Path         string  `json:"path"`
	Resolution   string  `json:"resolution"`
	Bitrate      string  `json:"bitrate"`
	Status       string  `json:"status"`
	OriginalSize int64   `json:"originalSize"`
}

type VideoQueue struct {
	Pending    []Video `json:"pending"`
	Processing []Video `json:"processing"`
	Completed  []Video `json:"completed"`
	Total      int     `json:"total"`
}

type PaginationParams struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=10"`
	Status   string `form:"status"`
}

type LastProcessedTime struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
}

const lastProcessedTimeFile = "last_processed_time.json"

var db *sql.DB          // локальная база данных
var immichDB *sql.DB    // удаленная база данных Immich

func loadLastProcessedTime() time.Time {
	data, err := os.ReadFile(lastProcessedTimeFile)
	if err != nil {
		return time.Time{}
	}
	var lastTime LastProcessedTime
	if err := json.Unmarshal(data, &lastTime); err != nil {
		return time.Time{}
	}
	return lastTime.LastCreatedAt
}

func saveLastProcessedTime(t time.Time) error {
	data, err := json.Marshal(LastProcessedTime{LastCreatedAt: t})
	if err != nil {
		return err
	}
	return os.WriteFile(lastProcessedTimeFile, data, 0644)
}

func initDB() {
	var err error
	
	// Подключение к локальной БД
	localDBURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		//"localhost",
		//"5432",
		//"postgres",
		//"postgres",
		//"videoqueue",
	)
	fmt.Println(localDBURL)
	
	db, err = sql.Open("postgres", localDBURL)
	if err != nil {
		log.Fatal("Error connecting to local DB:", err)
	}

	// Подключение к удаленной БД Immich
	immichDBURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("IMMICH_DB_HOST"),
		os.Getenv("IMMICH_DB_PORT"),
		os.Getenv("IMMICH_DB_USERNAME"),
		os.Getenv("IMMICH_DB_PASSWORD"),
		os.Getenv("IMMICH_DB_NAME"),
	)
	fmt.Println(immichDBURL)
	
	immichDB, err = sql.Open("postgres", immichDBURL)
	if err != nil {
		log.Fatal("Error connecting to Immich DB:", err)
	}

	// Create videos table if not exists в локальной БД
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			resolution TEXT,
			bitrate TEXT,
			status TEXT NOT NULL,
			original_size BIGINT NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create index on path and status
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_videos_path ON videos(path);
		CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize processed paths cache
	refreshProcessedPathsCache()
}

func refreshProcessedPathsCache() {
	processedPaths := make(map[string]bool)
	rows, err := db.Query("SELECT path FROM videos")
	if err != nil {
		log.Printf("Error refreshing paths cache: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			log.Printf("Error scanning path: %v", err)
			continue
		}
		processedPaths[path] = true
	}
}

func getUnprocessedVideos(c *gin.Context) {
	lastProcessed := loadLastProcessedTime()
	
	// Получаем новые видео из Immich
	query := `
		SELECT id, "originalPath", "createdAt", "originalFileName"
		FROM assets 
		WHERE type = 'VIDEO' 
		AND status = 'active' 
		AND "createdAt" > $1 
		AND "originalFileName" NOT LIKE '%.mkv'
		ORDER BY "createdAt" ASC
		LIMIT 100`

	rows, err := immichDB.Query(query, lastProcessed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var latestCreatedAt time.Time

	// Обработка новых видео из Immich
	for rows.Next() {
		var videoID, originalPath string
		var createdAt time.Time
		err := rows.Scan(&videoID, &originalPath, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Replace path prefix
		newPath := strings.Replace(originalPath, 
			os.Getenv("IMMICH_UPLOAD_PATH"), 
			os.Getenv("VIDEO_PATH"), 1)
		
		// Сохраняем новое видео в локальную БД со статусом wait
		_, err = db.Exec(`
			INSERT INTO videos (id, path, status, original_size)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING`,
			videoID, newPath, "wait", getFileSize(newPath))
		if err != nil {
			log.Printf("Error inserting video %s: %v", videoID, err)
		}
		
		if createdAt.After(latestCreatedAt) {
			latestCreatedAt = createdAt
		}
	}

	// Обновляем время последнего обработанного видео
	if !latestCreatedAt.IsZero() {
		if err := saveLastProcessedTime(latestCreatedAt); err != nil {
			log.Printf("Error saving last processed time: %v", err)
		}
	}

	// Получаем все видео со статусом wait из локальной БД
	waitingVideos := []Video{}
	localRows, err := db.Query(`
		SELECT id, path, status, original_size 
		FROM videos 
		WHERE status = 'wait'
		ORDER BY path DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer localRows.Close()

	for localRows.Next() {
		var video Video
		err := localRows.Scan(
			&video.ID,
			&video.Path,
			&video.Status,
			&video.OriginalSize,
		)
		if err != nil {
			log.Printf("Error scanning local video: %v", err)
			continue
		}
		waitingVideos = append(waitingVideos, video)
	}

	c.JSON(http.StatusOK, waitingVideos)
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func getQueue(c *gin.Context) {
	var params PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate and set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 10
	}

	offset := (params.Page - 1) * params.PageSize

	// Build query based on status filter
	baseQuery := "SELECT id, path, resolution, bitrate, status, original_size FROM videos"
	countQuery := "SELECT COUNT(*) FROM videos"
	var whereClause string

	if params.Status != "" {
		whereClause = fmt.Sprintf(" WHERE status = '%s'", params.Status)
	}

	// Get total count
	var total int
	err := db.QueryRow(countQuery + whereClause).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated results
	query := baseQuery + whereClause + 
		" ORDER BY CASE status " +
		"WHEN 'processing' THEN 1 " +
		"WHEN 'pending' THEN 2 " +
		"WHEN 'completed' THEN 3 " +
		"ELSE 4 END, path" +
		fmt.Sprintf(" LIMIT %d OFFSET %d", params.PageSize, offset)

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	videos := scanVideoRows(rows)

	// Group videos by status
	queue := VideoQueue{
		Pending:    []Video{},
		Processing: []Video{},
		Completed:  []Video{},
		Total:      total,
	}

	for _, video := range videos {
		switch video.Status {
		case "pending":
			queue.Pending = append(queue.Pending, video)
		case "processing":
			queue.Processing = append(queue.Processing, video)
		case "completed":
			queue.Completed = append(queue.Completed, video)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"queue": queue,
		"pagination": gin.H{
			"page":     params.Page,
			"pageSize": params.PageSize,
			"total":    total,
			"pages":    (total + params.PageSize - 1) / params.PageSize,
		},
	})
}

func scanVideoRows(rows *sql.Rows) []Video {
	var videos []Video
	for rows.Next() {
		var v Video
		err := rows.Scan(&v.ID, &v.Path, &v.Resolution, &v.Bitrate, &v.Status, &v.OriginalSize)
		if err != nil {
			log.Printf("Error scanning video row: %v", err)
			continue
		}
		videos = append(videos, v)
	}
	return videos
}

func addToQueue(c *gin.Context) {
	var video Video
	if err := c.ShouldBindJSON(&video); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update video in database
	_, err := db.Exec(`
		UPDATE videos 
		SET path = $2, resolution = $3, bitrate = $4, status = $5, original_size = $6
		WHERE id = $1
	`, video.ID, video.Path, video.Resolution, video.Bitrate, "pending", video.OriginalSize)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update processed paths cache
	processedPaths := make(map[string]bool)
	processedPaths[video.Path] = true

	c.JSON(http.StatusOK, video)
}

func updateVideoStatus(c *gin.Context) {
	id := c.Param("id")
	var statusUpdate struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update video status in database
	_, err := db.Exec("UPDATE videos SET status = $1 WHERE id = $2", statusUpdate.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func deleteVideo(c *gin.Context) {
	var request struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Создаем запрос на удаление в Immich
	immichURL := fmt.Sprintf("%s/api/assets", os.Getenv("IMMICH_HOST"))
	payload := map[string][]string{
		"ids": {request.ID},
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req, err := http.NewRequest("DELETE", immichURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", os.Getenv("IMMICH_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	// Удаляем видео из локальной базы данных
	_, err = db.Exec("DELETE FROM videos WHERE id = $1", request.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully"})
}

func main() {
	// Initialize database connection
	initDB()
	defer db.Close()
	defer immichDB.Close()

	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	
	r.Use(cors.New(config))
	r.Use(gin.Recovery())

	// Routes
	r.GET("/api/videos/unprocessed", getUnprocessedVideos)
	r.GET("/api/queue", getQueue)
	r.POST("/api/videos/process", addToQueue)
	r.PATCH("/api/videos/:id/status", updateVideoStatus)
	r.DELETE("/api/videos", deleteVideo)

	r.Run(":8080")
}
