package container

import (
	"context"
	"customers_kuber/closer"
	"customers_kuber/config"
	"customers_kuber/logger"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"log/slog"
	"time"
)

func RunKibana() error {

	elasticsearchAddress := "http://" + config.ElasticsearchHost + ":" + config.ElasticsearchPort
	ctx := context.Background()
	kibanaReq := testcontainers.ContainerRequest{
		Name:         "kibana",
		Image:        "kibana:8.15.0",
		ExposedPorts: []string{config.KibanaPort + "/tcp"},
		Env: map[string]string{
			"ELASTICSEARCH_HOSTS": elasticsearchAddress,
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.NetworkMode = "NET"
			hostConfig.PortBindings = nat.PortMap{
				"5601/tcp": []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: config.KibanaPort},
				}}
		},
	}

	//запуск контейнера
	kibanaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: kibanaReq,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start kibana container: %s", err)
	}

	//передача функции в closer для graceful shutdown
	closer.CloseFunctions = append(closer.CloseFunctions, func() {
		if err = kibanaContainer.Terminate(ctx); err != nil {
			ctx = logger.WithLogError(ctx, err)
			slog.ErrorContext(ctx, "failed to terminate kibana container")
			return
		}
		slog.Info("kibana container terminated successfully")
	})
	time.Sleep(time.Second * 3)
	return nil
}
