package models

type Job struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Company      string    `json:"company"`
	Location     string    `json:"location"`
	Description  string    `json:"description"`
	SalaryRange  string    `json:"salary_range"`
	Experience   string    `json:"experience"`
	PostedDate   string    `json:"posted_date"`
	Source       string    `json:"source"`
	URL          string    `json:"url"`
	Score        int       `json:"score"`
	Skills       []string  `json:"skills"`
	TechStack    []string  `json:"tech_stack"`
}

type Application struct {
	ID              string `json:"id"`
	JobID           string `json:"job_id"`
	Company         string `json:"company"`
	Role            string `json:"role"`
	AppliedDate     string `json:"applied_date"`
	Status          string `json:"status"`
	HiringManager   string `json:"hiring_manager"`
	Notes           string `json:"notes"`
}

type SkillsAnalysis struct {
	MissingSkills    []string `json:"missing_skills"`
	MatchingSkills   []string `json:"matching_skills"`
	Transferable     []string `json:"transferable_skills"`
	FitScore         int      `json:"fit_score"`
	Recommendations  []string `json:"recommendations"`
}