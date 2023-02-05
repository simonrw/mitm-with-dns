package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "docker").Logger()
}

type dockerClient struct {
	cli *client.Client
}

func copyFile(src, dst string) error {
	fin, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer fin.Close()

	fout, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer fout.Close()

	if _, err = io.Copy(fout, fin); err != nil {
		return fmt.Errorf("copying file contents: %w", err)
	}
	return nil
}

func (c dockerClient) buildImage(ctx context.Context, name, base string) error {
	logger.Debug().Str("name", name).Str("base", base).Msg("building image")
	dockerfileContents := fmt.Sprintf(`
	FROM %s
	COPY init /init
	RUN chmod +x /init
	ENTRYPOINT ["/init"]
	`, base)

	contextDir, err := os.MkdirTemp("", "dockerbuild-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}

	filename := filepath.Join(contextDir, "Dockerfile")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer file.Close()

	// create the dockerfile
	_, err = file.WriteString(dockerfileContents)
	if err != nil {
		return fmt.Errorf("writing dockerfile contents: %w", err)
	}

	logger.Debug().Str("filename", file.Name()).Msg("")

	// copy the init biext: %w", err)nary
	if err := copyFile("./init", filepath.Join(contextDir, "init")); err != nil {
		return fmt.Errorf("copying init binary into context: %w", err)
	}

	// tar up the docker build into a context
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		// ExcludePatterns: ...
		ChownOpts: &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return fmt.Errorf("creating build context: %w", err)
	}

	res, err := c.cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
		Tags:       []string{name},
		PullParent: true,
		Dockerfile: "Dockerfile",
	})
	if err != nil {
		return fmt.Errorf("building image: %w", err)
	}
	defer res.Body.Close()

	io.Copy(os.Stdout, res.Body)

	return nil
}

func (c dockerClient) Close() {
	c.cli.Close()
}

func Run(stop chan struct{}, complete chan struct{}) {
	logger.Info().Msg("running docker container")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create docker client")
	}
	client := dockerClient{
		cli: cli,
	}
	defer client.Close()

	if err := client.buildImage(ctx, "foo", "alpine"); err != nil {
		logger.Fatal().Err(err).Msg("building image")
	}

	res, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "foo",
	}, nil, nil, nil, "")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create container")
	}
	if err := cli.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		logger.Fatal().Err(err).Msg("could not start container")
	}

	statusCh, errCh := cli.ContainerWait(ctx, res.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logger.Fatal().Err(err).Msg("error running container")
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, res.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logger.Fatal().Err(err).Msg("could not get container logs")
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	logger.Info().Msg("container run complete")

	logger.Info().Msg("waiting for shutdown signal")
	<-stop
	logger.Info().Msg("shutting down docker container")
	complete <- struct{}{}
}
