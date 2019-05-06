package plugin

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"os/exec"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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


	hostIP := pod.Status.HostIP
	url := "http://" + hostIP + ":10027/migratePod"

	body := strings.NewReader("containerId=" + strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "docker://") + "&" + "destHost=" + a.DestHost)
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

	fmt.Println(resp.Body)


	err = clientset.CoreV1().Pods(a.Namespace).Delete(a.PodName, &metav1.DeleteOptions{})
	if err != nil {
		fmt.Println("delete error")
	}

	for ; err == nil; _, err = clientset.CoreV1().Pods("default").Get(pod.Name, metav1.GetOptions{}) {
		time.Sleep(1 * time.Second)
	}

	newPod := &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{"Pod", "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: pod.ObjectMeta.Name},
	}

	newPod.Spec = apiv1.PodSpec{
		Containers: make([]apiv1.Container, len(pod.Spec.Containers)),
	}

	for i := 0; i < len(pod.Spec.Containers); i++ {
		newPod.Spec.Containers[i].Name = pod.Spec.Containers[i].Name
		newPod.Spec.Containers[i].Image = pod.Spec.Containers[i].Image
		newPod.Spec.Containers[i].Command = pod.Spec.Containers[i].Command
	}

	newPod.Spec.NodeSelector = make(map[string]string)
	newPod.Spec.NodeSelector["kubernetes.io/hostname"] = a.DestHost

	cmd := exec.Command("sudo", "rm", "/home/qzy/indeed")
	cmd.Run()

	_, err = clientset.CoreV1().Pods(a.Namespace).Create(newPod)
	if err != nil {
		fmt.Println("create error")
	}



	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

