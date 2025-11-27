package handlers

import (
    "encoding/json"
    "fmt"
    "strconv"
    "strings"

    "github.com/gofiber/fiber/v2"
    "jobhunter-tool/ai"
    "jobhunter-tool/models"
    "jobhunter-tool/scraper"
)

func IndexHandler(c *fiber.Ctx) error {
    db := c.Locals("db").(*models.DB)
    
    // Get real stats
    var totalJobs, highScoreJobs, totalApplications int
    
    db.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&totalJobs)
    db.QueryRow("SELECT COUNT(*) FROM jobs WHERE score >= 80").Scan(&highScoreJobs)
    db.QueryRow("SELECT COUNT(*) FROM applications").Scan(&totalApplications)
    
    // Get recent high-score jobs
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
    db := c.Locals("db").(*models.DB)
    
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit := 20
    offset := (page - 1) * limit
    
    jobs, err := db.GetJobs(limit, offset)
    if err != nil {
        return c.Status(500).SendString("Error fetching jobs")
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

func ScrapeJobsHandler(c *fiber.Ctx) error {
    scraper := c.Locals("scraper").(*scraper.RealScraper)
    
    go func() {
        if err := scraper.ScrapeAllSources(); err != nil {
            fmt.Printf("Scraping error: %v\n", err)
        }
    }()
    
    return c.JSON(fiber.Map{
        "status":  "success",
        "message": "Scraping started in background",
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
        request.UserSkills = getSkillsFromDB(db)
    }
    
    analysis, matching, missing, score := ai.GenerateSkillsAnalysis(
        request.JobDescription, 
        request.UserSkills,
    )
    
    return c.JSON(fiber.Map{
        "analysis":        analysis,
        "matching_skills": matching,
        "missing_skills":  missing,
        "fit_score":       score,
    })
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

func getSkillsFromDB(db *models.DB) []string {
    var skills []string
    rows, err := db.Query("SELECT skill FROM user_skills")
    if err != nil {
        return []string{"AWS", "Python", "Go", "Fortinet", "SIEM"}
    }
    defer rows.Close()
    
    for rows.Next() {
        var skill string
        if err := rows.Scan(&skill); err == nil {
            skills = append(skills, skill)
        }
    }
    return skills
}