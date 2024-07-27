package listener

import (
	"context"
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
}

func GetEntityListener() EntityListener {
	if entityListenerInstance != nil {
		return entityListenerInstance
	}

	//определяю адрес кафки
	kafkaAddress := fmt.Sprintf("%s:%s", config.KafkaHost, config.KafkaPort)

	//Создаю объект для чтения сообщений из kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaAddress}, // здесь адрес подключения к кафке
		Topic:   config.KafkaTopic,
		//GroupID:  "json_consumer_group",
		//MinBytes: 10e3, // 10KB
		//MaxBytes: 10e6, // 10MB
	})

	//инициализируем инсстанс листенера
	entityRepository := repository.GetEntityRepository()
	entityListenerInstance := &entityListener{reader, entityRepository, false}

	//передаем функцию закрытия в клозер для graceful shutdown
	closer.CloseFunctions = append(closer.CloseFunctions, entityListenerInstance.CloseEntityListener())
	return entityListenerInstance
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
		if err != nil {
			log.Println("failed to read message:", err)
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
		listener.repository.SaveEntity(entity)
	}
}
