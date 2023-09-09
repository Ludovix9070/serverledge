package container

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/grussorusso/serverledge/internal/config"
	"io"
	"io/ioutil"
	"log"
	"strings"
	//	"github.com/docker/docker/pkg/stdcopy"
)

type DockerFactory struct {
	cli *client.Client
	ctx context.Context
}

func InitDockerContainerFactory() *DockerFactory {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	dockerFact := &DockerFactory{cli, ctx}
	cf = dockerFact
	return dockerFact
}

func (cf *DockerFactory) Create(image string, opts *ContainerOptions) (ContainerID, error) {
	if !cf.HasImage(image) {
		log.Printf("Pulling image: %s", image)
		pullResp, err := cf.cli.ImagePull(cf.ctx, image, types.ImagePullOptions{})
		if err != nil {
			log.Printf("Could not pull image: %s", image)
			// we do not return here, as a stale copy of the image
			// could still be available locally
		} else {
			defer pullResp.Close()
			// This seems to be necessary to wait for the image to be pulled:
			io.Copy(ioutil.Discard, pullResp)
			log.Printf("Pulled image: %s", image)
			refreshedImages[image] = true
		}
	} else {
		log.Printf("We already have the image: %s", image)
	}

	contResources := container.Resources{Memory: opts.MemoryMB * 1048576} // convert to bytes
	if opts.CPUQuota > 0.0 {
		contResources.CPUPeriod = 50000 // 50ms
		contResources.CPUQuota = (int64)(50000.0 * opts.CPUQuota)
	}

	resp, err := cf.cli.ContainerCreate(cf.ctx, &container.Config{
		Image: image,
		Cmd:   opts.Cmd,
		Env:   opts.Env,
		Tty:   false,
	}, &container.HostConfig{Resources: contResources}, nil, nil, "")

	id := resp.ID

	r, err := cf.cli.ContainerInspect(cf.ctx, id)
	log.Printf("Container %s has name %s", id, r.Name)
	// TODO: remove, only for debug
	//containers, err := cf.cli.ContainerList(cf.ctx, types.ContainerListOptions{})
	//if err != nil {
	//	panic(err)
	//}
	//
	//for _, c := range containers {
	//	fmt.Println(c)
	//} // end remove

	return id, err
}

func (cf *DockerFactory) CopyToContainer(contID ContainerID, content io.Reader, destPath string) error {
	return cf.cli.CopyToContainer(cf.ctx, contID, destPath, content, types.CopyToContainerOptions{})
}

func (cf *DockerFactory) Start(contID ContainerID) error {
	if err := cf.cli.ContainerStart(cf.ctx, contID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

func (cf *DockerFactory) Destroy(contID ContainerID) error {
	// force set to true causes running container to be killed (and then
	// removed)
	return cf.cli.ContainerRemove(cf.ctx, contID, types.ContainerRemoveOptions{Force: true})
}

func (cf *DockerFactory) HasImage(image string) bool {

	list, err := cf.cli.ImageList(context.TODO(), types.ImageListOptions{
		All:            false,
		Filters:        filters.Args{},
		SharedSize:     false,
		ContainerCount: false,
	})
	if err != nil {
		fmt.Printf("image list error: %v\n", err)
		return false
	}
	for _, summary := range list {
		if len(summary.RepoTags) > 0 && strings.HasPrefix(summary.RepoTags[0], image) {
			// We have the image, but we may need to refresh it
			if config.GetBool(config.FACTORY_REFRESH_IMAGES, false) {
				if refreshed, ok := refreshedImages[image]; !ok || !refreshed {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (cf *DockerFactory) GetIPAddress(contID ContainerID) (string, error) {
	contJson, err := cf.cli.ContainerInspect(cf.ctx, contID)
	if err != nil {
		return "", err
	}
	return contJson.NetworkSettings.IPAddress, nil
}

func (cf *DockerFactory) GetMemoryMB(contID ContainerID) (int64, error) {
	contJson, err := cf.cli.ContainerInspect(cf.ctx, contID)
	if err != nil {
		return -1, err
	}
	return contJson.HostConfig.Memory / 1048576, nil
}
