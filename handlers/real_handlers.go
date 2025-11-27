package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/C9b3rD3vi1/jobhunter-tool/ai"
	"github.com/C9b3rD3vi1/jobhunter-tool/database"
	"github.com/C9b3rD3vi1/jobhunter-tool/models"
	"github.com/C9b3rD3vi1/jobhunter-tool/scraper"
	"github.com/gofiber/fiber/v2"
)

func IndexHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*database.DB)
    
    totalJobs, highScoreJobs, err := db.GetJobStats()
    if err != nil {
        totalJobs, highScoreJobs = 0, 0
    }
    
    var totalApplications int
    db.QueryRow("SELECT COUNT(*) FROM applications").Scan(&totalApplications)
    
    recentJobs, _ := db.GetJobs(5, 0)
    
    return c.Render("index", fiber.Map{
        "Title":              "JobHunter AI - Real Job Search Engine",
        "TotalJobs":          totalJobs,
        "HighScoreJobs":      highScoreJobs,
        "TotalApplications":  totalApplications,
        "RecentJobs":         recentJobs,
    })
}

func JobsHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*database.DB)
    
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit := 20
    offset := (page - 1) * limit
    
    jobs, err := db.GetJobs(limit, offset)
    if err != nil {
        return c.Status(500).SendString("Error fetching jobs: " + err.Error())
    }
    
    // Get filters
    scoreFilter := c.Query("score")
    skillFilter := c.Query("skill")
    companyFilter := c.Query("company")
    
    // Apply filters
    filteredJobs := filterJobs(jobs, scoreFilter, skillFilter, companyFilter)
    
    return c.Render("jobs", fiber.Map{
        "Jobs":         filteredJobs,
        "CurrentPage":  page,
        "HasNext":      len(jobs) == limit,
        "ScoreFilter":  scoreFilter,
        "SkillFilter":  skillFilter,
        "CompanyFilter": companyFilter,
    })
}

func JobDetailHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*database.DB)
    jobID := c.Params("id")
    
    job, err := db.GetJobByID(jobID)
    if err != nil {
        return c.Status(404).SendString("Job not found")
    }
    
    return c.Render("job-detail", fiber.Map{
        "Job": job,
    })
}

func ScrapeJobsHandler(c *fiber.Ctx) error {
    scraper := c.Locals("scraper").(*scraper.RealScraper)
    
    go func() {
        if err := scraper.ScrapeAllSources(); err != nil {
            fmt.Printf("Scraping error: %v\n", err)
        }
    }()
    
    return c.JSON(fiber.Map{
        "status":  "success", 
        "message": "Scraping started in background. Jobs will appear shortly.",
    })
}

func AnalyzeSkillsHandler(c *fiber.Ctx) error {
    ai := c.Locals("ai").(*ai.AIGenerator)
    db := c.Locals("db").(*models.DB)
    
    var request struct {
        JobDescription string   `json:"job_description"`
        UserSkills     []string `json:"user_skills"`
    }
    
    if err := c.BodyParser(&request); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }
    
    if len(request.UserSkills) == 0 {
        // Get skills from database
        userSkills, err := db.GetUserSkills()
        if err != nil {
            userSkills = []string{"AWS", "Python", "Go", "Fortinet", "SIEM", "Docker"}
        }
        request.UserSkills = userSkills
    }
    
    analysis := ai.GenerateSkillsAnalysis(request.JobDescription, request.UserSkills)
    
    return c.JSON(analysis)
}

func GenerateCoverLetterHandler(c *fiber.Ctx) error {
    ai := c.Locals("ai").(*ai.AIGenerator)
    
    var request struct {
        JobTitle       string `json:"job_title"`
        Company        string `json:"company"`
        JobDescription string `json:"job_description"`
        UserProfile    string `json:"user_profile"`
    }
    
    if err := c.BodyParser(&request); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }
    
    // Default user profile
    if request.UserProfile == "" {
        request.UserProfile = "Cybersecurity professional with experience in Fortinet, AWS security, SIEM, and cloud security. Strong background in SOC operations, incident response, and vulnerability management."
    }
    
    coverLetter, err := ai.GenerateCoverLetter(
        request.JobTitle,
        request.Company, 
        request.JobDescription,
        request.UserProfile,
    )
    
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to generate cover letter"})
    }
    
    return c.JSON(fiber.Map{"cover_letter": coverLetter})
}

func AddApplicationHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*models.DB)
    
    var app models.Application
    if err := c.BodyParser(&app); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid application data"})
    }
    
    if err := db.SaveApplication(&app); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to save application"})
    }
    
    return c.JSON(fiber.Map{"status": "success", "id": app.ID})
}

func TrackerHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*models.DB)
    
    applications, err := db.GetApplications()
    if err != nil {
        return c.Status(500).SendString("Error fetching applications")
    }
    
    return c.Render("tracker", fiber.Map{
        "Applications": applications,
    })
}

func ApplyHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*models.DB)
    jobID := c.Params("id")
    
    job, err := db.GetJobByID(jobID)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
    }
    
    // Create application
    app := models.Application{
        JobID:       job.ID,
        Company:     job.Company,
        Role:        job.Title,
        AppliedDate: c.FormValue("applied_date"),
        Status:      "Applied",
        Notes:       c.FormValue("notes"),
    }
    
    if app.AppliedDate == "" {
        app.AppliedDate = time.Now().Format("2006-01-02")
    }
    
    if err := db.SaveApplication(&app); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to save application"})
    }
    
    return c.JSON(fiber.Map{
        "status": "success",
        "message": fmt.Sprintf("Application to %s for %s tracked successfully", job.Company, job.Title),
    })
}

func CompanyHandler(c *fiber.Ctx) error {
    companyName := c.Params("name")
    
    // Get company-specific jobs
    db := c.Locals("db").(*models.DB)
    jobs, err := db.GetJobs(50, 0) // Get more jobs to filter
    if err != nil {
        return c.Status(500).SendString("Error fetching jobs")
    }
    
    var companyJobs []models.Job
    for _, job := range jobs {
        if strings.Contains(strings.ToLower(job.Company), strings.ToLower(companyName)) {
            companyJobs = append(companyJobs, job)
        }
    }
    
    return c.Render("company", fiber.Map{
        "CompanyName": companyName,
        "Jobs":       companyJobs,
    })
}

// Helper functions
func filterJobs(jobs []models.Job, scoreFilter, skillFilter, companyFilter string) []models.Job {
    var filtered []models.Job
    
    for _, job := range jobs {
        // Score filter
        if scoreFilter != "" {
            minScore, _ := strconv.Atoi(scoreFilter)
            if job.Score < minScore {
                continue
            }
        }
        
        // Skill filter
        if skillFilter != "" {
            hasSkill := false
            for _, skill := range job.Skills {
                if strings.Contains(strings.ToLower(skill), strings.ToLower(skillFilter)) {
                    hasSkill = true
                    break
                }
            }
            if !hasSkill {
                continue
            }
        }
        
        // Company filter
        if companyFilter != "" && !strings.Contains(strings.ToLower(job.Company), strings.ToLower(companyFilter)) {
            continue
        }
        
        filtered = append(filtered, job)
    }
    
    return filtered
}