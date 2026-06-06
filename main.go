package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Task Represents our internal structural data mapping
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // "todo", "in-progress", "done"
	CreatedAt time.Time `json:"created_at"`
}

// ExternalTask represents the structure of the free incoming API data
type ExternalTask struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// Global In-Memory Store (Thread-Safe)
var (
	tasks      = make(map[string]Task)
	tasksMutex sync.RWMutex
)

type SocketMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

// Fetch data from free external API on startup
func fetchAndSeedTasks() {
	log.Println("🌐 Fetching external seed data from free API...")
	
	// Create client timeout to prevent cold start hanging
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get("https://jsonplaceholder.typicode.com/todos?_limit=5")
	if err != nil {
		log.Println("⚠️ API fetch failed, defaulting to local seed data:", err)
		seedFallbackData()
		return
	}
	defer resp.Body.Close()

	var externalTasks []ExternalTask
	if err := json.NewDecoder(resp.Body).Decode(&externalTasks); err != nil {
		log.Println("⚠️ Failed to parse API payload:", err)
		seedFallbackData()
		return
	}

	// Lock store and ingest external data transforms
	tasksMutex.Lock()
	for _, ext := range externalTasks {
		status := "todo"
		if ext.Completed {
			status = "done"
		}
		
		// Corrected string translation format
		idStr := strconv.Itoa(ext.ID) 
		
		tasks[idStr] = Task{
			ID:        idStr,
			Title:     "[API Sync] " + ext.Title,
			Status:    status,
			CreatedAt: time.Now(),
		}
	}
	tasksMutex.Unlock()
	log.Printf("✅ Successfully ingested %d tasks from external API!\n", len(externalTasks))
}

func seedFallbackData() {
	tasksMutex.Lock()
	tasks["1"] = Task{ID: "1", Title: "[Fallback] Setup Framework Design", Status: "done", CreatedAt: time.Now()}
	tasks["2"] = Task{ID: "2", Title: "[Fallback] Optimize Cold Start Engine", Status: "in-progress", CreatedAt: time.Now()}
	tasksMutex.Unlock()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Run the API data worker instantly before boot to optimize storage preparation
	fetchAndSeedTasks()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} | ${latency} | ${method} ${path}\n",
	}))

	// --- CORE INFRASTRUCTURE & HEALTH ---
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "healthy", "uptime": "100%"})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).SendString("🚀 QuikDB TaskBoard Engine Online!")
	})

	// --- TASKBOARD CRUD ENDPOINTS ---
	app.Get("/api/tasks", func(c *fiber.Ctx) error {
		tasksMutex.RLock()
		defer tasksMutex.RUnlock()

		var taskList []Task
		for _, task := range tasks {
			taskList = append(taskList, task)
		}
		return c.JSON(taskList)
	})

	app.Post("/api/tasks", func(c *fiber.Ctx) error {
		var input struct {
			Title string `json:"title"`
		}
		if err := c.BodyParser(&input); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input layout"})
		}

		tasksMutex.Lock()
		id := time.Now().Format("150405")
		newTask := Task{
			ID:        id,
			Title:     input.Title,
			Status:    "todo",
			CreatedAt: time.Now(),
		}
		tasks[id] = newTask
		tasksMutex.Unlock()

		return c.Status(http.StatusCreated).JSON(newTask)
	})

	app.Get("/api/analytics", func(c *fiber.Ctx) error {
		tasksMutex.RLock()
		defer tasksMutex.RUnlock()

		total := len(tasks)
		doneCount := 0
		for _, t := range tasks {
			if t.Status == "done" {
				doneCount++
			}
		}

		return c.JSON(fiber.Map{
			"total_tasks":     total,
			"completed_tasks": doneCount,
			"data_source":     "JSONPlaceholder API Mixed Matrix",
		})
	})

	// --- MOBILE & WEBSOCKET GATEWAYS ---
	app.Get("/api/mobile/ota-check", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"current_version": "1.0.4", "update_available": false})
	})

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) { return c.Next() }
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(wsc *websocket.Conn) {
		defer wsc.Close()
		for {
			mt, msg, err := wsc.ReadMessage()
			if err != nil { break }
			var incoming SocketMessage
			if err := json.Unmarshal(msg, &incoming); err == nil && incoming.Action == "ping" {
				response, _ := json.Marshal(map[string]string{"action": "pong"})
				_ = wsc.WriteMessage(mt, response)
			}
		}
	}))

	log.Printf("⚡ TaskBoard App running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
