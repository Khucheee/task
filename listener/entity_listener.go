package listener

import (
	"context"
	"customers_kuber/cache"
	"customers_kuber/closer"
	"customers_kuber/config"
	"customers_kuber/model"
	"customers_kuber/repository"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

var entityListenerInstance *entityListener

type EntityListener interface {
	StartListening()
	CloseEntityListener() func()
}

type entityListener struct {
	reader     *kafka.Reader
	repository repository.EntityRepository
	stopSignal bool
	cache      cache.EntityCache
}

func GetEntityListener() (EntityListener, error) {

	//если сущность уже существует, возвращаем её
	if entityListenerInstance != nil {
		return entityListenerInstance, nil
	}

	//определяю адрес кафки
	kafkaAddress := fmt.Sprintf("%s:%s", config.KafkaHost, config.KafkaPort)

	//Создаю объект для чтения сообщений из kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaAddress}, // здесь адрес подключения к кафке
		Topic:   config.KafkaTopic,
	})

	cacheConnect, err := cache.GetEntityCache()
	if err != nil {
		log.Printf("failed to get cache in listener: %s", err)
	}

	//инициализируем инсстанс листенера
	entityListenerInstance := &entityListener{reader: reader, stopSignal: false, cache: cacheConnect}
	entityRepository, err := repository.GetEntityRepository()
	entityListenerInstance.repository = entityRepository
	if err != nil {
		return entityListenerInstance, fmt.Errorf("failed to initislize repository in listener: %s", err)
	}

	//передаем функцию закрытия в клозер для graceful shutdown
	closer.CloseFunctions = append(closer.CloseFunctions, entityListenerInstance.CloseEntityListener())
	return entityListenerInstance, nil
}

func (listener *entityListener) CloseEntityListener() func() {
	return func() {
		listener.stopSignal = true
		if err := listener.reader.Close(); err != nil {
			log.Println("failed to close listener:", err)
			return
		}
		log.Println("entityListener closed successfully")
	}
}

func (listener *entityListener) StartListening() {

	//вызываем раз в секунду ReadMessage чтобы забрать сообщение из топика
	for {
		if listener.stopSignal == true {
			break
		}
		time.Sleep(time.Second * 1)
		msg, err := listener.reader.ReadMessage(context.Background())
		if err != nil && err.Error() != "fetching message: EOF" {
			log.Println("failed to read message:", err)
			continue
		}

		//парсю полученную json в структуру Entity
		var entity model.Entity
		err = json.Unmarshal(msg.Value, &entity)
		if err != nil {
			log.Println("failed to deserialize message from kafka:", err)
			continue
		}
		log.Println("message from kafka received:", entity)

		//сохраняю Entity в базу
		for i := 1; i < 3; i++ {
			if err = listener.repository.SaveEntity(entity); err == nil {
				break
			}
			log.Printf("Failed to save entity in service: %s", err)
			log.Printf("Retry to save entity: %d", i)
		}

		listener.cache.ClearCache()
	}
}
