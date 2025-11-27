package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"github.com/C9b3rD3vi1/jobhunter-tool/database"
	"github.com/C9b3rD3vi1/jobhunter-tool/handlers"
	"github.com/C9b3rD3vi1/jobhunter-tool/scraper"
)


func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using default values")
    }

    // Initialize database with GORM
    db, err := models.InitDB()
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer db.Close()

    // Initialize scraper
    jobScraper := scraper.NewRealScraper(db)

    // Initialize AI
    aiGenerator := handlers.NewAIGenerator(os.Getenv("OPENAI_API_KEY"))

    // Initialize template engine
    engine := html.New("./templates", ".html")
    
    app := fiber.New(fiber.Config{
        Views: engine,
    })
    
    // Middleware
    app.Use(logger.New())
    app.Static("/static", "./static")

    // Inject dependencies
    app.Use(func(c *fiber.Ctx) error {
        c.Locals("db", db)
        c.Locals("scraper", jobScraper)
        c.Locals("ai", aiGenerator)
        return c.Next()
    })

    // Routes
    setupRoutes(app)

    // Start background scraping cron job
    startBackgroundScraping(jobScraper)

    log.Println("🚀 JobHunter AI started on http://localhost:3000")
    log.Fatal(app.Listen(":3000"))
}

func setupRoutes(app *fiber.App) {
    app.Get("/", handlers.IndexHandler)
    app.Get("/jobs", handlers.JobsHandler)
    app.Get("/jobs/scrape", handlers.ScrapeJobsHandler)
    app.Get("/jobs/:id", handlers.JobDetailHandler)
    app.Post("/jobs/:id/apply", handlers.ApplyHandler)
    app.Get("/tracker", handlers.TrackerHandler)
    app.Post("/tracker/add", handlers.AddApplicationHandler)
    app.Get("/analyzer", handlers.AnalyzerHandler)
    app.Post("/analyze-skills", handlers.AnalyzeSkillsHandler)
    app.Get("/company/:name", handlers.CompanyHandler)
    app.Post("/cover-letter", handlers.GenerateCoverLetterHandler)
    
    // API routes
    app.Get("/api/jobs", handlers.APIJobsHandler)
    app.Get("/api/stats", handlers.APIStatsHandler)
}

func startBackgroundScraping(scraper *scraper.RealScraper) {
    c := cron.New()
    
    // Scrape every 6 hours
    c.AddFunc("0 */6 * * *", func() {
        log.Println("🔄 Starting scheduled job scraping...")
        if err := scraper.ScrapeAllSources(); err != nil {
            log.Printf("❌ Scheduled scraping failed: %v", err)
        } else {
            log.Println("✅ Scheduled scraping completed successfully")
        }
    })
    
    // Scrape immediately on startup
    go func() {
        time.Sleep(10 * time.Second) // Wait for server to start
        log.Println("🔄 Starting initial job scraping...")
        if err := scraper.ScrapeAllSources(); err != nil {
            log.Printf("❌ Initial scraping failed: %v", err)
        }
    }()
    
    c.Start()
}