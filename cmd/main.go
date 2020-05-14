package main

import (
	"flag"
	"k8s.io/klog"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/lwolf/kube-cleanup-operator/pkg/controller"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // TODO: Add all auth providers
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	// kubernetes client-go uses klog, which logs to file by default. Change defaults to log to stderr instead of file.
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	logtostderr := klogFlags.Lookup("logtostderr")
	logtostderr.Value.Set("true")

	sigs := make(chan os.Signal, 1) // Create channel to receive OS signals
	stop := make(chan struct{})     // Create channel to receive stop signal

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigs channel to receieve SIGTERM

	wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on so that they finish

	runOutsideCluster := flag.Bool("run-outside-cluster", false, "Set this flag when running outside of the cluster.")
	namespace := flag.String("namespace", "", "Watch only this namespaces")
	keepSuccessHours := flag.Int("keep-successful", 0, "Number of hours to keep successful jobs, -1 - forever, 0 - never (default), >0 number of hours")
	keepFailedHours := flag.Int("keep-failures", -1, "Number of hours to keep faild jobs, -1 - forever (default) 0 - never, >0 number of hours")
	keepPendingHours := flag.Int("keep-pending", -1, "Number of hours to keep pending jobs, -1 - forever (default) >0 number of hours")
	dryRun := flag.Bool("dry-run", false, "Print only, do not delete anything.")
	flag.Parse()

	// Create clientset for interacting with the kubernetes cluster
	clientset, err := newClientSet(*runOutsideCluster)

	if err != nil {
		log.Fatal(err.Error())
	}

	options := map[string]float64{
		"keepSuccessHours": float64(*keepSuccessHours),
		"keepFailedHours":  float64(*keepFailedHours),
		"keepPendingHours": float64(*keepPendingHours),
	}
	if *dryRun {
		log.Println("Performing dry run...")
	}
	log.Printf(
		"Provided settings: namespace=%s, dryRun=%t, keepSuccessHours: %d, keepFailedHours: %d, keepPendingHours: %d",
		*namespace, *dryRun, *keepSuccessHours, *keepFailedHours, *keepPendingHours,
	)

	wg.Add(1)
	go func() {
		controller.NewPodController(clientset, *namespace, *dryRun, options).Run(stop)
		wg.Done()
	}()
	log.Printf("Controller started...")

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
