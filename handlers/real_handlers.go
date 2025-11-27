package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/C9b3rD3vi1/jobhunter-tool/ai"
	"github.com/C9b3rD3vi1/jobhunter-tool/database"
	"github.com/C9b3rD3vi1/jobhunter-tool/models"
	"github.com/C9b3rD3vi1/jobhunter-tool/scraper"
	"github.com/gofiber/fiber/v2"
)

// Response types for consistent API responses
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Request types for better type safety
type AnalyzeSkillsRequest struct {
	JobDescription string   `json:"job_description" validate:"required"`
	UserSkills     []string `json:"user_skills"`
}

type CoverLetterRequest struct {
	JobTitle       string `json:"job_title" validate:"required"`
	Company        string `json:"company" validate:"required"`
	JobDescription string `json:"job_description" validate:"required"`
	UserProfile    string `json:"user_profile"`
}

type AddSkillRequest struct {
	Skill string `json:"skill" validate:"required"`
}

type AddApplicationRequest struct {
	JobID         string `json:"job_id"`
	Company       string `json:"company" validate:"required"`
	Role          string `json:"role" validate:"required"`
	AppliedDate   string `json:"applied_date"`
	Status        string `json:"status"`
	HiringManager string `json:"hiring_manager"`
	Notes         string `json:"notes"`
}

// Success responses
func success(message string, data ...interface{}) Response {
	resp := Response{
		Status:  "success",
		Message: message,
	}
	if len(data) > 0 {
		resp.Data = data[0]
	}
	return resp
}

// Error responses
func errorResponse(message string) Response {
	return Response{
		Status: "error",
		Error:  message,
	}
}

// HandlerContext provides common dependencies for handlers
type HandlerContext struct {
	DB      *database.DB
	Scraper *scraper.RealScraper
	AI      *ai.AIGenerator
}

// getHandlerContext extracts common dependencies from Fiber context
func getHandlerContext(c *fiber.Ctx) *HandlerContext {
	return &HandlerContext{
		DB:      c.Locals("db").(*database.DB),
		Scraper: c.Locals("scraper").(*scraper.RealScraper),
		AI:      c.Locals("ai").(*ai.AIGenerator),
	}
}

// IndexHandler displays the dashboard
func IndexHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	stats, err := getDashboardStats(ctx.DB)
	if err != nil {
		log.Printf("Error getting dashboard stats: %v", err)
		stats = getEmptyStats()
	}

	recentJobs, err := ctx.DB.GetJobs(5, 0)
	if err != nil {
		log.Printf("Error getting recent jobs: %v", err)
		recentJobs = []models.Job{}
	}

	return c.Render("index", fiber.Map{
		"Page":              "index",
		"Title":             "JobHunter AI - Dashboard",
		"TotalJobs":         stats.TotalJobs,
		"HighScoreJobs":     stats.HighScoreJobs,
		"TotalApplications": stats.TotalApplications,
		"ResponseRate":      stats.ResponseRate,
		"RecentJobs":        recentJobs,
	})
}

// JobsHandler displays the job board with filtering
func JobsHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit := 20
	offset := (page - 1) * limit

	jobs, err := ctx.DB.GetJobs(limit, offset)
	if err != nil {
		log.Printf("Error fetching jobs: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to fetch jobs"))
	}

	filters := parseJobFilters(c)
	filteredJobs := applyJobFilters(jobs, filters)

	return c.Render("jobs", fiber.Map{
		"Page":         "jobs",
		"Title":        "Job Board",
		"Jobs":         filteredJobs,
		"CurrentPage":  page,
		"HasNext":      len(jobs) == limit,
		"ScoreFilter":  filters.Score,
		"SkillFilter":  filters.Skill,
		"CompanyFilter": filters.Company,
		"LocationFilter": filters.Location,
	})
}

// APIJobsHandler returns jobs as JSON for API consumption
func APIJobsHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := (page - 1) * limit

	jobs, err := ctx.DB.GetJobs(limit, offset)
	if err != nil {
		log.Printf("Error fetching jobs for API: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to fetch jobs"))
	}

	return c.JSON(success("Jobs retrieved successfully", fiber.Map{
		"jobs": jobs,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": len(jobs),
		},
	}))
}

// APIStatsHandler returns system statistics as JSON
func APIStatsHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	stats, err := getDashboardStats(ctx.DB)
	if err != nil {
		log.Printf("Error getting stats for API: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to fetch statistics"))
	}

	return c.JSON(success("Statistics retrieved successfully", stats))
}

// JobDetailHandler displays details for a specific job
func JobDetailHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(400).JSON(errorResponse("Job ID is required"))
	}

	job, err := ctx.DB.GetJobByID(jobID)
	if err != nil {
		return c.Status(404).JSON(errorResponse("Job not found"))
	}

	return c.Render("job-detail", fiber.Map{
		"Page":  "jobs",
		"Title": fmt.Sprintf("%s - %s", job.Title, job.Company),
		"Job":   job,
	})
}

// ScrapeJobsHandler initiates job scraping
func ScrapeJobsHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	go func() {
		log.Println("🔄 Starting background job scraping...")
		if err := ctx.Scraper.ScrapeAllSources(); err != nil {
			log.Printf("❌ Background scraping failed: %v", err)
		} else {
			log.Println("✅ Background scraping completed")
		}
	}()

	return c.JSON(success("Scraping started in background. Jobs will appear shortly."))
}

// AnalyzeSkillsHandler analyzes skills gap for a job description
func AnalyzeSkillsHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	var req AnalyzeSkillsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(errorResponse("Invalid request format"))
	}

	if req.JobDescription == "" {
		return c.Status(400).JSON(errorResponse("Job description is required"))
	}

	// Get user skills if not provided
	if len(req.UserSkills) == 0 {
		userSkills, err := ctx.DB.GetUserSkills()
		if err != nil {
			log.Printf("Error getting user skills: %v", err)
			userSkills = getDefaultSkills()
		}
		req.UserSkills = userSkills
	}

	analysis := ctx.AI.GenerateSkillsAnalysis(req.JobDescription, req.UserSkills)

	return c.JSON(success("Skills analysis completed", analysis))
}

// GenerateCoverLetterHandler generates a cover letter for a job
func GenerateCoverLetterHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	var req CoverLetterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(errorResponse("Invalid request format"))
	}

	if req.JobTitle == "" || req.Company == "" || req.JobDescription == "" {
		return c.Status(400).JSON(errorResponse("Job title, company, and description are required"))
	}

	// Set default user profile if not provided
	if req.UserProfile == "" {
		req.UserProfile = getDefaultUserProfile()
	}

	coverLetter, err := ctx.AI.GenerateCoverLetter(
		req.JobTitle,
		req.Company,
		req.JobDescription,
		req.UserProfile,
	)

	if err != nil {
		log.Printf("Error generating cover letter: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to generate cover letter"))
	}

	return c.JSON(success("Cover letter generated", fiber.Map{
		"cover_letter": coverLetter,
	}))
}

// ApplyHandler tracks a job application from a job listing
func ApplyHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(400).JSON(errorResponse("Job ID is required"))
	}

	job, err := ctx.DB.GetJobByID(jobID)
	if err != nil {
		return c.Status(404).JSON(errorResponse("Job not found"))
	}

	appliedDate := c.FormValue("applied_date")
	if appliedDate == "" {
		appliedDate = time.Now().Format("2006-01-02")
	}

	application := models.Application{
		JobID:       job.ID,
		Company:     job.Company,
		Role:        job.Title,
		AppliedDate: appliedDate,
		Status:      "Applied",
		Notes:       c.FormValue("notes"),
	}

	if err := ctx.DB.SaveApplication(&application); err != nil {
		log.Printf("Error saving application: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to save application"))
	}

	return c.JSON(success(
		fmt.Sprintf("Application to %s for %s tracked successfully", job.Company, job.Title),
		fiber.Map{"application_id": application.ID},
	))
}

// AddApplicationHandler adds a new application manually
func AddApplicationHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	var req AddApplicationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(errorResponse("Invalid request format"))
	}

	if req.Company == "" || req.Role == "" {
		return c.Status(400).JSON(errorResponse("Company and role are required"))
	}

	if req.AppliedDate == "" {
		req.AppliedDate = time.Now().Format("2006-01-02")
	}

	if req.Status == "" {
		req.Status = "Applied"
	}

	application := models.Application{
		JobID:         req.JobID,
		Company:       req.Company,
		Role:          req.Role,
		AppliedDate:   req.AppliedDate,
		Status:        req.Status,
		HiringManager: req.HiringManager,
		Notes:         req.Notes,
	}

	if err := ctx.DB.SaveApplication(&application); err != nil {
		log.Printf("Error saving application: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to save application"))
	}

	return c.JSON(success(
		"Application added successfully",
		fiber.Map{"application_id": application.ID},
	))
}

// TrackerHandler displays the application tracker
func TrackerHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	applications, err := ctx.DB.GetApplications()
	if err != nil {
		log.Printf("Error fetching applications: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to fetch applications"))
	}

	stats := calculateApplicationStats(applications)

	return c.Render("tracker", fiber.Map{
		"Page":          "tracker",
		"Title":         "Application Tracker",
		"Applications":  applications,
		"Stats":         stats,
	})
}

// AnalyzerHandler displays the skills analyzer page
func AnalyzerHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	userSkills, err := ctx.DB.GetUserSkills()
	if err != nil {
		log.Printf("Error getting user skills: %v", err)
		userSkills = getDefaultSkills()
	}

	skillsList := strings.Join(userSkills, ", ")

	return c.Render("analyzer", fiber.Map{
		"Page":       "analyzer",
		"Title":      "Skills Analyzer",
		"UserSkills": skillsList,
		"SkillsList": userSkills,
	})
}

// AddSkillHandler adds a new skill to user's profile
func AddSkillHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	var req AddSkillRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(errorResponse("Invalid request format"))
	}

	if req.Skill == "" {
		return c.Status(400).JSON(errorResponse("Skill is required"))
	}

	if err := ctx.DB.AddUserSkill(strings.TrimSpace(req.Skill)); err != nil {
		log.Printf("Error adding skill: %v", err)
		return c.Status(500).JSON(errorResponse("Failed to add skill"))
	}

	return c.JSON(success("Skill added successfully"))
}

// CompanyHandler displays jobs for a specific company
func CompanyHandler(c *fiber.Ctx) error {
	ctx := getHandlerContext(c)

	companyName := c.Params("name")
	if companyName == "" {
		return c.Status(400).JSON(errorResponse("Company name is required"))
	}

	// Use database query for company-specific jobs instead of filtering in memory
	companyJobs, err := ctx.DB.GetJobsByCompany(companyName)
	if err != nil {
		log.Printf("Error fetching company jobs: %v", err)
		// Fallback to client-side filtering
		allJobs, err := ctx.DB.GetJobs(100, 0)
		if err != nil {
			companyJobs = []models.Job{}
		} else {
			companyJobs = filterJobsByCompany(allJobs, companyName)
		}
	}

	return c.Render("company", fiber.Map{
		"Page":        "jobs",
		"Title":       fmt.Sprintf("Jobs at %s", companyName),
		"CompanyName": companyName,
		"Jobs":        companyJobs,
	})
}

// Helper types and functions

type DashboardStats struct {
	TotalJobs         int    `json:"total_jobs"`
	HighScoreJobs     int    `json:"high_score_jobs"`
	TotalApplications int    `json:"total_applications"`
	ResponseRate      string `json:"response_rate"`
}

type JobFilters struct {
	Score    string
	Skill    string
	Company  string
	Location string
}

func getDashboardStats(db *database.DB) (DashboardStats, error) {
	totalJobs, highScoreJobs, err := db.GetJobStats()
	if err != nil {
		return DashboardStats{}, err
	}

	totalApplications, err := db.GetApplicationStats()
	if err != nil {
		return DashboardStats{}, err
	}

	return DashboardStats{
		TotalJobs:         totalJobs,
		HighScoreJobs:     highScoreJobs,
		TotalApplications: totalApplications,
		ResponseRate:      "25", // This could be calculated from actual data
	}, nil
}

func getEmptyStats() DashboardStats {
	return DashboardStats{
		TotalJobs:         0,
		HighScoreJobs:     0,
		TotalApplications: 0,
		ResponseRate:      "0",
	}
}

func parseJobFilters(c *fiber.Ctx) JobFilters {
	return JobFilters{
		Score:    c.Query("score", ""),
		Skill:    c.Query("skill", ""),
		Company:  c.Query("company", ""),
		Location: c.Query("location", ""),
	}
}

func applyJobFilters(jobs []models.Job, filters JobFilters) []models.Job {
	if filters.Score == "" && filters.Skill == "" && filters.Company == "" && filters.Location == "" {
		return jobs
	}

	var filtered []models.Job
	for _, job := range jobs {
		if !matchesFilters(job, filters) {
			continue
		}
		filtered = append(filtered, job)
	}
	return filtered
}

func matchesFilters(job models.Job, filters JobFilters) bool {
	// Score filter
	if filters.Score != "" {
		minScore, _ := strconv.Atoi(filters.Score)
		if job.Score < minScore {
			return false
		}
	}

	// Skill filter
	if filters.Skill != "" {
		hasSkill := false
		for _, skill := range job.Skills {
			if strings.Contains(strings.ToLower(string(skill)), strings.ToLower(filters.Skill)) {
				hasSkill = true
				break
			}
		}
		if !hasSkill {
			return false
		}
	}

	// Company filter
	if filters.Company != "" && !strings.Contains(strings.ToLower(job.Company), strings.ToLower(filters.Company)) {
		return false
	}

	// Location filter
	if filters.Location != "" && !strings.Contains(strings.ToLower(job.Location), strings.ToLower(filters.Location)) {
		return false
	}

	return true
}

func filterJobsByCompany(jobs []models.Job, companyName string) []models.Job {
	var filtered []models.Job
	for _, job := range jobs {
		if strings.Contains(strings.ToLower(job.Company), strings.ToLower(companyName)) {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func calculateApplicationStats(applications []models.Application) map[string]int {
	stats := map[string]int{
		"Applied":      0,
		"Interviewing": 0,
		"Offer":        0,
		"Rejected":     0,
	}

	for _, app := range applications {
		stats[app.Status]++
	}

	return stats
}

func getDefaultSkills() []string {
	return []string{"AWS", "Python", "Go", "Fortinet", "SIEM", "Docker", "Kubernetes", "Cybersecurity"}
}

func getDefaultUserProfile() string {
	return "Cybersecurity professional with experience in Fortinet, AWS security, SIEM, and cloud security. " +
		"Strong background in SOC operations, incident response, and vulnerability management. " +
		"Proficient in Python, Go, and various security frameworks."
}

// Utility function to parse JSON skills from database
func ParseSkillsFromJSON(skillsData []byte) []string {
	var skills []string
	if err := json.Unmarshal(skillsData, &skills); err != nil {
		return []string{}
	}
	return skills
}

// Utility function to parse tech stack from database
func ParseTechStackFromJSON(techStackData []byte) []string {
	var techStack []string
	if err := json.Unmarshal(techStackData, &techStack); err != nil {
		return []string{}
	}
	return techStack
}