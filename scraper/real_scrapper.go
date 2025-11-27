package scraper

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "regexp"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
    "github.com/gocolly/colly/v2"
    "C9b3rD3vi1/jobhunter-tool/models"
)

type RealScraper struct {
    collector *colly.Collector
    db        *models.DB
}

func NewRealScraper(db *models.DB) *RealScraper {
    c := colly.NewCollector(
        colly.AllowedDomains("www.brightermonday.co.ke", "www.fuzu.com", "ke.linkedin.com"),
        colly.Async(true),
    )

    c.Limit(&colly.LimitRule{
        DomainGlob:  "*",
        Parallelism: 2,
        Delay:       2 * time.Second,
    })

    return &RealScraper{
        collector: c,
        db:        db,
    }
}

func (s *RealScraper) ScrapeAllSources() error {
    log.Println("Starting real job scraping from all sources...")
    
    var errors []string
    
    // Scrape BrighterMonday
    if err := s.ScrapeBrighterMonday(); err != nil {
        errors = append(errors, fmt.Sprintf("BrighterMonday: %v", err))
    }
    
    // Scrape Fuzu
    if err := s.ScrapeFuzu(); err != nil {
        errors = append(errors, fmt.Sprintf("Fuzu: %v", err))
    }
    
    // Scrape company career pages directly
    companies := []string{
        "https://www.safaricom.co.ke/careers/",
        "https://www.kcbgroup.com/careers/",
        "https://www.equitybankgroup.com/careers/",
    }
    
    for _, company := range companies {
        if err := s.ScrapeCompanyPage(company); err != nil {
            errors = append(errors, fmt.Sprintf("%s: %v", company, err))
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("scraping errors: %v", strings.Join(errors, "; "))
    }
    
    return nil
}

func (s *RealScraper) ScrapeBrighterMonday() error {
    keywords := []string{"cybersecurity", "security", "soc", "cloud security", "network security", "fortinet", "aws security"}
    
    for _, keyword := range keywords {
        url := fmt.Sprintf("https://www.brightermonday.co.ke/jobs?q=%s", strings.ReplaceAll(keyword, " ", "+"))
        
        s.collector.OnHTML("div.search-result", func(e *colly.HTMLElement) {
            title := e.ChildText("h3.search-result__job-title")
            company := e.ChildText("div.search-result__job-meta > span:first-child")
            location := e.ChildText("div.search-result__job-meta > span:nth-child(2)")
            description := e.ChildText("div.search-result__job-description")
            url := e.ChildAttr("a.search-result__job-title", "href")
            
            if url != "" {
                url = "https://www.brightermonday.co.ke" + url
            }
            
            if title != "" && company != "" {
                job := &models.Job{
                    Title:       strings.TrimSpace(title),
                    Company:     strings.TrimSpace(company),
                    Location:    strings.TrimSpace(location),
                    Description: strings.TrimSpace(description),
                    Source:      "BrighterMonday",
                    URL:         url,
                    PostedDate:  time.Now().Format("2006-01-02"),
                }
                
                // Extract skills and calculate score
                job.Skills = s.ExtractSkills(job.Description)
                job.Score = s.CalculateScore(job)
                
                // Save to database
                if err := s.db.SaveJob(job); err != nil {
                    log.Printf("Error saving job: %v", err)
                } else {
                    log.Printf("Saved job: %s at %s", job.Title, job.Company)
                }
            }
        })
        
        err := s.collector.Visit(url)
        if err != nil {
            log.Printf("Error visiting BrighterMonday: %v", err)
        }
    }
    
    s.collector.Wait()
    return nil
}

func (s *RealScraper) ScrapeFuzu() error {
    // Fuzu scraping implementation
    categories := []string{"technology", "IT", "security"}
    
    for _, category := range categories {
        url := fmt.Sprintf("https://www.fuzu.com/kenya/%s-jobs", category)
        
        s.collector.OnHTML("div.job-card", func(e *colly.HTMLElement) {
            title := e.ChildText("h3.job-title")
            company := e.ChildText("div.company-name")
            location := e.ChildText("div.job-location")
            description := e.ChildText("div.job-description")
            url := e.ChildAttr("a", "href")
            
            if url != "" && !strings.HasPrefix(url, "http") {
                url = "https://www.fuzu.com" + url
            }
            
            if title != "" {
                job := &models.Job{
                    Title:       strings.TrimSpace(title),
                    Company:     strings.TrimSpace(company),
                    Location:    strings.TrimSpace(location),
                    Description: strings.TrimSpace(description),
                    Source:      "Fuzu",
                    URL:         url,
                    PostedDate:  time.Now().Format("2006-01-02"),
                }
                
                job.Skills = s.ExtractSkills(job.Description)
                job.Score = s.CalculateScore(job)
                
                if err := s.db.SaveJob(job); err != nil {
                    log.Printf("Error saving Fuzu job: %v", err)
                }
            }
        })
        
        err := s.collector.Visit(url)
        if err != nil {
            log.Printf("Error visiting Fuzu: %v", err)
        }
    }
    
    s.collector.Wait()
    return nil
}

func (s *RealScraper) ScrapeCompanyPage(url string) error {
    s.collector.OnHTML("div.job-listing, li.job, tr.job-row", func(e *colly.HTMLElement) {
        title := e.ChildText("h3, .job-title, .title")
        company := "Safaricom" // Extract from URL or page
        
        if strings.Contains(url, "kcb") {
            company = "KCB Bank"
        } else if strings.Contains(url, "equity") {
            company = "Equity Bank"
        }
        
        if title != "" && !strings.Contains(strings.ToLower(title), "intern") {
            job := &models.Job{
                Title:      strings.TrimSpace(title),
                Company:    company,
                Location:   "Nairobi", // Default location
                Source:     "Company Website",
                URL:        url,
                PostedDate: time.Now().Format("2006-01-02"),
            }
            
            // Get full description by visiting job detail page
            detailURL := e.ChildAttr("a", "href")
            if detailURL != "" {
                if !strings.HasPrefix(detailURL, "http") {
                    detailURL = url + detailURL
                }
                job.URL = detailURL
                job.Description = s.ScrapeJobDescription(detailURL)
            }
            
            job.Skills = s.ExtractSkills(job.Description)
            job.Score = s.CalculateScore(job)
            
            if err := s.db.SaveJob(job); err != nil {
                log.Printf("Error saving company job: %v", err)
            }
        }
    })
    
    return s.collector.Visit(url)
}

func (s *RealScraper) ScrapeJobDescription(url string) string {
    var description string
    
    descCollector := colly.NewCollector()
    descCollector.OnHTML("div.job-description, div.description, article.content", func(e *colly.HTMLElement) {
        description = e.Text
    })
    
    descCollector.Visit(url)
    return strings.TrimSpace(description)
}

func (s *RealScraper) ExtractSkills(description string) []string {
    skills := []string{}
    description = strings.ToLower(description)
    
    // Technical skills patterns
    skillPatterns := []string{
        `aws`, `azure`, `gcp`, `cloud`, `python`, `go`, `golang`, `java`, `javascript`, 
        `react`, `node\.js`, `docker`, `kubernetes`, `terraform`, `ansible`, 
        `fortinet`, `palo alto`, `cisco`, `siem`, `splunk`, `qradar`, 
        `cybersecurity`, `security`, `soc`, `incident response`, `threat intelligence`,
        `network security`, `firewall`, `vpn`, `ids/ips`, `endpoint security`,
        `mysql`, `postgresql`, `mongodb`, `redis`, `linux`, `windows server`,
        `ci/cd`, `jenkins`, `gitlab`, `github actions`, `devops`,
    }
    
    for _, pattern := range skillPatterns {
        matched, _ := regexp.MatchString(pattern, description)
        if matched {
            skills = append(skills, strings.ToUpper(pattern))
        }
    }
    
    return skills
}

func (s *RealScraper) CalculateScore(job *models.Job) int {
    score := 0
    desc := strings.ToLower(job.Description + " " + job.Title)
    
    // Get user skills from database
    userSkills := s.GetUserSkills()
    
    // Skill matching (60 points)
    matchedSkills := 0
    for _, skill := range userSkills {
        if strings.Contains(desc, strings.ToLower(skill)) {
            matchedSkills++
        }
    }
    
    if len(userSkills) > 0 {
        skillRatio := float64(matchedSkills) / float64(len(userSkills))
        score += int(skillRatio * 60)
    }
    
    // Salary indication (20 points)
    salaryIndicators := []string{"ksh", "salary", "competitive", "attractive"}
    for _, indicator := range salaryIndicators {
        if strings.Contains(desc, indicator) {
            score += 10
            break
        }
    }
    
    // Experience level matching (20 points)
    experiencePatterns := map[string]int{
        `(\d+)\+?\s*years`: 15,
        `mid.*level`:       15,
        `senior`:           20,
        `junior`:           10,
        `entry.*level`:      5,
    }
    
    for pattern, points := range experiencePatterns {
        matched, _ := regexp.MatchString(pattern, desc)
        if matched {
            score += points
            break
        }
    }
    
    return min(score, 100)
}

func (s *RealScraper) GetUserSkills() []string {
    var skills []string
    rows, err := s.db.Query("SELECT skill FROM user_skills")
    if err != nil {
        return []string{"AWS", "Python", "Go", "Fortinet", "SIEM", "Docker"}
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