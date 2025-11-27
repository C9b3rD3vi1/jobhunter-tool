package scraper

import (
    "fmt"
    "log"
    "regexp"
    "strings"
    "time"

    "github.com/gocolly/colly/v2"
    "github.com/C9b3rD3vi1/jobhunter-tool/models"
    "github.com/C9b3rD3vi1/jobhunter-tool/database"
)

type RealScraper struct {
    collector *colly.Collector
    db        *database.DB
}

func NewRealScraper(db *database.DB) *RealScraper {
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

    c.Limit(&colly.LimitRule{
        DomainGlob:  "*",
        Parallelism: 2,
        Delay:       3 * time.Second,
    })

    c.OnRequest(func(r *colly.Request) {
        r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
        log.Printf("Visiting %s\n", r.URL)
    })

    return &RealScraper{
        collector: c,
        db:        db,
    }
}

func (s *RealScraper) ScrapeAllSources() error {
    log.Println("Starting real job scraping from all sources...")
    
    // Scrape multiple sources concurrently
    go s.ScrapeBrighterMonday()
    go s.ScrapeFuzu()
    go s.ScrapeCompanyPages()
    
    s.collector.Wait()
    log.Println("Scraping completed")
    return nil
}

func (s *RealScraper) ScrapeBrighterMonday() {
    keywords := []string{
        "cybersecurity", "security", "soc", "cloud security", "network security", 
        "fortinet", "aws security", "information security", "security analyst",
    }
    
    for _, keyword := range keywords {
        url := fmt.Sprintf("https://www.brightermonday.co.ke/jobs?q=%s", strings.ReplaceAll(keyword, " ", "+"))
        
        s.collector.OnHTML("div.search-result", func(e *colly.HTMLElement) {
            title := e.ChildText("h3.search-result__job-title")
            company := e.ChildText("div.search-result__job-meta > span:first-child")
            location := e.ChildText("div.search-result__job-meta > span:nth-child(2)")
            description := e.ChildText("div.search-result__job-description")
            url := e.ChildAttr("a.search-result__job-title", "href")
            
            if url != "" && !strings.HasPrefix(url, "http") {
                url = "https://www.brightermonday.co.ke" + url
            }
            
            if title != "" && company != "" {
                job := &models.Job{
                    ID:          fmt.Sprintf("bm-%d", time.Now().UnixNano()),
                    Title:       strings.TrimSpace(title),
                    Company:     strings.TrimSpace(company),
                    Location:    strings.TrimSpace(location),
                    Description: strings.TrimSpace(description),
                    Source:      "BrighterMonday",
                    URL:         url,
                    PostedDate:  time.Now().Format("2006-01-02"),
                }
                
                // Get full description and extract details
                if url != "" {
                    job.Description = s.ScrapeJobDescription(url)
                }
                
                job.Skills = s.ExtractSkills(job.Description + " " + job.Title)
                job.TechStack = s.ExtractTechStack(job.Description)
                job.Score = s.CalculateScore(job)
                job.SalaryRange = s.ExtractSalary(job.Description)
                job.Experience = s.ExtractExperience(job.Description)
                
                if err := s.db.SaveJob(job); err != nil {
                    log.Printf("Error saving job: %v", err)
                } else {
                    log.Printf("✅ Saved: %s at %s (Score: %d)", job.Title, job.Company, job.Score)
                }
            }
        })
        
        s.collector.OnError(func(r *colly.Response, err error) {
            log.Printf("Request URL %s failed: %v", r.Request.URL, err)
        })
        
        err := s.collector.Visit(url)
        if err != nil {
            log.Printf("Error visiting BrighterMonday: %v", err)
        }
        
        time.Sleep(2 * time.Second)
    }
}

func (s *RealScraper) ScrapeFuzu() {
    categories := []string{"technology", "it", "security", "cyber-security"}
    
    for _, category := range categories {
        url := fmt.Sprintf("https://www.fuzu.com/kenya/%s-jobs", category)
        
        s.collector.OnHTML("div[class*='job-card'], div[data-testid*='job']", func(e *colly.HTMLElement) {
            title := e.ChildText("h3, h4, [class*='title']")
            company := e.ChildText("[class*='company'], [class*='employer']")
            location := e.ChildText("[class*='location'], [class*='address']")
            
            if title != "" && company != "" {
                job := &models.Job{
                    ID:         fmt.Sprintf("fz-%d", time.Now().UnixNano()),
                    Title:      strings.TrimSpace(title),
                    Company:    strings.TrimSpace(company),
                    Location:   strings.TrimSpace(location),
                    Source:     "Fuzu",
                    PostedDate: time.Now().Format("2006-01-02"),
                }
                
                // Try to get job URL
                jobURL := e.ChildAttr("a", "href")
                if jobURL != "" && !strings.HasPrefix(jobURL, "http") {
                    jobURL = "https://www.fuzu.com" + jobURL
                }
                job.URL = jobURL
                
                if jobURL != "" {
                    job.Description = s.ScrapeJobDescription(jobURL)
                }
                
                job.Skills = s.ExtractSkills(job.Description + " " + job.Title)
                job.TechStack = s.ExtractTechStack(job.Description)
                job.Score = s.CalculateScore(job)
                job.SalaryRange = s.ExtractSalary(job.Description)
                job.Experience = s.ExtractExperience(job.Description)
                
                if err := s.db.SaveJob(job); err != nil {
                    log.Printf("Error saving Fuzu job: %v", err)
                } else {
                    log.Printf("✅ Saved: %s at %s (Score: %d)", job.Title, job.Company, job.Score)
                }
            }
        })
        
        err := s.collector.Visit(url)
        if err != nil {
            log.Printf("Error visiting Fuzu: %v", err)
        }
        
        time.Sleep(2 * time.Second)
    }
}

func (s *RealScraper) ScrapeCompanyPages() {
    companies := []struct {
        name string
        url  string
    }{
        {"Safaricom", "https://www.safaricom.co.ke/careers/"},
        {"KCB Bank", "https://www.kcbgroup.com/careers/"},
        {"Equity Bank", "https://www.equitybankgroup.com/careers/"},
    }
    
    for _, company := range companies {
        s.collector.OnHTML("div.job-listing, li.job, tr.job-row, a[href*='job'], div[class*='job']", func(e *colly.HTMLElement) {
            title := e.ChildText("h3, h4, .title, .job-title")
            if title == "" {
                title = e.Text
            }
            
            // Filter out non-relevant jobs
            if title != "" && 
               (strings.Contains(strings.ToLower(title), "security") ||
                strings.Contains(strings.ToLower(title), "cyber") ||
                strings.Contains(strings.ToLower(title), "soc") ||
                strings.Contains(strings.ToLower(title), "cloud") ||
                strings.Contains(strings.ToLower(title), "network") ||
                strings.Contains(strings.ToLower(title), "analyst")) {
                
                job := &models.Job{
                    ID:         fmt.Sprintf("comp-%d", time.Now().UnixNano()),
                    Title:      strings.TrimSpace(title),
                    Company:    company.name,
                    Location:   "Nairobi, Kenya",
                    Source:     "Company Website",
                    PostedDate: time.Now().Format("2006-01-02"),
                }
                
                // Get job URL
                jobURL := e.ChildAttr("a", "href")
                if jobURL != "" && !strings.HasPrefix(jobURL, "http") {
                    jobURL = company.url + jobURL
                }
                job.URL = jobURL
                
                if jobURL != "" {
                    job.Description = s.ScrapeJobDescription(jobURL)
                }
                
                job.Skills = s.ExtractSkills(job.Description + " " + job.Title)
                job.TechStack = s.ExtractTechStack(job.Description)
                job.Score = s.CalculateScore(job)
                job.SalaryRange = s.ExtractSalary(job.Description)
                job.Experience = s.ExtractExperience(job.Description)
                
                if err := s.db.SaveJob(job); err != nil {
                    log.Printf("Error saving company job: %v", err)
                } else {
                    log.Printf("✅ Saved: %s at %s (Score: %d)", job.Title, job.Company, job.Score)
                }
            }
        })
        
        err := s.collector.Visit(company.url)
        if err != nil {
            log.Printf("Error visiting %s: %v", company.url, err)
        }
        
        time.Sleep(3 * time.Second)
    }
}

func (s *RealScraper) ScrapeJobDescription(url string) string {
    if url == "" {
        return ""
    }
    
    var description string
    descCollector := colly.NewCollector()
    
    descCollector.OnHTML("div.job-description, div.description, article.content, section.description, div[class*='desc']", func(e *colly.HTMLElement) {
        description = e.Text
    })
    
    descCollector.OnHTML("body", func(e *colly.HTMLElement) {
        if description == "" {
            description = e.Text
        }
    })
    
    err := descCollector.Visit(url)
    if err != nil {
        log.Printf("Error scraping job description from %s: %v", url, err)
    }
    
    return strings.TrimSpace(description)
}

func (s *RealScraper) ExtractSkills(text string) []string {
    skills := []string{}
    text = strings.ToLower(text)
    
    skillPatterns := map[string]string{
        `aws`: "AWS", `amazon web services`: "AWS",
        `azure`: "Azure", `microsoft azure`: "Azure",
        `gcp`: "GCP", `google cloud`: "GCP",
        `python`: "Python", 
        `go`: "Go", `golang`: "Go",
        `java`: "Java",
        `javascript`: "JavaScript", `node\.js`: "Node.js", `react`: "React",
        `docker`: "Docker", `kubernetes`: "Kubernetes", `k8s`: "Kubernetes",
        `terraform`: "Terraform", `ansible`: "Ansible",
        `fortinet`: "Fortinet", `palo alto`: "Palo Alto", `cisco`: "Cisco",
        `siem`: "SIEM", `splunk`: "Splunk", `qradar`: "QRadar", `arcsight`: "ArcSight",
        `cybersecurity`: "Cybersecurity", `cyber security`: "Cybersecurity",
        `cloud security`: "Cloud Security",
        `network security`: "Network Security",
        `soc`: "SOC", `security operations center`: "SOC",
        `incident response`: "Incident Response",
        `threat intelligence`: "Threat Intelligence",
        `vulnerability management`: "Vulnerability Management",
        `penetration testing`: "Penetration Testing", `pen testing`: "Penetration Testing",
        `firewall`: "Firewall", `vpn`: "VPN", `ids`: "IDS", `ips`: "IPS",
        `linux`: "Linux", `windows`: "Windows", `active directory`: "Active Directory",
    }
    
    for pattern, skill := range skillPatterns {
        matched, _ := regexp.MatchString(pattern, text)
        if matched {
            duplicate := false
            for _, existing := range skills {
                if existing == skill {
                    duplicate = true
                    break
                }
            }
            if !duplicate {
                skills = append(skills, skill)
            }
        }
    }
    
    return skills
}

func (s *RealScraper) ExtractTechStack(text string) []string {
    techStack := []string{}
    text = strings.ToLower(text)
    
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
            tech := strings.Title(pattern)
            duplicate := false
            for _, existing := range techStack {
                if existing == tech {
                    duplicate = true
                    break
                }
            }
            if !duplicate {
                techStack = append(techStack, tech)
            }
        }
    }
    
    return techStack
}

func (s *RealScraper) ExtractSalary(text string) string {
    text = strings.ToLower(text)
    
    salaryPatterns := []string{
        `ksh\s*[\d,]+\.?\d*\s*[-–]\s*ksh\s*[\d,]+\.?\d*`,
        `ksh\s*[\d,]+\.?\d*\s*[-–]\s*[\d,]+\.?\d*\s*ksh`,
        `salary\s*:?\s*ksh\s*[\d,]+\.?\d*\s*[-–]\s*ksh\s*[\d,]+\.?\d*`,
        `ksh\s*[\d,]+\.?\d*\s*per\s*month`,
        `ksh\s*[\d,]+\.?\d*\s*pm`,
    }
    
    for _, pattern := range salaryPatterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(text)
        if len(matches) > 0 {
            return strings.Title(matches[0])
        }
    }
    
    return "Negotiable"
}

func (s *RealScraper) ExtractExperience(text string) string {
    text = strings.ToLower(text)
    
    experiencePatterns := map[string]string{
        `(\d+)\+?\s*years?`: " years experience",
        `mid.*level`: "Mid Level",
        `senior`: "Senior Level",
        `junior`: "Junior Level",
        `entry.*level`: "Entry Level",
        `lead`: "Lead Level",
        `principal`: "Principal Level",
    }
    
    for pattern, level := range experiencePatterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(text)
        if len(matches) > 0 {
            if pattern == `(\d+)\+?\s*years?` && len(matches) > 1 {
                return matches[1] + level
            }
            return level
        }
    }
    
    return "Not specified"
}

func (s *RealScraper) CalculateScore(job *models.Job) int {
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
    
    // Experience level (20 points)
    if strings.Contains(text, "3 years") || strings.Contains(text, "mid-level") {
        score += 15
    } else if strings.Contains(text, "5 years") || strings.Contains(text, "senior") {
        score += 20
    } else if strings.Contains(text, "1 year") || strings.Contains(text, "junior") {
        score += 10
    } else {
        score += 5
    }
    
    // Salary indication (10 points)
    if strings.Contains(text, "ksh") || strings.Contains(text, "salary") {
        score += 10
    }
    
    // Company reputation (10 points)
    goodCompanies := []string{"safaricom", "kcb", "equity", "google", "microsoft", "amazon"}
    for _, company := range goodCompanies {
        if strings.Contains(strings.ToLower(job.Company), company) {
            score += 10
            break
        }
    }
    
    return min(score, 100)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}