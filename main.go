package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Simple Task Modeling
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // "todo", "in-progress", "done"
	CreatedAt time.Time `json:"created_at"`
}

// Global In-Memory Store for TaskBoard Data (Thread-Safe)
var (
	tasks      = make(map[string]Task)
	tasksMutex sync.RWMutex
)

// Mobile Socket Payload Framing
type SocketMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default fallback for local testing
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Performance Middlewares
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} | ${latency} | ${method} ${path}\n",
	}))

	// Seed data for the TaskBoard reference project
	tasks["1"] = Task{ID: "1", Title: "Initialize Framework Design", Status: "done", CreatedAt: time.Now()}
	tasks["2"] = Task{ID: "2", Title: "Optimize Cold Start Engine", Status: "in-progress", CreatedAt: time.Now()}

	// --- 1. CORE INFRASTRUCTURE & HEALTH ---
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"status": "healthy",
			"uptime": "100%",
		})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).SendString("🚀 QuikDB TaskBoard Engine Online!")
	})

	// --- 2. AUTHENTICATION ---
	app.Post("/api/auth/login", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"token":   "quik_jwt_secure_session_token_xyz123",
			"user_id": "user_01",
		})
	})

	// --- 3. TASKBOARD CRUD ENDPOINTS ---
	// Get All Tasks
	app.Get("/api/tasks", func(c *fiber.Ctx) error {
		tasksMutex.RLock()
		defer tasksMutex.RUnlock()

		var taskList []Task
		for _, task := range tasks {
			taskList = append(taskList, task)
		}
		return c.JSON(taskList)
	})

	// Create New Task
	app.Post("/api/tasks", func(c *fiber.Ctx) error {
		var input struct {
			Title string `json:"title"`
		}
		if err := c.BodyParser(&input); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input layout"})
		}

		tasksMutex.Lock()
		id := time.Now().Format("150405") // Quick generation strategy
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

	// Analytics Summary
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
			"efficiency":      "100%",
		})
	})

	// --- 4. MOBILE MOBILE READY HOOKS ---
	// OTA Update Check
	app.Get("/api/mobile/ota-check", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"current_version":  "1.0.4",
			"update_available": false,
		})
	})

	// Offline Data Sync Endpoint
	app.Post("/api/mobile/sync", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"status":    "synchronized",
			"server_ts": time.Now().Unix(),
		})
	})

	// --- 5. NATIVE WEBSOCKET CHAT / ENGINE ---
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(wsc *websocket.Conn) {
		log.Println("📱 Phone connected to WebSocket sync node.")
		defer wsc.Close()

		for {
			mt, msg, err := wsc.ReadMessage()
			if err != nil {
				break
			}

			var incoming SocketMessage
			if err := json.Unmarshal(msg, &incoming); err != nil {
				continue
			}

			// Handle Realtime Interactions (e.g., Ping or Task Sync updates)
			if incoming.Action == "ping" {
				response, _ := json.Marshal(map[string]string{"action": "pong"})
				_ = wsc.WriteMessage(mt, response)
			}
		}
	}))

	log.Printf("⚡ TaskBoard App running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}