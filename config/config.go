package config

import (
	"os"
	"strconv"
)

var Kuber string
var KafkaHost string
var KafkaPort string
var KafkaTopic string
var KafkaLogTopic string
var PostgresHost string
var PostgresPort string
var PostgresDatabaseName string
var PostgresPassword string //храним с секрете
var PostgresUser string     //храним в секрете
var RedisHost string
var RedisPort string
var RedisPassword string
var LogstashPort string
var LogstashHost string
var ElasticsearchPort string
var ElasticsearchHost string
var KibanaHost string
var KibanaPort string
var WorkersCount string
var RepositoryRetries int
var KafkaEnabled bool
var GracefulShutdownTimeoutSec int
var LoggingLevel int
var LogSourceEnabled bool
var RedisDataExpirationSec int

//если приложение поднято в кубере, то будут использоваться переменные окружения из configmap и secrets

func SetConfig() {
	//todo надо тут прибраться
	Kuber = os.Getenv("kuber")

	if KafkaHost = os.Getenv("kafkaHost"); KafkaHost == "" {
		KafkaHost = "localhost"
	}
	if KafkaPort = os.Getenv("kafkaPort"); KafkaPort == "" {
		KafkaPort = "9092"
	}
	if KafkaTopic = os.Getenv("kafkaTopic"); KafkaTopic == "" {
		KafkaTopic = "json_topic"
	}
	if KafkaLogTopic = os.Getenv("kafkaLogTopic"); KafkaLogTopic == "" {
		KafkaLogTopic = "log_topic"
	}
	if PostgresHost = os.Getenv("postgresHost"); PostgresHost == "" {
		PostgresHost = "localhost"
	}
	if PostgresPort = os.Getenv("postgresPort"); PostgresPort == "" {
		PostgresPort = "5432"
	}
	if RedisHost = os.Getenv("redisHost"); RedisHost == "" {
		RedisHost = "localhost"
	}
	if RedisPort = os.Getenv("redisPort"); RedisPort == "" {
		RedisPort = "6379"
	}
	expiration, err := strconv.Atoi(os.Getenv("gracefulShutdownTimeoutSec"))
	if err != nil {
		RedisDataExpirationSec = 3600
	} else {
		GracefulShutdownTimeoutSec = expiration
	}
	if LogstashHost = os.Getenv("logstashHost"); LogstashHost == "" {
		LogstashHost = "logstash"
	}
	if LogstashPort = os.Getenv("logstashPort"); LogstashPort == "" {
		LogstashPort = "5044"
	}
	if ElasticsearchHost = os.Getenv("elasticsearchHost"); ElasticsearchHost == "" {
		ElasticsearchHost = "elasticsearch"
	}
	if ElasticsearchPort = os.Getenv("elasticsearchPort"); ElasticsearchPort == "" {
		ElasticsearchPort = "9200"
	}
	if KibanaHost = os.Getenv("kibanaHost"); KibanaHost == "" {
		KibanaHost = "localhost"
	}
	if KibanaPort = os.Getenv("kibanaPort"); KibanaPort == "" {
		KibanaPort = "5601"
	}
	if WorkersCount = os.Getenv("workersCount"); WorkersCount == "" {
		WorkersCount = "15"
	}
	retries, err := strconv.Atoi(os.Getenv("repositoryRetries"))
	if err != nil {
		RepositoryRetries = 3
	} else {
		RepositoryRetries = retries
	}
	timeout, err := strconv.Atoi(os.Getenv("gracefulShutdownTimeoutSec"))
	if err != nil {
		GracefulShutdownTimeoutSec = 30
	} else {
		GracefulShutdownTimeoutSec = timeout
	}
	LoggingLevelString := os.Getenv("loggingLevel")
	if LoggingLevelString == "info" {
		LoggingLevel = 0
	}
	if LoggingLevelString == "debug" {
		LoggingLevel = -4
	}
	if LoggingLevelString == "warn" {
		LoggingLevel = 4
	}
	if LoggingLevelString == "error" {
		LoggingLevel = 8
	}
	LogSourceEnabledString := os.Getenv("logSourceEnabled")
	if LogSourceEnabledString == "true" {
		LogSourceEnabled = true
	}
	if PostgresDatabaseName = os.Getenv("postgresDatabaseName"); PostgresDatabaseName == "" {
		PostgresDatabaseName = "postgres"
	}
	if PostgresPassword = os.Getenv("postgresPassword"); PostgresPassword == "" {
		PostgresPassword = "root"
	}
	if PostgresUser = os.Getenv("postgresUser"); PostgresUser == "" {
		PostgresUser = "postgres"
	}
	if RedisPassword = os.Getenv("redisPassword"); RedisPassword == "" {
		RedisPassword = ""
	}

}
