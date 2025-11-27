package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/C9b3rD3vi1/jobhunter-tool/database"
	"github.com/C9b3rD3vi1/jobhunter-tool/models"
	"github.com/gocolly/colly/v2"
	"gorm.io/datatypes"
)

type RealScraper struct {
	collector *colly.Collector
	db        *database.DB
	mu        sync.Mutex
	jobsFound int
}

// ScrapingConfig holds configuration for scraping behavior
type ScrapingConfig struct {
	Parallelism int
	Delay       time.Duration
	Timeout     time.Duration
	UserAgent   string
}

// DefaultConfig returns sensible default scraping configuration
func DefaultConfig() ScrapingConfig {
	return ScrapingConfig{
		Parallelism: 1,
		Delay:       4 * time.Second,
		Timeout:     30 * time.Second,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

func NewRealScraper(db *database.DB) *RealScraper {
	return NewRealScraperWithConfig(db, DefaultConfig())
}

func NewRealScraperWithConfig(db *database.DB, config ScrapingConfig) *RealScraper {
	c := colly.NewCollector(
		colly.AllowedDomains(
			"www.brightermonday.co.ke",
			"www.brightermonday.com",
			"www.fuzu.com",
			"www.safaricom.co.ke",
			"www.kcbgroup.com",
			"www.equitybankgroup.com",
		),
		colly.Async(true),
	)

	c.SetRequestTimeout(config.Timeout)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: config.Parallelism,
		Delay:       config.Delay,
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", config.UserAgent)
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Connection", "keep-alive")
		log.Printf("🌐 Visiting: %s", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("❌ Request failed: %s - Error: %v", r.Request.URL, err)
	})

	c.OnResponse(func(r *colly.Response) {
		log.Printf("✅ Success: %s (%d bytes)", r.Request.URL, len(r.Body))
	})

	return &RealScraper{
		collector: c,
		db:        db,
	}
}

func (s *RealScraper) ScrapeAllSources() error {
	log.Println("🚀 Starting job scraping from all sources...")
	
	startTime := time.Now()
	s.jobsFound = 0

	var wg sync.WaitGroup
	var errors []string
	var errorsMu sync.Mutex

	// Define scraping tasks
	tasks := []struct {
		name string
		fn   func() error
	}{
		{"BrighterMonday", s.ScrapeBrighterMonday},
		{"Company Websites", s.ScrapeCompanyPages},
		{"Fuzu", s.ScrapeFuzu},
	}

	// Execute scraping tasks concurrently
	for _, task := range tasks {
		wg.Add(1)
		go func(t struct {
			name string
			fn   func() error
		}) {
			defer wg.Done()
			log.Printf("🔄 Starting %s scraping...", t.name)
			
			if err := t.fn(); err != nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", t.name, err))
				errorsMu.Unlock()
			}
		}(task)
	}

	wg.Wait()
	s.collector.Wait()

	duration := time.Since(startTime)
	log.Printf("📊 Scraping completed in %v. Found %d jobs.", duration, s.jobsFound)

	if len(errors) > 0 {
		return fmt.Errorf("scraping completed with errors: %v", errors)
	}

	return nil
}

func (s *RealScraper) ScrapeBrighterMonday() error {
	keywords := []string{
		"cybersecurity", "security", "soc", "cloud security", "network security",
		"fortinet", "aws security", "information security", "security analyst",
	}

	var jobsFound int
	var mu sync.Mutex

	s.collector.OnHTML("div.search-result", func(e *colly.HTMLElement) {
		job := s.extractBrighterMondayJob(e)
		if job != nil {
			s.enrichAndSaveJob(job)
			mu.Lock()
			jobsFound++
			mu.Unlock()
		}
	})

	for _, keyword := range keywords {
		url := fmt.Sprintf("https://www.brightermonday.co.ke/jobs?q=%s", strings.ReplaceAll(keyword, " ", "+"))
		
		if err := s.collector.Visit(url); err != nil {
			log.Printf("⚠️ Error visiting BrighterMonday for '%s': %v", keyword, err)
			continue
		}
		
		time.Sleep(3 * time.Second) // Be respectful to the server
	}

	log.Printf("✅ BrighterMonday scraping completed. Found %d jobs.", jobsFound)
	return nil
}

func (s *RealScraper) extractBrighterMondayJob(e *colly.HTMLElement) *models.Job {
	title := strings.TrimSpace(e.ChildText("h3.search-result__job-title"))
	company := strings.TrimSpace(e.ChildText("div.search-result__job-meta > span:first-child"))
	
	if title == "" || company == "" {
		return nil
	}

	location := strings.TrimSpace(e.ChildText("div.search-result__job-meta > span:nth-child(2)"))
	description := strings.TrimSpace(e.ChildText("div.search-result__job-description"))
	url := e.ChildAttr("a.search-result__job-title", "href")

	if url != "" && !strings.HasPrefix(url, "http") {
		url = "https://www.brightermonday.co.ke" + url
	}

	return &models.Job{
		ID:          fmt.Sprintf("bm-%d", time.Now().UnixNano()),
		Title:       title,
		Company:     company,
		Location:    location,
		Description: description,
		Source:      "BrighterMonday",
		URL:         url,
		PostedDate:  time.Now().Format("2006-01-02"),
	}
}

func (s *RealScraper) ScrapeFuzu() error {
	categories := []string{"technology", "it", "security", "cyber-security"}
	var jobsFound int

	s.collector.OnHTML("div[class*='job-card'], div[data-testid*='job']", func(e *colly.HTMLElement) {
		job := s.extractFuzuJob(e)
		if job != nil {
			s.enrichAndSaveJob(job)
			jobsFound++
		}
	})

	for _, category := range categories {
		url := fmt.Sprintf("https://www.fuzu.com/kenya/%s-jobs", category)
		
		if err := s.collector.Visit(url); err != nil {
			log.Printf("⚠️ Error visiting Fuzu category '%s': %v", category, err)
			continue
		}
		
		time.Sleep(3 * time.Second)
	}

	log.Printf("✅ Fuzu scraping completed. Found %d jobs.", jobsFound)
	return nil
}

func (s *RealScraper) extractFuzuJob(e *colly.HTMLElement) *models.Job {
	title := strings.TrimSpace(e.ChildText("h3, h4, [class*='title']"))
	company := strings.TrimSpace(e.ChildText("[class*='company'], [class*='employer']"))
	
	if title == "" || company == "" {
		return nil
	}

	location := strings.TrimSpace(e.ChildText("[class*='location'], [class*='address']"))
	jobURL := e.ChildAttr("a", "href")

	if jobURL != "" && !strings.HasPrefix(jobURL, "http") {
		jobURL = "https://www.fuzu.com" + jobURL
	}

	return &models.Job{
		ID:         fmt.Sprintf("fz-%d", time.Now().UnixNano()),
		Title:      title,
		Company:    company,
		Location:   location,
		Source:     "Fuzu",
		URL:        jobURL,
		PostedDate: time.Now().Format("2006-01-02"),
	}
}

func (s *RealScraper) ScrapeCompanyPages() error {
	companies := []struct {
		name string
		url  string
	}{
		{"Safaricom", "https://www.safaricom.co.ke/careers/"},
		{"KCB Bank", "https://www.kcbgroup.com/careers/"},
		{"Equity Bank", "https://www.equitybankgroup.com/careers/"},
	}

	var jobsFound int

	s.collector.OnHTML("div.job-listing, li.job, tr.job-row, a[href*='job'], div[class*='job']", func(e *colly.HTMLElement) {
		for _, company := range companies {
			job := s.extractCompanyJob(e, company.name, company.url)
			if job != nil {
				s.enrichAndSaveJob(job)
				jobsFound++
				break // Avoid duplicate processing
			}
		}
	})

	for _, company := range companies {
		if err := s.collector.Visit(company.url); err != nil {
			log.Printf("⚠️ Error visiting %s: %v", company.name, err)
			continue
		}
		time.Sleep(4 * time.Second)
	}

	log.Printf("✅ Company websites scraping completed. Found %d jobs.", jobsFound)
	return nil
}

func (s *RealScraper) extractCompanyJob(e *colly.HTMLElement, companyName, companyURL string) *models.Job {
	title := strings.TrimSpace(e.ChildText("h3, h4, .title, .job-title"))
	if title == "" {
		title = strings.TrimSpace(e.Text)
	}

	// Filter for relevant cybersecurity roles
	if !s.isRelevantJob(title) {
		return nil
	}

	jobURL := e.ChildAttr("a", "href")
	if jobURL != "" && !strings.HasPrefix(jobURL, "http") {
		jobURL = companyURL + jobURL
	}

	return &models.Job{
		ID:         fmt.Sprintf("comp-%d", time.Now().UnixNano()),
		Title:      title,
		Company:    companyName,
		Location:   "Nairobi, Kenya",
		Source:     "Company Website",
		URL:        jobURL,
		PostedDate: time.Now().Format("2006-01-02"),
	}
}

func (s *RealScraper) isRelevantJob(title string) bool {
	if title == "" {
		return false
	}

	titleLower := strings.ToLower(title)
	relevantKeywords := []string{
		"security", "cyber", "soc", "cloud", "network",
		"analyst", "engineer", "specialist", "consultant",
		"fortinet", "aws", "azure", "siem", "incident",
	}

	for _, keyword := range relevantKeywords {
		if strings.Contains(titleLower, keyword) {
			return true
		}
	}

	return false
}

func (s *RealScraper) enrichAndSaveJob(job *models.Job) {
	// Get full description if URL is available
	if job.URL != "" {
		job.Description = s.ScrapeJobDescription(job.URL)
	}

	// Extract and set job attributes
	job.Skills = s.ConvertToJSON(s.ExtractSkills(job.Description + " " + job.Title))
	job.TechStack = s.ConvertToJSON(s.ExtractTechStack(job.Description))
	job.Score = s.CalculateScore(job)
	job.SalaryRange = s.ExtractSalary(job.Description)
	job.Experience = s.ExtractExperience(job.Description)

	// Save to database
	if err := s.db.SaveJob(job); err != nil {
		log.Printf("❌ Error saving job '%s' at '%s': %v", job.Title, job.Company, err)
	} else {
		s.mu.Lock()
		s.jobsFound++
		s.mu.Unlock()
		log.Printf("✅ Saved: %s at %s (Score: %d)", job.Title, job.Company, job.Score)
	}
}

func (s *RealScraper) ScrapeJobDescription(url string) string {
	if url == "" {
		return ""
	}

	var description string
	descCollector := colly.NewCollector()
	descCollector.SetRequestTimeout(30 * time.Second)

	// Try multiple selectors for job description
	selectors := []string{
		"div.job-description",
		"div.description",
		"article.content",
		"section.description",
		"div[class*='desc']",
		"div.job-details",
		"div.requirements",
		"div.responsibilities",
	}

	descCollector.OnHTML(strings.Join(selectors, ", "), func(e *colly.HTMLElement) {
		if description == "" {
			description = strings.TrimSpace(e.Text)
		}
	})

	// Fallback to body text if no specific description found
	descCollector.OnHTML("body", func(e *colly.HTMLElement) {
		if description == "" {
			description = strings.TrimSpace(e.Text)
		}
	})

	if err := descCollector.Visit(url); err != nil {
		log.Printf("⚠️ Error scraping job description from %s: %v", url, err)
	}

	return description
}

// SkillPattern defines a pattern for skill extraction
type SkillPattern struct {
	Pattern string
	Name    string
}

var (
	skillPatterns = []SkillPattern{
		{`aws`, "AWS"},
		{`amazon web services`, "AWS"},
		{`azure`, "Azure"},
		{`microsoft azure`, "Azure"},
		{`gcp`, "GCP"},
		{`google cloud`, "GCP"},
		{`python`, "Python"},
		{`go`, "Go"},
		{`golang`, "Go"},
		{`java`, "Java"},
		{`javascript`, "JavaScript"},
		{`node\.js`, "Node.js"},
		{`react`, "React"},
		{`docker`, "Docker"},
		{`kubernetes`, "Kubernetes"},
		{`k8s`, "Kubernetes"},
		{`terraform`, "Terraform"},
		{`ansible`, "Ansible"},
		{`fortinet`, "Fortinet"},
		{`palo alto`, "Palo Alto"},
		{`cisco`, "Cisco"},
		{`siem`, "SIEM"},
		{`splunk`, "Splunk"},
		{`qradar`, "QRadar"},
		{`arcsight`, "ArcSight"},
		{`cybersecurity`, "Cybersecurity"},
		{`cyber security`, "Cybersecurity"},
		{`cloud security`, "Cloud Security"},
		{`network security`, "Network Security"},
		{`soc`, "SOC"},
		{`security operations center`, "SOC"},
		{`incident response`, "Incident Response"},
		{`threat intelligence`, "Threat Intelligence"},
		{`vulnerability management`, "Vulnerability Management"},
		{`penetration testing`, "Penetration Testing"},
		{`pen testing`, "Penetration Testing"},
		{`firewall`, "Firewall"},
		{`vpn`, "VPN"},
		{`ids`, "IDS"},
		{`ips`, "IPS"},
		{`linux`, "Linux"},
		{`windows`, "Windows"},
		{`active directory`, "Active Directory"},
	}
)

func (s *RealScraper) ExtractSkills(text string) []string {
	if text == "" {
		return []string{}
	}

	text = strings.ToLower(text)
	skills := make([]string, 0)
	skillSet := make(map[string]bool)

	for _, pattern := range skillPatterns {
		matched, _ := regexp.MatchString(pattern.Pattern, text)
		if matched && !skillSet[pattern.Name] {
			skills = append(skills, pattern.Name)
			skillSet[pattern.Name] = true
		}
	}

	return skills
}

func (s *RealScraper) ExtractTechStack(text string) []string {
	if text == "" {
		return []string{}
	}

	text = strings.ToLower(text)
	techStack := make([]string, 0)
	techSet := make(map[string]bool)

	techPatterns := []string{
		`aws`, `ec2`, `s3`, `lambda`, `cloudformation`,
		`azure`, `azure ad`, `azure security center`,
		`docker`, `kubernetes`, `container`,
		`terraform`, `ansible`, `chef`, `puppet`,
		`jenkins`, `gitlab`, `github actions`,
		`linux`, `ubuntu`, `centos`, `redhat`,
		`windows server`, `active directory`,
		`mysql`, `postgresql`, `mongodb`, `redis`,
		`elasticsearch`, `kibana`, `logstash`,
	}

	for _, pattern := range techPatterns {
		matched, _ := regexp.MatchString(pattern, text)
		if matched {
			techName := strings.Title(strings.ReplaceAll(pattern, `\.`, " "))
			if !techSet[techName] {
				techStack = append(techStack, techName)
				techSet[techName] = true
			}
		}
	}

	return techStack
}

func (s *RealScraper) ExtractSalary(text string) string {
	if text == "" {
		return "Negotiable"
	}

	text = strings.ToLower(text)
	salaryPatterns := []string{
		`ksh\s*[\d,]+\.?\d*\s*[-–]\s*ksh\s*[\d,]+\.?\d*`,
		`ksh\s*[\d,]+\.?\d*\s*[-–]\s*[\d,]+\.?\d*\s*ksh`,
		`salary\s*:?\s*ksh\s*[\d,]+\.?\d*\s*[-–]\s*ksh\s*[\d,]+\.?\d*`,
		`ksh\s*[\d,]+\.?\d*\s*per\s*month`,
		`ksh\s*[\d,]+\.?\d*\s*pm`,
		`[\d,]+\.?\d*\s*[-–]\s*[\d,]+\.?\d*\s*ksh`,
	}

	for _, pattern := range salaryPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 0 {
			return formatSalary(matches[0])
		}
	}

	return "Negotiable"
}

func formatSalary(salary string) string {
	// Clean up and format salary string
	salary = strings.TrimSpace(salary)
	salary = strings.Title(salary)
	return salary
}

func (s *RealScraper) ExtractExperience(text string) string {
	if text == "" {
		return "Not specified"
	}

	text = strings.ToLower(text)
	experiencePatterns := map[string]string{
		`(\d+)\+?\s*years?`:                       "$1 years experience",
		`mid.*level|intermediate`:                  "Mid Level",
		`senior|lead|principal|head\s*of`:         "Senior Level",
		`junior|associate|entry.*level`:           "Junior Level",
		`intern|internship|trainee`:               "Internship",
		`executive|director|vice.*president|vp`:   "Executive Level",
	}

	for pattern, level := range experiencePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 0 {
			if strings.Contains(level, "$1") && len(matches) > 1 {
				return strings.Replace(level, "$1", matches[1], 1)
			}
			return level
		}
	}

	return "Not specified"
}

func (s *RealScraper) CalculateScore(job *models.Job) int {
	if job == nil {
		return 0
	}

	score := 0
	text := strings.ToLower(job.Description + " " + job.Title)

	// Get user skills from database
	userSkills, err := s.db.GetUserSkills()
	if err != nil {
		userSkills = []string{"AWS", "Python", "Go", "Fortinet", "SIEM", "Docker"}
	}

	// Skill matching (60 points)
	matchedSkills := 0
	for _, userSkill := range userSkills {
		if strings.Contains(text, strings.ToLower(userSkill)) {
			matchedSkills++
		}
	}

	if len(userSkills) > 0 {
		skillRatio := float64(matchedSkills) / float64(len(userSkills))
		score += int(skillRatio * 60)
	}

	// Experience level matching (20 points)
	score += s.calculateExperienceScore(text)

	// Salary indication (10 points)
	if strings.Contains(text, "ksh") || strings.Contains(text, "salary") || strings.Contains(text, "compensation") {
		score += 10
	}

	// Company reputation (10 points)
	score += s.calculateCompanyScore(job.Company)

	return min(score, 100)
}

func (s *RealScraper) calculateExperienceScore(text string) int {
	switch {
	case strings.Contains(text, "5 years") || strings.Contains(text, "senior") || strings.Contains(text, "lead"):
		return 20
	case strings.Contains(text, "3 years") || strings.Contains(text, "mid-level") || strings.Contains(text, "intermediate"):
		return 15
	case strings.Contains(text, "1 year") || strings.Contains(text, "junior") || strings.Contains(text, "entry"):
		return 10
	default:
		return 5
	}
}

func (s *RealScraper) calculateCompanyScore(company string) int {
	premiumCompanies := []string{"safaricom", "kcb", "equity", "google", "microsoft", "amazon", "oracle", "ibm"}
	companyLower := strings.ToLower(company)

	for _, premium := range premiumCompanies {
		if strings.Contains(companyLower, premium) {
			return 10
		}
	}
	return 5
}

func (s *RealScraper) ConvertToJSON(slice []string) datatypes.JSON {
	if len(slice) == 0 {
		return datatypes.JSON([]byte(`[]`))
	}

	jsonData, err := json.Marshal(slice)
	if err != nil {
		log.Printf("❌ Error marshaling JSON: %v", err)
		return datatypes.JSON([]byte(`[]`))
	}

	return datatypes.JSON(jsonData)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}