package controller

import (
	"context"
	"customers_kuber/closer"
	"customers_kuber/logger"
	"customers_kuber/middleware"
	"customers_kuber/model"
	"customers_kuber/service"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"os"
)

var entityControllerInstance *entityController

type EntityController interface {
	Route(context.Context)
}

// entityController принимает входящие http запросы и передает их в service,
// в зависимости от ответа сервсиса генерирует и отправляет ответ на запрос
type entityController struct {
	service service.EntityService
	server  *http.Server //храню сервер для graceful shutdown
}

// GetEntityController возвращает контроллер, обрабатывающий http запросы,
// если в сервисе возникла ошибка, она будет возвращаеться в теле ответа,
// реализовано через синглтон
func GetEntityController() EntityController {
	if entityControllerInstance != nil {
		return entityControllerInstance
	}
	ctx := context.Background()
	entityService, err := service.GetEntityService()
	entityControllerInstance = &entityController{service: entityService}
	if err != nil {
		ctx = logger.WithLogError(ctx, err)
		slog.ErrorContext(ctx, "failed to create controller")
		os.Exit(1)
	}
	closer.CloseFunctions = append(closer.CloseFunctions, func() {
		go func() {
			if err := entityControllerInstance.server.Shutdown(ctx); err != nil {
				ctx = logger.WithLogError(ctx, err)
				slog.ErrorContext(ctx, "failed to shut server down")
			}
		}()
		slog.Info("entityController closed successfully")
	})
	return entityControllerInstance
}

// Route запускает сервер и маршрутизирует запросы
func (controller *entityController) Route(ctx context.Context) {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logging())
	router.GET("/getAll", controller.getAllEntities)
	router.POST("/create", controller.saveEntity)
	router.PUT("/:id", controller.updateEntity)
	router.DELETE("/:id", controller.deleteEntity)

	//Запускаем сервер
	//todo конфигурировать порт моего сервера
	server := &http.Server{Addr: ":8080", Handler: router}
	controller.server = server
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		ctx = logger.WithLogError(ctx, err)
		slog.ErrorContext(ctx, "failed to start server")
		os.Exit(1)
	}
}

// saveEntity отвечает за сохранение новых model.Entity
// для успешного создания необходимо передать все поля структуры
// 0 <= age <= 100
func (controller *entityController) saveEntity(ctx *gin.Context) {

	slog.Info("saveEntity in controller started")

	//Парсинг полученной json и валидация полей
	test := &model.Test{}
	if err := ctx.BindJSON(test); err != nil {
		ctx.JSON(http.StatusBadRequest, err.Error())
		slog.Info("saveEntity in controller finished")
		return
	}

	//сохранение данных, если сервис вернет ошибку, то 500
	if err := controller.service.SaveEntity(ctx, *test); err != nil {
		slog.ErrorContext(logger.WithLogError(ctx, err), "failed to save entity")
		ctx.JSON(http.StatusInternalServerError, err.Error())
		slog.Info("saveEntity in controller finished")
		return
	}

	//если все ок, 200
	ctx.Status(http.StatusAccepted)
	slog.Info("saveEntity in controller finished")
}

// getAllEntities возвращает все ранее сохраненные в приложении model.Entity
func (controller *entityController) getAllEntities(ctx *gin.Context) {

	slog.Info("getAllEntities in controller started")

	//получение всех существующих entity, если вернется ошибка, то 500
	entities, err := controller.service.GetAllEntities(ctx, ctx.Request.URL.Path)
	if err != nil {
		slog.ErrorContext(logger.WithLogError(ctx, err), "failed to get all entities")
		ctx.JSON(http.StatusInternalServerError, err.Error())
		slog.Info("getAllEntities in controller finished")
		return
	}
	ctx.JSON(http.StatusOK, entities)
	slog.Info("getAllEntities in controller finished")
}

// updateEntity обновляет ранее сохраненный model.Entity
// для успешного обновления id в url и id в body запроса должны быть типа UUID и совпадать
// 0 <= age <= 100
func (controller *entityController) updateEntity(ctx *gin.Context) {

	slog.Info("updateEntity in controller started")

	//проверка на uuid, если в URL передан не uuid, то 400
	uuidFromURL, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err.Error())
		slog.Info("updateEntity in controller finished")
		return
	}

	//парсю json и передаю сервису, если не прошли валидации полей, 400
	entity := &model.EntityForUpdate{}
	if err := ctx.ShouldBindJSON(entity); err != nil {
		ctx.JSON(http.StatusBadRequest, err.Error())
		slog.Info("updateEntity in controller finished")
		return
	}

	//сравниваем id в теле и в урле, если не совпадают, 400
	if entity.Id != uuidFromURL {
		ctx.JSON(http.StatusTeapot, "ID in url and ID in body doesn't match!")
		slog.Info("updateEntity in controller finished")
		return
	}

	//обновляем данные, если ошибка, то 500
	err = controller.service.UpdateEntity(ctx, model.Entity{Id: entity.Id, Test: model.Test{Name: entity.Test.Name, Age: entity.Test.Age}})
	if err != nil {
		if err.Error() == "record not found" {
			ctx.Status(http.StatusNotFound)
			slog.Info("updateEntity in controller finished")
			return
		}
		slog.ErrorContext(logger.WithLogError(ctx, err), "failed to update entity")
		ctx.JSON(http.StatusInternalServerError, err.Error())
		slog.Info("updateEntity in controller finished")
		return
	}

	//если дошли до сюда, значит все обновилось, отправляем 200
	ctx.Status(http.StatusOK)
	slog.Info("updateEntity in controller finished")
}

// deleteEntity удаляет model.Entity из приложения
func (controller *entityController) deleteEntity(ctx *gin.Context) {

	slog.Info("deleteEntity in controller started")

	//проверка uuid, переданного в урле, если не uuid, то 400
	uuidFromURL, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err.Error())
		slog.Info("deleteEntity in controller finished")
	}

	//удаление entity из базы, если ошибка, то 500
	if err = controller.service.DeleteEntity(ctx, uuidFromURL); err != nil {
		slog.ErrorContext(logger.WithLogError(ctx, err), "failed to update entity")
		ctx.JSON(http.StatusInternalServerError, err.Error())
		slog.Info("deleteEntity in controller finished")
		return
	}

	ctx.Status(http.StatusOK)
	slog.Info("deleteEntity in controller finished")
}
