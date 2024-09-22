package cache

import (
	"context"
	"customers_kuber/closer"
	"customers_kuber/config"
	"customers_kuber/logger"
	"customers_kuber/model"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log/slog"
	"time"
)

var entityCacheInstance *entityCache

type entityCache struct {
	connect *redis.Client
	path    string
}

type EntityCache interface {
	ClearCache(ctx context.Context)
	UpdateCache(context.Context, []model.Entity)
	GetCache(ctx context.Context) []model.Entity
	SetPath(string)
}

func GetEntityCache() (EntityCache, error) {
	slog.Debug("func getEntityCache started")
	if entityCacheInstance != nil {
		slog.Debug("entityCache already exists, returning existing interface")
		return entityCacheInstance, nil
	}

	slog.Debug("entityCache not exists, creating new interface")
	ctx := context.Background()
	redisDBNumber := 0
	redisAddress := fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: config.RedisPassword,
		DB:       redisDBNumber,
	})
	slog.Debug("redis client created, redis address: " + redisAddress + " db number: " + string(redisDBNumber))
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		return entityCacheInstance, fmt.Errorf("redis ping was failed: %s", err)
	}
	slog.Debug("ping redis is successful")
	entityCacheInstance = &entityCache{connect: redisClient}
	slog.Debug("redis client handed over to entityCache")
	slog.Debug("handing over entityCache closer function to closer.CloseFunctions")
	closer.CloseFunctions = append(closer.CloseFunctions, func() {
		if err := entityCacheInstance.connect.Close(); err != nil {
			ctx = logger.WithLogError(ctx, err)
			slog.ErrorContext(ctx, "failed to close redis connection")
			return
		}
		slog.Debug("entityCache closed successfully")
	})
	slog.Debug("entityCache created successfully")
	return entityCacheInstance, nil
}

func (cache *entityCache) ClearCache(ctx context.Context) {
	status := cache.connect.FlushAll(ctx)
	if err := status.Err(); err != nil {
		ctx = logger.WithLogError(ctx, err)
		slog.WarnContext(ctx, "failed to clear cache")
		return
	}
	slog.Info("cache cleared successfully")
	return
}

func (cache *entityCache) UpdateCache(ctx context.Context, entitySlice []model.Entity) {
	slog.DebugContext(logger.WithLogValues(ctx, entitySlice), "update cache started")
	marshalledEntity, err := json.Marshal(entitySlice)
	if err != nil {
		logger.WithLogError(ctx, err)
		slog.WarnContext(ctx, "failed to marshal entity while saving to cache")
		return
	}
	slog.Debug("entitySlice in updateCache marshalled to JSON successfully", "Debug values", string(marshalledEntity))
	key := fmt.Sprintf("entity_%s", cache.path)
	redisDataExpiration := time.Second * time.Duration(config.RedisDataExpirationSec)
	ctx = logger.WithLogCacheKey(ctx, key)
	slog.DebugContext(ctx, "cache key in UpdateCache was set"+", redis data expiration time in seconds = "+
		string(redisDataExpiration/1000))
	data := cache.connect.Set(ctx, key, marshalledEntity, redisDataExpiration)
	if err = data.Err(); err != nil {
		logger.WithLogError(ctx, err)
		slog.WarnContext(ctx, "failed to save cache to redis")
		return
	}
	slog.Info("cache updated successfully")

}

func (cache *entityCache) GetCache(ctx context.Context) []model.Entity {

	slog.Debug("start getting cache from redis")
	var entities []model.Entity

	//забираем значения из кэша
	key := fmt.Sprintf("entity_%s", cache.path)
	ctx = logger.WithLogCacheKey(ctx, key)
	slog.DebugContext(ctx, "cache key in GetCache was set")

	//получаем кэш
	data := cache.connect.Get(ctx, key)
	if err := data.Err(); err != nil && err.Error() != "redis: nil" {
		ctx = logger.WithLogError(ctx, err)
		slog.WarnContext(ctx, "failed to get cache from redis")
		return nil
	}
	slog.Debug("cache taken from redis successfully")
	//если кэш пустой, возвращаем nil слайс
	if data.Val() == "" {
		slog.Info("cache is empty, returning nil slice to service")
		return entities
	}
	//если в кэше что-то лежит, парсим json в слайс, возвращаем его
	if err := json.Unmarshal([]byte(data.Val()), &entities); err != nil {
		ctx = logger.WithLogError(ctx, err)
		slog.WarnContext(ctx, "failed to unmarshal json taken from redis")
		return nil
	}
	slog.Info("cache is not empty returning values to service")
	return entities
}

func (cache *entityCache) SetPath(path string) {
	cache.path = path
	slog.Debug("cache path was set", "cache_path", path)
}
