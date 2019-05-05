package agent

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"
	"fmt"
)

const (
	dockerContainerPrefix = "docker://"
)

type Server struct {
	config     *Config
	runtimeApi *RuntimeManager
}

func NewServer(config *Config) (*Server, error) {
	runtime, err := NewRuntimeManager(config.DockerEndpoint, config.DockerTimeout)
	if err != nil {
		return nil, err
	}
	return &Server{config: config, runtimeApi: runtime}, nil
}

func (s *Server) Run() error {

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	mux := http.NewServeMux()

	mux.HandleFunc("/create", s.Create)
	mux.HandleFunc("/checkpoint", s.Checkpoint)
	mux.HandleFunc("/healthCheck", s.HealthCheck)
	mux.HandleFunc("/migratePod", s.migratePod)
	server := &http.Server{Addr: s.config.ListenAddress, Handler: mux}

	go func() {
		log.Printf("Listening on %s \n", s.config.ListenAddress)

		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	<-stop

	log.Println("shutting done server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	return nil
}

func (s *Server) HealthCheck(w http.ResponseWriter, req *http.Request) {
	hostName, _ := os.Hostname()
	hostName = "I'm an agent running on " + hostName + "\n"
	hostName = hostName + req.FormValue("hehe")

	w.Write([]byte(hostName))
}

func (s *Server) Checkpoint(w http.ResponseWriter, req *http.Request) {
	checkpointContainerWithName()
	w.Write([]byte("checkpointed"))
}

func (s *Server) Create(w http.ResponseWriter, req *http.Request) {
	createContainerWithName()
	w.Write([]byte("created"))
}

func createContainerWithName() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		panic(err)
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/busybox", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "busybox",
		///bin/sh -c 'i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done'
		Cmd:   []string{"/bin/sh", "-c", "i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done"},
	}, nil, nil, "cr")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, out)
}


func checkpointContainerWithName() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		panic(err)
	}

	err = cli.CheckpointCreate(ctx, "cr", types.CheckpointCreateOptions{"cr0", "", true})

	if err != nil {
		panic(err)
	}
}
func (s *Server) migratePod(w http.ResponseWriter, req *http.Request) {
	containerId := req.FormValue("containerId")
	destHost := req.FormValue("destHost")
	fmt.Println(containerId)
	fmt.Println(destHost)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		panic(err)
	}
	err = cli.CheckpointCreate(ctx, containerId, types.CheckpointCreateOptions{"savedState", "/home/qzy/checkpoint", false})
	if err != nil {
		panic(err)
	}
	w.Write([]byte("checkpointed " + destHost + "\n"))
	cmd := exec.Command("sudo", "scp", "-r", "/hone/qzy/checkpoint", "qzy@" + destHost + ":/home/qzy")
	cmd.Run()
}
