package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/aalubin/kube-cleanup-operator/pkg/controller"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"strconv"
)

func main() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
	stop := make(chan struct{})     // Create channel to receive stop signal

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on so that they finish

	runOutsideCluster := flag.Bool("run-outside-cluster", false, "Set this flag when running outside of the cluster.")
	namespace := flag.String("namespace", "", "Watch only this namespaces")
	keepSuccessDays := flag.Int("keep-successful", -1, "Number of days to keep successful jobs, -1 - forever (default), 0 - never, >0 number of days")
	keepFailedDays := flag.Int("keep-failures", 0, "Number of days to keep faild jobs, -1 - forever 0 - never (default), >0 number of days")
	flag.Parse()

	// Create clientset for interacting with the kubernetes cluster
	clientset, err := newClientSet(*runOutsideCluster)

	if err != nil {
		panic(err.Error())
	}

	options := map[string]string{
		"namespace": *namespace,
		"keepSuccessDays": strconv.Itoa(*keepSuccessDays),
		"keepFailedDays": strconv.Itoa(*keepFailedDays),
	}

	log.Printf("Configured namespace: '%s', keepSuccessDays: %d, keepFailedDays: %d", options["namespace"], *keepSuccessDays, *keepFailedDays)
	log.Printf("Starting controller...")

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

