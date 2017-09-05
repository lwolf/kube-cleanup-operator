package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/lwolf/kube-cleanup-operator/pkg/controller"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/labels"
	"fmt"
)

func main() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
	stop := make(chan struct{})     // Create channel to receive stop signal

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on so that they finish

	runOutsideCluster := flag.Bool("run-outside-cluster", false, "Set this flag when running outside of the cluster.")
	labelSelectorString := flag.String("label", "", "Watch only jobs with this label")
	namespace := flag.String("namespace", "", "Watch only this namespaces")
	mode := flag.String("mode", "", "Change working mode: keep all or delete all. Default is delete all")
	flag.Parse()

	// Create clientset for interacting with the kubernetes cluster
	clientset, err := newClientSet(*runOutsideCluster)

	if err != nil {
		panic(err.Error())
	}
	var labelSelector labels.Selector
	labelSelector, err = labels.Parse(*labelSelectorString)
	if err != nil {
		panic(fmt.Errorf("invalid selector %q: %v", *labelSelectorString, err))
	}

	options := map[string]string{
		"labelSelector": labelSelector.String(),
		"mode":          *mode,
		"namespace":     *namespace,
	}

	fmt.Println("Configured namespace: ", options["namespace"])
	fmt.Println("Configured labelSelector: ", options["labelSelector"])
	fmt.Println("Configured mode: ", options["mode"])
	fmt.Println("Starting controller...")

	go controller.NewPodController(clientset, options).Run(stop, wg)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	log.Printf("Shutting down...")

	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped
}

func newClientSet(runOutsideCluster bool) (*kubernetes.Clientset, error) {
	kubeConfigLocation := ""

	if runOutsideCluster == true {
		if os.Getenv("KUBECONFIG") != "" {
			kubeConfigLocation = filepath.Join(os.Getenv("KUBECONFIG"))
		} else {
			homeDir := os.Getenv("HOME")
			kubeConfigLocation = filepath.Join(homeDir, ".kube", "config")
		}
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigLocation)

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
