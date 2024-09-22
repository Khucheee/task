package repository

import (
	"context"
	"customers_kuber/closer"
	"customers_kuber/config"
	"customers_kuber/logger"
	"customers_kuber/model"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
)

var entityRepositoryInstance *entityRepository

type EntityRepository interface {
	SaveEntity(e model.Entity)
	GetEntities() ([]model.Entity, error)
	UpdateEntity(update model.Entity) error
	DeleteEntity(uuid2 uuid.UUID) error
}

type entityRepository struct {
	db *gorm.DB
}

func GetEntityRepository() (EntityRepository, error) {
	if entityRepositoryInstance != nil {
		return entityRepositoryInstance, nil
	}

	ctx := context.Background()

	//устанавливаемм адрес базы
	dbConfig := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.PostgresHost, config.PostgresUser, config.PostgresPassword, config.PostgresDatabaseName, config.PostgresPort)

	//открываем соединение
	dbConnect, err := gorm.Open(postgres.Open(dbConfig), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
	})

	//инициализируем инстанс репозитория
	entityRepositoryInstance = &entityRepository{dbConnect}
	if err != nil {
		return entityRepositoryInstance, fmt.Errorf("failed to create database connection: %s", err)
	}

	//делаем миграцию
	err = dbConnect.AutoMigrate(&model.Entity{})
	if err != nil {
		return entityRepositoryInstance, fmt.Errorf("migration failed: %s", err)
	}

	//передаем функцию закрытия в клозер для graceful shutdown
	closer.CloseFunctions = append(closer.CloseFunctions, func() {
		dbInterface, err := dbConnect.DB()
		if err != nil {
			slog.ErrorContext(logger.WithLogError(ctx, err), "failed to get DB interface while closing connection")
			return
		}
		if err := dbInterface.Close(); err != nil {
			slog.ErrorContext(logger.WithLogError(ctx, err), "failed while closing DB connection")
			return
		}
		slog.Info("entityRepository closed successfully")
	})
	return entityRepositoryInstance, nil
}

func (repository *entityRepository) SaveEntity(entity model.Entity) {
	//сохраняю Entity в базу
	ctx := context.Background()
	for i := 0; i <= config.RepositoryRetries; i++ {
		if err := repository.db.Create(&entity).Error; err != nil {
			slog.ErrorContext(
				logger.WithLogError(ctx, err),
				"listener failed trying to save entity, retry "+string(i)+" of "+string(config.RepositoryRetries))
			continue
		}
		slog.Info("entity successfully saved to repository")
		break
	}
}

func (repository *entityRepository) GetEntities() ([]model.Entity, error) {

	//запрашиваем в базе все записи Entities
	var entities []model.Entity
	if result := repository.db.Find(&entities); result.Error != nil {
		return entities, result.Error
	}

	return entities, nil
}

func (repository *entityRepository) UpdateEntity(entity model.Entity) error {
	//проверяем наличие записи в базе
	checkExistence := model.Entity{}
	result := repository.db.Where("id=?", entity.Id).First(&checkExistence)
	if result.Error != nil {
		return result.Error
	}

	//если запись нашлась, то обновляю её
	if result = repository.db.Model(&entity).Where("id=?", entity.Id).Updates(map[string]interface{}{"id": entity.Id, "Test": entity.Test}); result.Error != nil {
		return result.Error
	}
	slog.Info("entity successfully updated in repository")
	return nil
}

func (repository *entityRepository) DeleteEntity(id uuid.UUID) error {
	entity := model.Entity{}
	if result := repository.db.Where("Id = ?", id).Delete(&entity); result.Error != nil {
		return fmt.Errorf("failed to delete data from repository: %s", result.Error)
	}
	slog.Info("entity successfully updated in repository")
	return nil
}
