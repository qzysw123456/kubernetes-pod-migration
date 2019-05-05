package plugin

import (
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"strings"

	"flag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
)


const (
	example = `
	# Checkpoint a Pod
	kubectl migrate POD_NAME destHost
	kubectl migrate POD_NAME --namespace string destHost
`
	longDesc = `
migrate POD_NAME to destHost
`
)

type MigrateArgs struct {

	// Pod select options
	Namespace string
	PodName   string
	DestHost string
}


func NewPluginCmd() *cobra.Command {
	var Margs MigrateArgs
	cmd := &cobra.Command{
		Use: "migrate [OPTIONS] POD_NAME destHost",
		Short:   "migrate a Pod",
		Long:    longDesc,
		Example:	example,
		Run: func(c *cobra.Command, args []string) {
			if err := Margs.Complete(c, args); err != nil {
				fmt.Println(err)
			}
			/*
			if err := opts.Validate(); err != nil {
				fmt.Println(err)
			}
			if err := opts.Run(); err != nil {
				fmt.Println(err)
			}
			*/
			if err := Margs.Run(); err != nil {
				fmt.Println(err)
			}
		},
	}
	cmd.Flags().StringVar(&Margs.Namespace, "namespace", "default",
		"default namespace is \"default\"")
	return cmd
}

func (a * MigrateArgs) Complete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("error pod not specified")
	}
	if len(args) == 1 {
		return fmt.Errorf("destHost not specified")
	}

	a.PodName = args[0]
	a.DestHost = args[1]
	return nil
}



func (a * MigrateArgs) Run() error {
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

	var errVal error
	pod, err := clientset.CoreV1().Pods(a.Namespace).Get(a.PodName, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		fmt.Printf("Pod %s in namespace %s not found\n", a.PodName, a.Namespace)
		return errVal
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %s in namespace %s: %v\n",
			a.PodName, a.Namespace, statusError.ErrStatus.Message)
		return errVal
	} else if err != nil {
		panic(err.Error())
		return errVal
	}

	fmt.Printf("Found pod %s in namespace %s\n", a.PodName, a.Namespace)
	fmt.Println(pod.Status.ContainerStatuses[0].ContainerID)

	hostIP := pod.Status.HostIP
	url := hostIP + ":10027/migratePod"
	fmt.Println(url)
	fmt.Println(pod.Status.ContainerStatuses[0].ContainerID)
	fmt.Println(a.DestHost)

	body := strings.NewReader("containerId=" + pod.Status.ContainerStatuses[0].ContainerID + "&" + "destHost=" + a.DestHost)
	req, err := http.NewRequest("POST", url, body)

	if err != nil {
		// handle err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

