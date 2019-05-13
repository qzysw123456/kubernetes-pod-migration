package agent

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"
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
	mux.HandleFunc("/healthCheck", s.HealthCheck)
	mux.HandleFunc("/migratePod", s.migratePod)
	mux.HandleFunc("/clear", s.clear)
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
	w.Write([]byte(hostName))
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

	user := os.Getenv("USER")

	err = cli.CheckpointCreate(ctx, containerId, types.CheckpointCreateOptions{"savedState", "/home/" + user + "/checkpoint", false})

	if err != nil {
		panic(err)
	}
	w.Write([]byte("checkpointed " + destHost + "\n"))

	cmd := exec.Command("sudo", "scp", "-r", "/home/" + user + "/checkpoint", user + "@" + destHost + ":/home/" + user)
	cmd.Run()
}

func (s *Server) clear(w http.ResponseWriter, req *http.Request) {
	user := os.Getenv("USER")
	cmd := exec.Command("sudo", "rm", "-rf", "/home/" + user + "/checkpoint")
	cmd.Run()
	cmd = exec.Command("sudo", "rm", "/home/" + user + "/indeed")
	cmd.Run()
}