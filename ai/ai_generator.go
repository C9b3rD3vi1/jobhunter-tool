package ai

import (
    "context"
    "fmt"
    "strings"

    openai "github.com/sashabaranov/go-openai"
)

type AIGenerator struct {
    client *openai.Client
}

func NewAIGenerator(apiKey string) *AIGenerator {
    if apiKey == "" {
        return &AIGenerator{client: nil}
    }
    return &AIGenerator{
        client: openai.NewClient(apiKey),
    }
}

func (g *AIGenerator) GenerateCoverLetter(jobTitle, company, jobDescription, userProfile string) (string, error) {
    if g.client == nil {
        return g.generateFallbackCoverLetter(jobTitle, company), nil
    }

    prompt := fmt.Sprintf(`
Generate a professional cover letter for a cybersecurity position with the following details:

Job Title: %s
Company: %s
Job Description: %s
My Profile: %s

Please write a compelling cover letter that highlights relevant skills and experience. Keep it professional and tailored to the specific role.`, jobTitle, company, jobDescription, userProfile)

    resp, err := g.client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT3Dot5Turbo,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: prompt,
                },
            },
            MaxTokens: 500,
        },
    )

    if err != nil {
        return g.generateFallbackCoverLetter(jobTitle, company), err
    }

    return resp.Choices[0].Message.Content, nil
}

func (g *AIGenerator) GenerateSkillsAnalysis(jobDescription string, userSkills []string) (string, []string, []string, int) {
    // Real skills gap analysis
    descLower := strings.ToLower(jobDescription)
    userSkillsLower := make([]string, len(userSkills))
    for i, skill := range userSkills {
        userSkillsLower[i] = strings.ToLower(skill)
    }

    var matchingSkills []string
    var missingSkills []string

    // Extract required skills from job description
    requiredSkills := g.extractRequiredSkills(descLower)
    
    // Find matches and gaps
    for _, reqSkill := range requiredSkills {
        found := false
        for _, userSkill := range userSkillsLower {
            if strings.Contains(userSkill, reqSkill) || strings.Contains(reqSkill, userSkill) {
                matchingSkills = append(matchingSkills, reqSkill)
                found = true
                break
            }
        }
        if !found {
            missingSkills = append(missingSkills, reqSkill)
        }
    }

    // Calculate fit score
    fitScore := 0
    if len(requiredSkills) > 0 {
        fitScore = (len(matchingSkills) * 100) / len(requiredSkills)
    }

    recommendations := g.generateRecommendations(missingSkills, matchingSkills)

    analysis := fmt.Sprintf(`
## Skills Fit Analysis
**Fit Score: %d%%**

**✅ Matching Skills (%d):** %s
**❌ Missing Skills (%d):** %s
**💡 Recommendations:** %s`,
        fitScore, len(matchingSkills), strings.Join(matchingSkills, ", "),
        len(missingSkills), strings.Join(missingSkills, ", "),
        strings.Join(recommendations, "; "))

    return analysis, matchingSkills, missingSkills, fitScore
}

func (g *AIGenerator) extractRequiredSkills(description string) []string {
    skills := []string{}
    
    skillKeywords := []string{
        "aws", "azure", "gcp", "cloud", "python", "go", "java", "javascript",
        "docker", "kubernetes", "terraform", "ansible", "jenkins", "git",
        "fortinet", "palo alto", "cisco", "check point", "siem", "splunk",
        "qradar", "arcsight", "wireshark", "metasploit", "nessus", "nexpose",
        "burp suite", "nmap", "security+", "ceh", "cissp", "oscp", "gsoc",
        "firewall", "vpn", "ids", "ips", "dlp", "soc", "incident response",
        "threat intelligence", "vulnerability management", "penetration testing",
        "risk assessment", "compliance", "iso 27001", "nist", "pci dss",
        "linux", "windows", "active directory", "network security",
    }
    
    for _, skill := range skillKeywords {
        if strings.Contains(description, skill) {
            skills = append(skills, skill)
        }
    }
    
    return skills
}

func (g *AIGenerator) generateRecommendations(missingSkills, matchingSkills []string) []string {
    recommendations := []string{}
    
    transferableSkills := map[string]string{
        "splunk": "Consider highlighting your experience with FortiAnalyzer or other log analysis tools",
        "python": "Emphasize your Go programming experience as both are used for security automation",
        "azure":  "Your AWS experience is transferable to Azure cloud security concepts",
        "palo alto": "Your Fortinet firewall experience is directly relevant to Palo Alto networks",
    }
    
    for _, missing := range missingSkills {
        if suggestion, exists := transferableSkills[missing]; exists {
            recommendations = append(recommendations, suggestion)
        }
    }
    
    if len(missingSkills) > 0 {
        recommendations = append(recommendations, 
            fmt.Sprintf("Focus on learning: %s", strings.Join(missingSkills[:min(3, len(missingSkills))], ", ")))
    }
    
    if len(matchingSkills) > 0 {
        recommendations = append(recommendations,
            fmt.Sprintf("Strongly emphasize your experience with: %s", strings.Join(matchingSkills[:min(5, len(matchingSkills))], ", ")))
    }
    
    return recommendations
}

func (g *AIGenerator) generateFallbackCoverLetter(jobTitle, company string) string {
    return fmt.Sprintf(`
Dear Hiring Manager,

I am writing to express my keen interest in the %s position at %s. With my background in cybersecurity and cloud security, I am confident in my ability to contribute effectively to your security team.

My experience includes working with Fortinet firewalls, AWS security services, SIEM solutions, and implementing security best practices in cloud environments. I have hands-on experience with security monitoring, incident response, and vulnerability management.

I am particularly impressed by %s's commitment to security excellence and would be thrilled to bring my technical skills and dedication to your organization.

Thank you for considering my application. I look forward to the opportunity to discuss how my skills and experience can benefit your security initiatives.

Sincerely,
[Your Name]
`, jobTitle, company, company)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}