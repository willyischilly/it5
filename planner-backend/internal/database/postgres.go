package database

import (
	"fmt"
	"log"
	"os"
	"strings"

	"planner-backend/internal/config"
	"planner-backend/internal/models"
	"planner-backend/pkg/auth"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	if err := ensureDatabase(cfg); err != nil {
		return nil, err
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	cleanupOrphans(db)
	migrateUserFIOBeforeAutoMigrate(db)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Work{},
		&models.DeploymentContour{},
		&models.Request{},
		&models.Task{},
		&models.TaskLog{},
		&models.RequestLog{},
	); err != nil {
		return nil, err
	}

	dropLegacyUserNameColumn(db)

	seedContours(db)
	seedWorks(db)
	seedAdmin(db)
	return db, nil
}

func cleanupOrphans(db *gorm.DB) {
	db.Exec(`UPDATE tasks SET executor_id = NULL WHERE executor_id IS NOT NULL AND executor_id NOT IN (SELECT id FROM users)`)
	db.Exec(`DELETE FROM task_logs WHERE user_id NOT IN (SELECT id FROM users)`)
	db.Exec(`DELETE FROM request_logs WHERE user_id NOT IN (SELECT id FROM users)`)
}

func ensureDatabase(cfg *config.Config) error {
	adminDSN := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBPort,
	)
	adminDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}

	var exists int
	err = adminDB.Raw("SELECT 1 FROM pg_database WHERE datname = ?", cfg.DBName).Scan(&exists).Error
	if err != nil {
		return err
	}
	if exists == 1 {
		return nil
	}

	createSQL := fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)
	if err := adminDB.Exec(createSQL).Error; err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("create database %s: %w", cfg.DBName, err)
	}
	log.Printf("database %s created", cfg.DBName)
	return nil
}

func seedContours(db *gorm.DB) {
	names := []string{"Dev", "Qa", "Uat", "Prod"}
	for _, name := range names {
		var count int64
		db.Model(&models.DeploymentContour{}).Where("name = ?", name).Count(&count)
		if count == 0 {
			if err := db.Create(&models.DeploymentContour{Name: name}).Error; err != nil {
				log.Printf("seed contour %s: %v", name, err)
			}
		}
	}
}

func seedWorks(db *gorm.DB) {
	works := []models.Work{
		{Name: "Развертывание VM", Description: "Создание и настройка виртуальной машины", NormativeHours: 2},
		{Name: "Настройка сети", Description: "VLAN, firewall, маршрутизация", NormativeHours: 1},
		{Name: "Установка СУБД", Description: "PostgreSQL / кластер", NormativeHours: 3},
		{Name: "Развертывание приложения", Description: "Деплой backend-сервиса", NormativeHours: 2},
		{Name: "Настройка мониторинга", Description: "Метрики и алерты", NormativeHours: 1},
		{Name: "Резервное копирование", Description: "Политики backup", NormativeHours: 1},
		{Name: "Приемочное тестирование", Description: "Smoke / регресс на контуре", NormativeHours: 2},
	}
	for _, w := range works {
		var count int64
		db.Model(&models.Work{}).Where("name = ?", w.Name).Count(&count)
		if count == 0 {
			if err := db.Create(&w).Error; err != nil {
				log.Printf("seed work %s: %v", w.Name, err)
			}
		}
	}
}

func seedAdmin(db *gorm.DB) {
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if email == "" {
		email = "admin@planner.local"
	}
	if password == "" {
		password = "admin123456"
	}

	var count int64
	db.Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("seed admin hash: %v", err)
		return
	}

	user := &models.User{
		Email:      email,
		Password:   hash,
		Role:       models.RoleAdmin,
		LastName:   "Администратор",
		FirstName:  "Системный",
		Patronymic: "—",
	}
	if err := db.Create(user).Error; err != nil {
		log.Printf("seed admin: %v", err)
		return
	}
	log.Printf("seed admin created: %s", email)
}

// migrateUserFIOBeforeAutoMigrate добавляет ФИО до AutoMigrate (NOT NULL), чтобы не падать на старых users.
func migrateUserFIOBeforeAutoMigrate(db *gorm.DB) {
	if !db.Migrator().HasTable(&models.User{}) {
		return
	}
	if db.Migrator().HasColumn(&models.User{}, "last_name") {
		db.Exec(`UPDATE users SET last_name = 'Пользователь' WHERE last_name IS NULL OR TRIM(last_name) = ''`)
		db.Exec(`UPDATE users SET first_name = '—' WHERE first_name IS NULL OR TRIM(first_name) = ''`)
		db.Exec(`UPDATE users SET patronymic = '—' WHERE patronymic IS NULL OR TRIM(patronymic) = ''`)
		return
	}

	hasName := db.Migrator().HasColumn(&models.User{}, "name")

	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name varchar(100)`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name varchar(100)`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS patronymic varchar(100)`)

	if hasName {
		db.Exec(`
			UPDATE users SET
				last_name = COALESCE(NULLIF(TRIM(last_name), ''), NULLIF(TRIM(name), ''), 'Пользователь'),
				first_name = COALESCE(NULLIF(TRIM(first_name), ''), '—'),
				patronymic = COALESCE(NULLIF(TRIM(patronymic), ''), '—')
		`)
	} else {
		db.Exec(`
			UPDATE users SET
				last_name = COALESCE(NULLIF(TRIM(last_name), ''), 'Пользователь'),
				first_name = COALESCE(NULLIF(TRIM(first_name), ''), '—'),
				patronymic = COALESCE(NULLIF(TRIM(patronymic), ''), '—')
		`)
	}

	db.Exec(`ALTER TABLE users ALTER COLUMN last_name SET NOT NULL`)
	db.Exec(`ALTER TABLE users ALTER COLUMN first_name SET NOT NULL`)
	db.Exec(`ALTER TABLE users ALTER COLUMN patronymic SET NOT NULL`)
}

func dropLegacyUserNameColumn(db *gorm.DB) {
	if db.Migrator().HasColumn(&models.User{}, "name") {
		if err := db.Migrator().DropColumn(&models.User{}, "name"); err != nil {
			log.Printf("drop users.name: %v", err)
		}
	}
}
