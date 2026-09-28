package main

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kanban/config"
	"kanban/db"
	"kanban/handlers"
	"kanban/middleware"
	"kanban/services"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist
var frontendFS embed.FS

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	if err := connectDatabase(cfg.DatabaseURL); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := db.RunMigrations(ctx); err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}

	bot := services.NewMaxBot(cfg.MaxBotToken, cfg.BotName, cfg.FrontendURL)
	if cfg.MaxCAFile != "" {
		pool, err := services.LoadCustomCA(cfg.MaxCAFile)
		if err != nil {
			log.Fatalf("Invalid MAX CA file: %v", err)
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{RootCAs: pool}
		bot.HTTPClient.Transport = transport
	}
	notifier := services.NewNotifier(db.Pool, bot)
	services.SetDefaultNotifier(notifier)
	background, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()
	go func() {
		if err := notifier.Run(background); err != nil && err != context.Canceled {
			log.Printf("Notification worker stopped: %v", err)
		}
	}()
	go runDeadlineScheduler(background, notifier)
	go setupMaxIntegration(background, cfg, bot)
	router := setupRouter(cfg, bot)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server listening on :%s", cfg.Port)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	case <-shutdown:
		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}
	}
}

func setupMaxIntegration(ctx context.Context, cfg *config.Config, bot *services.MaxBot) {
	if cfg.MaxBotToken == "" {
		return
	}
	setup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := bot.RegisterCommands(setup); err != nil {
		log.Printf("Register MAX commands failed: %v", err)
	}
	if cfg.MaxWebhookURL != "" {
		if err := bot.EnsureSubscription(setup, cfg.MaxWebhookURL, cfg.MaxWebhookUpdateTypes, cfg.WebhookSecret); err != nil {
			log.Printf("Ensure MAX subscription failed: %v", err)
		}
	}
}

func runDeadlineScheduler(ctx context.Context, notifier *services.Notifier) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if err := notifier.EnqueueDueDeadlines(ctx, time.Now()); err != nil {
			log.Printf("Queue deadline notifications failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func connectDatabase(databaseURL string) error {
	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		attemptContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := db.Connect(attemptContext, databaseURL)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("Database connection attempt %d failed: %v", attempt, err)
		time.Sleep(time.Second)
	}
	return lastErr
}

func setupRouter(cfg *config.Config, bot *services.MaxBot) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORSMiddleware(cfg.AllowedOrigins))

	authHandler := handlers.NewAuthHandler(cfg.MaxBotToken)
	orgHandler := handlers.NewOrgHandler()
	boardHandler := handlers.NewBoardHandler()
	columnHandler := handlers.NewColumnHandler()
	taskHandler := handlers.NewTaskHandler()
	checklistHandler := handlers.NewChecklistHandler()
	commentHandler := handlers.NewCommentHandler()
	boardChatHandler := handlers.NewBoardChatHandler()
	healthHandler := handlers.NewHealthHandler(cfg.Version, cfg.Revision, time.Now())
	webhookHandler := handlers.NewWebhookHandler(bot, cfg.WebhookSecret)

	registerHealth := func(router *gin.Engine, prefix string) {
		router.GET(prefix+"/api/health", healthHandler.Check)
	}
	registerHealth(router, "")
	registerHealth(router, "/max-kanban")

	router.POST("/api/auth/validate", authHandler.Validate)
	router.POST("/max-kanban/api/auth/validate", authHandler.Validate)
	router.POST("/webhook", webhookHandler.Handle)
	router.POST("/max-kanban/webhook", webhookHandler.Handle)
	router.GET("/ws", ws.HandleWebSocket)
	router.GET("/max-kanban/ws", ws.HandleWebSocket)

	registerAPI := func(prefix string) {
		api := router.Group(prefix + "/api")
		api.Use(middleware.AuthMiddleware(cfg.MaxBotToken))
		api.GET("/orgs", orgHandler.List)
		api.POST("/orgs", orgHandler.Create)
		api.GET("/metrics", healthHandler.Metrics)
		api.POST("/ws-ticket", handlers.IssueWebSocketTicket)

		orgs := api.Group("/orgs/:id")
		orgs.GET("", orgHandler.Get)
		orgs.PATCH("", orgHandler.Update)
		orgs.POST("/members", orgHandler.AddMember)
		orgs.DELETE("/members/:uid", orgHandler.RemoveMember)
		orgs.GET("/boards", boardHandler.ListByOrg)
		orgs.POST("/boards", boardHandler.Create)

		boards := api.Group("/boards/:id")
		boards.GET("", boardHandler.Get)
		boards.PATCH("", boardHandler.Update)
		boards.DELETE("", boardHandler.Delete)
		boards.POST("/columns", columnHandler.Create)
		boards.PATCH("/columns/reorder", columnHandler.Reorder)
		boards.GET("/tasks", taskHandler.ListByBoard)
		boards.POST("/tasks", taskHandler.Create)
		boards.GET("/chats", boardChatHandler.List)
		boards.POST("/chats", boardChatHandler.Create)

		api.PATCH("/columns/:id", columnHandler.Update)
		api.DELETE("/columns/:id", columnHandler.Delete)
		api.DELETE("/board-chats/:id", boardChatHandler.Delete)
		api.GET("/tasks/:id", taskHandler.Get)
		api.PATCH("/tasks/:id", taskHandler.Update)
		api.PUT("/tasks/:id/move", taskHandler.Move)
		api.DELETE("/tasks/:id", taskHandler.Delete)
		api.POST("/tasks/:id/assign", taskHandler.Assign)
		api.DELETE("/tasks/:id/assign/:uid", taskHandler.Unassign)
		api.POST("/tasks/:id/checklists", checklistHandler.Create)
		api.GET("/tasks/:id/comments", commentHandler.List)
		api.POST("/tasks/:id/comments", commentHandler.Create)
		api.DELETE("/comments/:id", commentHandler.Delete)
		api.PATCH("/checklists/:id", checklistHandler.Update)
		api.DELETE("/checklists/:id", checklistHandler.Delete)
		api.POST("/checklists/:id/items", checklistHandler.CreateItem)
		api.PATCH("/checklist-items/:id", checklistHandler.UpdateItem)
		api.DELETE("/checklist-items/:id", checklistHandler.DeleteItem)
	}
	registerAPI("")
	registerAPI("/max-kanban")

	frontend, _ := fs.Sub(frontendFS, "frontend/dist")
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/max-kanban/api/") {
			c.JSON(404, gin.H{"error": "API route not found"})
			return
		}
		relative := path
		if path == "/max-kanban" || strings.HasPrefix(path, "/max-kanban/") {
			relative = strings.TrimPrefix(path, "/max-kanban")
		}
		if relative == "" || relative == "/" {
			relative = "/index.html"
		}
		contents, err := fs.ReadFile(frontend, strings.TrimPrefix(relative, "/"))
		if err != nil {
			contents, err = fs.ReadFile(frontend, "index.html")
		}
		if err != nil {
			c.Status(404)
			return
		}
		c.Data(200, contentType(relative), contents)
	})
	return router
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	default:
		return "text/html"
	}
}
