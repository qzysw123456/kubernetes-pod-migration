package agent

import (
	"context"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

	mux.HandleFunc("/create", s.Create)
	mux.HandleFunc("/checkpoint", s.Checkpoint)
	mux.HandleFunc("/healthCheck", s.HealthCheck)
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
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	desiredPod := req.FormValue("pod")
	desiredNamespace := req.FormValue("namespace")
	if len(desiredNamespace) == 0 {
		desiredNamespace = "default"
	}
	desiredHost := req.FormValue("desHost")

	var errVal error
	pod, err := clientset.CoreV1().Pods(desiredNamespace).Get(desiredPod, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		fmt.Printf("Pod %s in namespace %s not found\n", desiredPod, desiredNamespace)
		//return errVal
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %s in namespace %s: %v\n",
			desiredPod, desiredNamespace, statusError.ErrStatus.Message)
		//return errVal
	} else if err != nil {
		panic(err.Error())
		//return errVal
	} else {
		fmt.Printf("Found pod %s in namespace %s\n", desiredPod, desiredNamespace)
		fmt.Println(pod.Status.ContainerStatuses[0].ContainerID)
		//hostIP := pod.Status.HostIP
	}
	thisHost, _ := os.Hostname()
	if errVal == nil {
		cmd := exec.Command("scp", "-r", thisHost + ":/home/qzy/testK8s2", desiredHost + ":/home/qzy/")
		cmd.Run()
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
