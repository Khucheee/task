package listener

import (
	"context"
	"customers_kuber/cache"
	"customers_kuber/closer"
	"customers_kuber/config"
	"customers_kuber/logger"
	"customers_kuber/model"
	"customers_kuber/repository"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log/slog"
)

var entityListenerInstance *entityListener

type EntityListener interface {
	StartListening()
}

type entityListener struct {
	reader      *kafka.Reader
	repository  repository.EntityRepository
	stopSignal  bool
	stopChannel chan struct{}
	cache       cache.EntityCache
}

func GetEntityListener() (EntityListener, error) {

	//если сущность уже существует, возвращаем её
	if entityListenerInstance != nil {
		return entityListenerInstance, nil
	}

	ctx := context.Background()

	//определяю адрес кафки
	kafkaAddress := fmt.Sprintf("%s:%s", config.KafkaHost, config.KafkaPort)

	//Создаю объект для чтения сообщений из kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaAddress}, // здесь адрес подключения к кафке
		Topic:   config.KafkaTopic,
	})

	//получаю кэш
	cacheConnect, err := cache.GetEntityCache()
	if err != nil {
		slog.ErrorContext(logger.WithLogError(ctx, err), "failed to get cache, while creating entityListener")
	}

	//инициализируем инсстанс листенера
	entityListenerInstance = &entityListener{reader: reader, stopChannel: make(chan struct{}), cache: cacheConnect}
	entityRepository, err := repository.GetEntityRepository()
	entityListenerInstance.repository = entityRepository
	if err != nil {
		return entityListenerInstance, fmt.Errorf("failed to initislize repository in listener: %s", err)
	}

	//передаем функцию закрытия в клозер для graceful shutdown
	closer.CloseFunctions = append(closer.CloseFunctions, func() {
		close(entityListenerInstance.stopChannel)
		//entityListenerInstance.stopSignal = true
		if err := entityListenerInstance.reader.Close(); err != nil {
			slog.ErrorContext(logger.WithLogError(ctx, err), "failed to close listener")
			return
		}
		slog.Info("entityListener closed successfully")
	})
	return entityListenerInstance, nil
}

func (listener *entityListener) StartListening() {
	for {
		select {
		case <-listener.stopChannel:
			break
		default:
			ctx := context.Background()
			msg, err := listener.reader.ReadMessage(ctx)
			if err != nil {
				if err.Error() == "fetching message: EOF" {
					continue
				}
				slog.ErrorContext(logger.WithLogError(ctx, err), "failed to read message from kafka in listener")
				continue
			}

			//парсю полученную json в структуру Entity
			var entity model.Entity
			err = json.Unmarshal(msg.Value, &entity)
			if err != nil {
				slog.ErrorContext(logger.WithLogError(ctx, err), "failed to deserialize message from kafka in listener")
				continue
			}
			slog.Info("listener received message from kafka")
			listener.repository.SaveEntity(entity)
			listener.cache.ClearCache(ctx)
		}
	}
}
