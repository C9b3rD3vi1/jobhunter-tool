package models

import (
    "time"
    "gorm.io/gorm"
)

type Job struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    Title       string    `gorm:"not null" json:"title"`
    Company     string    `gorm:"not null" json:"company"`
    Location    string    `json:"location"`
    Description string    `json:"description"`
    SalaryRange string    `json:"salary_range"`
    Experience  string    `json:"experience"`
    PostedDate  string    `json:"posted_date"`
    Source      string    `json:"source"`
    URL         string    `gorm:"unique" json:"url"`
    Score       int       `gorm:"default:0" json:"score"`
    Skills      []string  `gorm:"serializer:json" json:"skills"`
    TechStack   []string  `gorm:"serializer:json" json:"tech_stack"`
    CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type Application struct {
    ID            string    `gorm:"primaryKey" json:"id"`
    JobID         string    `json:"job_id"`
    Company       string    `json:"company"`
    Role          string    `json:"role"`
    AppliedDate   string    `json:"applied_date"`
    Status        string    `gorm:"default:Applied" json:"status"`
    HiringManager string    `json:"hiring_manager"`
    Notes         string    `json:"notes"`
    CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type UserSkill struct {
    ID       uint   `gorm:"primaryKey" json:"id"`
    Skill    string `gorm:"unique" json:"skill"`
    Category string `json:"category"`
}

type SkillsAnalysis struct {
    MissingSkills   []string `json:"missing_skills"`
    MatchingSkills  []string `json:"matching_skills"`
    Transferable    []string `json:"transferable_skills"`
    FitScore        int      `json:"fit_score"`
    Recommendations []string `json:"recommendations"`
}