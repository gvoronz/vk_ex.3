package main

import (
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PingResult представляет структуру для хранения результатов пинга
type PingResult struct {
	ID        uint      `gorm:"primaryKey"`
	IPAddress string    `gorm:"not null"`
	PingTime  float64   `gorm:"not null"`
	LastSeen  time.Time `gorm:"not null"`
}

var db *gorm.DB

// initDB инициализирует подключение к базе данных и выполняет миграции
func initDB() {
	dsn := "host=localhost user=youruser password=yourpassword dbname=yourdb port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	db.AutoMigrate(&PingResult{})
}

// getPingResults обрабатывает запросы на получение результатов пинга
func getPingResults(c *gin.Context) {
	var results []PingResult
	if err := db.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
		return
	}
	c.JSON(http.StatusOK, results)
}

// pingContainers выполняет пинг всех контейнеров Docker
func pingContainers() {
	for {
		cmd := exec.Command("docker", "ps", "--format", "{{.ID}}")
		output, err := cmd.Output()
		if err != nil {
			log.Println("Failed to get container list:", err)
			continue
		}
		containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, id := range containerIDs {
			cmd := exec.Command("docker", "inspect", "-f", "{{.NetworkSettings.IPAddress}}", id)
			ipOutput, err := cmd.Output()
			if err != nil {
				log.Println("Failed to get IP for container", id, ":", err)
				continue
			}
			ip := strings.TrimSpace(string(ipOutput))
			pingCmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
			start := time.Now()
			if err := pingCmd.Run(); err == nil {
				pingTime := time.Since(start).Seconds() * 1000
				db.Create(&PingResult{IPAddress: ip, PingTime: pingTime, LastSeen: time.Now()})
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func main() {
	// Инициализация базы данных
	initDB()

	// Запуск горутины для пинга контейнеров
	go pingContainers()

	// Инициализация маршрутов с Gin
	r := gin.Default()

	// Настройка CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"}, // Разрешенные origins
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"}, // Разрешенные методы
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"}, // Разрешенные заголовки
	}))

	// Маршрут для получения результатов пинга
	r.GET("/ping-results", getPingResults)

	// Запуск сервера на порту 8080
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
