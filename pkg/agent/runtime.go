package agent

import (
	dockerclient "github.com/docker/docker/client"
	"time"
)

// RuntimeManager is responsible for docker operation

type RuntimeManager struct {
	client  *dockerclient.Client
	timeout time.Duration
}

func NewRuntimeManager(host string, timeout time.Duration) (*RuntimeManager, error) {
	client, err := dockerclient.NewClientWithOpts(dockerclient.WithVersion("1.39"))
	if err != nil {
		return nil, err
	}
	return &RuntimeManager{
		client:  client,
		timeout: timeout,
	}, nil
}