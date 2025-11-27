package database

import (

    "fmt"
    "log"
    "time"

    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/clause" // <-- needed for OnConflict
    "gorm.io/gorm/logger"
    "gorm.io/datatypes"
    
	"github.com/C9b3rD3vi1/jobhunter-tool/models"
)

type DB struct {
    *gorm.DB
}

func InitDB() (*DB, error) {
    // Connect to SQLite database
    db, err := gorm.Open(sqlite.Open("jobhunter.db"), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent), // Reduce log noise
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %v", err)
    }

    // Auto migrate all tables
    err = db.AutoMigrate(
        &models.Job{},
        &models.Application{},
        &models.UserSkill{},
    )
    if err != nil {
        return nil, fmt.Errorf("failed to auto migrate: %v", err)
    }

    // Create default skills
    defaultSkills := []string{
        "AWS", "Python", "Go", "Fortinet", "SIEM", "Docker", 
        "Kubernetes", "Terraform", "JavaScript", "Cybersecurity", 
        "Cloud Security", "SOC", "Network Security",
    }
    
    for _, skill := range defaultSkills {
        userSkill := models.UserSkill{Skill: skill, Category: "Technical"}
        result := db.FirstOrCreate(&userSkill, models.UserSkill{Skill: skill})
        if result.Error != nil {
            log.Printf("Error creating skill %s: %v", skill, result.Error)
        }
    }

    log.Println("Database initialized and migrated successfully")
    return &DB{db}, nil
}

func (db *DB) SaveJob(job *models.Job) error {
    // Generate ID if not provided
    if job.ID == "" {
        job.ID = fmt.Sprintf("%d", time.Now().UnixNano())
    }

    // Ensure Skills and TechStack are properly formatted JSON
    if job.Skills == nil {
        job.Skills = datatypes.JSON([]byte(`[]`))
    }
    if job.TechStack == nil {
        job.TechStack = datatypes.JSON([]byte(`[]`))
    }

    // Use GORM's Create with conflict handling
    result := db.Clauses(
        clause.OnConflict{
            Columns:   []clause.Column{{Name: "url"}},
            DoUpdates: clause.Assignments(map[string]interface{}{
                "title":        job.Title,
                "company":      job.Company,
                "location":     job.Location,
                "description":  job.Description,
                "salary_range": job.SalaryRange,
                "experience":   job.Experience,
                "posted_date":  job.PostedDate,
                "source":       job.Source,
                "score":        job.Score,
                "skills":       job.Skills,
                "tech_stack":   job.TechStack,
            }),
        },
    ).Create(job)

    if result.Error != nil {
        log.Printf("Error saving job %s: %v", job.Title, result.Error)
    }
    return result.Error
}

func (db *DB) GetJobs(limit, offset int) ([]models.Job, error) {
    var jobs []models.Job
    result := db.Order("score DESC, posted_date DESC").
        Limit(limit).
        Offset(offset).
        Find(&jobs)
    
    if result.Error != nil {
        return nil, result.Error
    }

    // Convert JSON fields to proper slices for templates
    for i := range jobs {
        if jobs[i].Skills == nil {
            jobs[i].Skills = datatypes.JSON([]byte(`[]`))
        }
        if jobs[i].TechStack == nil {
            jobs[i].TechStack = datatypes.JSON([]byte(`[]`))
        }
    }
    
    return jobs, nil
}

func (db *DB) GetJobByID(id string) (*models.Job, error) {
    var job models.Job
    result := db.First(&job, "id = ?", id)
    if result.Error != nil {
        return nil, result.Error
    }
    return &job, nil
}

func (db *DB) SaveApplication(app *models.Application) error {
    if app.ID == "" {
        app.ID = fmt.Sprintf("%d", time.Now().UnixNano())
    }
    
    result := db.Create(app)
    return result.Error
}

func (db *DB) GetApplications() ([]models.Application, error) {
    var applications []models.Application
    result := db.Order("applied_date DESC").Find(&applications)
    return applications, result.Error
}

func (db *DB) UpdateApplicationStatus(id, status string) error {
    result := db.Model(&models.Application{}).Where("id = ?", id).Update("status", status)
    return result.Error
}

func (db *DB) GetUserSkills() ([]string, error) {
    var userSkills []models.UserSkill
    result := db.Find(&userSkills)
    if result.Error != nil {
        return nil, result.Error
    }

    skills := make([]string, len(userSkills))
    for i, userSkill := range userSkills {
        skills[i] = userSkill.Skill
    }
    
    return skills, nil
}

func (db *DB) AddUserSkill(skill string) error {
    userSkill := models.UserSkill{Skill: skill, Category: "Technical"}
    result := db.FirstOrCreate(&userSkill, models.UserSkill{Skill: skill})
    return result.Error
}

func (db *DB) GetJobStats() (totalJobs, highScoreJobs int, err error) {
    var total int64
    result := db.Model(&models.Job{}).Count(&total)
    if result.Error != nil {
        return 0, 0, result.Error
    }
    totalJobs = int(total)

    var highScore int64
    result = db.Model(&models.Job{}).Where("score >= ?", 80).Count(&highScore)
    if result.Error != nil {
        return 0, 0, result.Error
    }
    highScoreJobs = int(highScore)

    return totalJobs, highScoreJobs, nil
}

func (db *DB) GetApplicationStats() (int, error) {
    var count int64
    result := db.Model(&models.Application{}).Count(&count)
    return int(count), result.Error
}

func (db *DB) GetJobsByCompany(companyName string) ([]models.Job, error) {
    var jobs []models.Job
    result := db.Where("company = ?", companyName).Find(&jobs)
    return jobs, result.Error
}

func (db *DB) Close() error {
    sqlDB, err := db.DB.DB()
    if err != nil {
        return err
    }
    return sqlDB.Close()
}