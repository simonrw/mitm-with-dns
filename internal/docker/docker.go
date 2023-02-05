package docker

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "docker").Logger()
}

func Run(stop chan struct{}, complete chan struct{}) {
	logger.Info().Msg("running docker container")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Fatal().Msg("could not create docker client")
	}
	defer cli.Close()

	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	if err != nil {
		logger.Fatal().Msg("could not pull image")
	}
	io.Copy(os.Stdout, reader)

	res, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
	}, nil, nil, nil, "")
	if err != nil {
		logger.Fatal().Msg("could not create container")
	}
	if err := cli.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		logger.Fatal().Msg("could not start container")
	}

	statusCh, errCh := cli.ContainerWait(ctx, res.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logger.Fatal().Msg("error running container")
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, res.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logger.Fatal().Msg("could not get container logs")
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	logger.Info().Msg("container run complete")

	logger.Info().Msg("waiting for shutdown signal")
	<-stop
	logger.Info().Msg("shutting down docker container")
	complete <- struct{}{}
}
