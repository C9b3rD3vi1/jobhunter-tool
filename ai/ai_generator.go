package ai

import (
    "context"
    "fmt"
    "strings"

    "github.com/sashabaranov/go-openai"
    "github.com/C9b3rD3vi1/jobhunter-tool/models"
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

func (g *AIGenerator) GenerateSkillsAnalysis(jobDescription string, userSkills []string) models.SkillsAnalysis {
    analysis := models.SkillsAnalysis{
        MissingSkills:   []string{},
        MatchingSkills:  []string{},
        Transferable:    []string{},
        Recommendations: []string{},
    }
    
    descLower := strings.ToLower(jobDescription)
    userSkillsLower := make([]string, len(userSkills))
    for i, skill := range userSkills {
        userSkillsLower[i] = strings.ToLower(skill)
    }

    // Extract required skills from job description
    requiredSkills := g.extractRequiredSkills(descLower)
    
    // Find matches and gaps
    for _, reqSkill := range requiredSkills {
        found := false
        for _, userSkill := range userSkillsLower {
            if strings.Contains(userSkill, reqSkill) || strings.Contains(reqSkill, userSkill) {
                analysis.MatchingSkills = append(analysis.MatchingSkills, reqSkill)
                found = true
                break
            }
        }
        if !found {
            analysis.MissingSkills = append(analysis.MissingSkills, reqSkill)
        }
    }

    // Calculate fit score
    if len(requiredSkills) > 0 {
        analysis.FitScore = (len(analysis.MatchingSkills) * 100) / len(requiredSkills)
    }

    // Generate transferable skills and recommendations
    analysis.Transferable = g.generateTransferableSkills(analysis.MissingSkills, analysis.MatchingSkills)
    analysis.Recommendations = g.generateRecommendations(analysis.MissingSkills, analysis.MatchingSkills, analysis.FitScore)

    return analysis
}

func (g *AIGenerator) extractRequiredSkills(description string) []string {
    skills := []string{}
    
    skillKeywords := []string{
        "aws", "azure", "gcp", "cloud", "python", "go", "golang", "java", "javascript",
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
            // Capitalize skill name
            capitalized := strings.Title(skill)
            // Avoid duplicates
            duplicate := false
            for _, existing := range skills {
                if existing == capitalized {
                    duplicate = true
                    break
                }
            }
            if !duplicate {
                skills = append(skills, capitalized)
            }
        }
    }
    
    return skills
}

func (g *AIGenerator) generateTransferableSkills(missingSkills, matchingSkills []string) []string {
    transferable := []string{}
    
    transferMap := map[string]string{
        "Splunk": "FortiAnalyzer log analysis experience",
        "Python": "Go programming experience for automation",
        "Azure":  "AWS cloud security knowledge",
        "Palo Alto": "Fortinet firewall administration",
        "QRadar": "SIEM monitoring experience",
        "Nessus": "Vulnerability assessment background",
        "Metasploit": "Penetration testing fundamentals",
    }
    
    for _, missing := range missingSkills {
        if transfer, exists := transferMap[missing]; exists {
            transferable = append(transferable, transfer)
        }
    }
    
    return transferable
}

func (g *AIGenerator) generateRecommendations(missingSkills, matchingSkills []string, fitScore int) []string {
    recommendations := []string{}
    
    if fitScore >= 80 {
        recommendations = append(recommendations, "Excellent fit! Focus on highlighting your matching skills in applications.")
    } else if fitScore >= 60 {
        recommendations = append(recommendations, "Good fit. Emphasize transferable skills and relevant experience.")
    } else {
        recommendations = append(recommendations, "Consider upskilling in missing areas or focusing on roles with better alignment.")
    }
    
    if len(missingSkills) > 0 {
        if len(missingSkills) <= 3 {
            recommendations = append(recommendations, 
                fmt.Sprintf("Consider learning: %s", strings.Join(missingSkills, ", ")))
        } else {
            recommendations = append(recommendations,
                fmt.Sprintf("Priority skills to learn: %s", strings.Join(missingSkills[:3], ", ")))
        }
    }
    
    if len(matchingSkills) > 0 {
        recommendations = append(recommendations,
            fmt.Sprintf("Strongly emphasize: %s", strings.Join(matchingSkills, ", ")))
    }
    
    if len(missingSkills) > 0 && len(g.generateTransferableSkills(missingSkills, matchingSkills)) > 0 {
        recommendations = append(recommendations,
            "Highlight your transferable skills to bridge experience gaps")
    }
    
    return recommendations
}

func (g *AIGenerator) GenerateCoverLetter(jobTitle, company, jobDescription, userProfile string) (string, error) {
    if g.client == nil {
        return g.generateFallbackCoverLetter(jobTitle, company, jobDescription), nil
    }

    prompt := fmt.Sprintf(`
Generate a professional cover letter for a cybersecurity position with the following details:

Job Title: %s
Company: %s
Job Description: %s
My Profile: %s

Please write a compelling, professional cover letter that highlights relevant skills and experience. Focus on cybersecurity aspects mentioned in the job description. Keep it concise (250-300 words) and tailored to the specific role.`, jobTitle, company, jobDescription, userProfile)

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
        return g.generateFallbackCoverLetter(jobTitle, company, jobDescription), err
    }

    return resp.Choices[0].Message.Content, nil
}

func (g *AIGenerator) generateFallbackCoverLetter(jobTitle, company, jobDescription string) string {
    return fmt.Sprintf(`
Dear Hiring Manager,

I am writing to express my enthusiastic interest in the %s position at %s, as advertised. With my comprehensive background in cybersecurity and cloud security, I am confident in my ability to contribute significantly to your security initiatives.

My experience encompasses working with Fortinet firewalls, AWS security services, SIEM solutions, and implementing robust security practices in cloud environments. I have hands-on expertise in security monitoring, incident response, vulnerability management, and threat intelligence.

Having reviewed the job description, I am particularly excited about the opportunity to apply my skills in %s. My technical proficiencies align well with your requirements, and I am eager to bring my dedication to security excellence to your organization.

Thank you for considering my application. I look forward to the opportunity to discuss how my skills and experience can benefit %s's security objectives.

Sincerely,
[Your Name]
`, jobTitle, company, company, company)
}