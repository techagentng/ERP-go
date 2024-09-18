package db

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/techagentng/telair-erp/config"
	"github.com/techagentng/telair-erp/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormDB struct {
	DB *gorm.DB
}

func GetDB(c *config.Config) *GormDB {
	gormDB := &GormDB{}
	gormDB.Init(c)
	return gormDB
}

func (g *GormDB) Init(c *config.Config) {
	g.DB = getPostgresDB(c)

	if err := migrate(g.DB); err != nil {
		log.Fatalf("unable to run migrations: %v", err)
	}
}

func getPostgresDB(c *config.Config) *gorm.DB {
	log.Printf("Connecting to postgres: %+v", c)
	postgresDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d TimeZone=Africa/Lagos",
		c.PostgresHost, c.PostgresUser, c.PostgresPassword, c.PostgresDB, c.PostgresPort)

	// Create GORM DB instance
	gormConfig := &gorm.Config{}
	if c.Env != "prod" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN: postgresDSN,
	}), gormConfig)
	if err != nil {
		log.Fatal(err)
	}

	return gormDB
}

func SeedRoles(db *gorm.DB) error {
    roles := []models.Role{
        {ID: uuid.New(), Name: "Admin"},
        {ID: uuid.New(), Name: "User"},
    }

    for _, role := range roles {
        if err := db.FirstOrCreate(&role, models.Role{Name: role.Name}).Error; err != nil {
            return err
        }
    }

    return nil
}

func migrate(db *gorm.DB) error {
	// AutoMigrate all the models
	err := db.AutoMigrate(
		&models.User{},
		&models.Trailer{},
		&models.Role{}, 
	)
	if err != nil {
		return fmt.Errorf("migrations error: %v", err)
	}

	// Add any additional migrations here if needed

	return nil
}

func seedRoles(db *gorm.DB) error {
    roles := []string{"Admin", "User"}

    for _, roleName := range roles {
        var existingRole models.Role
        if err := db.Where("name = ?", roleName).First(&existingRole).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                newRole := models.Role{Name: roleName}
                if err := db.Create(&newRole).Error; err != nil {
                    return fmt.Errorf("error creating role %s: %v", roleName, err)
                }
                log.Printf("Role %s created successfully", roleName)
            } else {
                return fmt.Errorf("error checking role existence: %v", err)
            }
        }
    }
    return nil
}
