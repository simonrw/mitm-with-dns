package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/stdcopy"
	"go.uber.org/zap"
)

type dockerClient struct {
	ipAddresses []net.IP
	cli         *client.Client
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

func copyFiles(srcs, dsts []string) error {
	if len(srcs) != len(dsts) {
		panic("mismatch between sources and destinations")
	}
	n := len(srcs)
	for i := 0; i < n; i++ {
		err := copyFile(srcs[i], dsts[i])
		if err != nil {
			return fmt.Errorf("copying %s to %s: %w", srcs[i], dsts[i], err)
		}
	}
	return nil
}

func (c dockerClient) buildNewCommand(ctx context.Context, image string) ([]string, error) {

	res, _, err := c.cli.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("inspecting image %s: %w", image, err)
	}

	cfg := res.ContainerConfig
	oldEntrypoint := cfg.Entrypoint
	oldCmd := cfg.Cmd

	return append(oldEntrypoint, oldCmd...), nil
}

func (c dockerClient) buildImage(ctx context.Context, name, base string) error {
	logger.Debugw("building image", "name", name, "base", base)
	dockerfileContents := fmt.Sprintf(`
	FROM %s
	COPY init /init
	RUN chmod +x /init
	RUN mkdir -p /customcerts/ca
	COPY *.pem /customcerts/
	COPY ca/*.pem /customcerts/ca/
	`, base)

	contextDir, err := os.MkdirTemp("", "dockerbuild-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}

	if err := os.Mkdir(filepath.Join(contextDir, "ca"), 0o777); err != nil {
		return fmt.Errorf("creating ca dir: %w", err)
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

	logger.Debugw("", "filename", file.Name())

	// copy the init binary
	if err := copyFile("./init", filepath.Join(contextDir, "init")); err != nil {
		return fmt.Errorf("copying init binary into context: %w", err)
	}

	// copy the certificates
	if err := copyFiles([]string{
		"./_wildcard.amazonaws.com+1-key.pem",
		"./_wildcard.amazonaws.com+1.pem",
		"./ca/rootCA-key.pem",
		"./ca/rootCA.pem",
	}, []string{
		filepath.Join(contextDir, "_wildcard.amazonaws.com+1-key.pem"),
		filepath.Join(contextDir, "_wildcard.amazonaws.com+1.pem"),
		filepath.Join(contextDir, "ca", "rootCA-key.pem"),
		filepath.Join(contextDir, "ca", "rootCA.pem"),
	}); err != nil {
		return err
	}

	// tar up the docker build into a context
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		// ExcludePatterns: ...
		ChownOpts: &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return fmt.Errorf("creating build context: %w", err)
	}

	_, err = c.cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
		Tags:       []string{name},
		Dockerfile: "Dockerfile",
	})
	if err != nil {
		return fmt.Errorf("building image: %w", err)
	}

	return nil
}

func (c dockerClient) runContainer(ctx context.Context, image, name string, stop chan struct{}) error {
	is := []string{}
	for _, addr := range c.ipAddresses {
		if addr.IsLoopback() {
			continue
		}
		is = append(is, addr.String())
	}
	hostCfg := &container.HostConfig{
		DNS: is,
	}
	res, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:      image,
		Entrypoint: []string{"/init", "--"},
	}, hostCfg, nil, nil, name)
	if err != nil {
		logger.Fatalw("could not create container", "err", err)
	}
	logger := logger.With("container-id", res.ID)

	if err := c.cli.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		logger.Fatalw("could not start container", "err", err)
	}
	logger.Debug("started container")

	containerRemove := func() {
		logger.Debug("stopping container")
		timeout := 1
		if err := c.cli.ContainerStop(ctx, res.ID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			logger.Warnw("failed to stop container", "err", err)
		}

		logger.Info("removing container")
		// remove the container
		if err := c.cli.ContainerRemove(ctx, res.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			logger.Warnf("failed to remove container", "err", err)
		}
	}

	logger.Debug("waiting for container to finish")
	statusCh, errCh := c.cli.ContainerWait(ctx, res.ID, container.WaitConditionNotRunning)
	logger.Debug("wait call finished")
	select {
	case <-stop:
		logger.Debug("stop command received")
		out, err := c.cli.ContainerLogs(ctx, res.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			logger.Warnf("could not get container logs", "err", err)
		} else {
			stdcopy.StdCopy(os.Stdout, os.Stderr, out)
		}
		containerRemove()
		return nil
	case err := <-errCh:
		logger.Warnf("error received from container", "err", err)
		if err != nil {
			containerRemove()
			logger.Fatalw("error running container", "err", err)
		}
	case <-statusCh:
		logger.Debug("conatiner stopped by itself")
		out, err := c.cli.ContainerLogs(ctx, res.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			logger.Warnf("could not get container logs", "err", err)
		} else {
			stdcopy.StdCopy(os.Stdout, os.Stderr, out)
		}
		containerRemove()
	}

	logger.Info("container run complete")
	return nil
}

func (c dockerClient) Close() {
	c.cli.Close()
}

var logger *zap.SugaredLogger

func Run(l *zap.SugaredLogger, baseName string, ipAddresses []net.IP, stop chan struct{}, complete *sync.WaitGroup, earlyExit chan<- struct{}) {
	logger = l
	complete.Add(1)
	defer complete.Done()
	logger.Info("running docker container")

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Fatalw("could not create docker client", "err", err)
	}
	client := dockerClient{
		ipAddresses: ipAddresses,
		cli:         cli,
	}
	defer client.Close()

	imageName := "foo"
	containerName := "container"
	if err := client.buildImage(ctx, imageName, baseName); err != nil {
		logger.Fatalf("building image", "err", err)
	}
	if err := client.runContainer(ctx, imageName, containerName, stop); err != nil {
		logger.Fatalw("running container", "err", err)
	}

	logger.Info("docker process finished")
	earlyExit <- struct{}{}
}
