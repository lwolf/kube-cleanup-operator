package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // TODO: Add all auth providers
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/lwolf/kube-cleanup-operator/pkg/controller"
)

func setupLogging() {
	// Set logging output to standard console out
	log.SetOutput(os.Stdout)

	// kubernetes client-go uses klog, which logs to file by default. Change defaults to log to stderr instead of file.
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	logtostderr := klogFlags.Lookup("logtostderr")
	logtostderr.Value.Set("true")
}

func main() {
	runOutsideCluster := flag.Bool("run-outside-cluster", false, "Set this flag when running outside of the cluster.")
	namespace := flag.String("namespace", "", "Limit scope to a single namespaces")

	deleteOrphanedAfter := flag.Duration("delete-orphaned-after", 1*time.Hour, "Delete orphaned pods. Pods without an owner in non-running state (golang duration format, e.g 5m), 0 - never delete")
	deleteSuccessAfter := flag.Duration("delete-successful-after", 15*time.Minute, "Delete jobs in successful state after X duration (golang duration format, e.g 5m), 0 - never delete")
	deleteFailedAfter := flag.Duration("delete-failed-after", 0, "Delete jobs in failed state after X duration (golang duration format, e.g 5m), 0 - never delete")
	deletePendingAfter := flag.Duration("delete-pending-after", 0, "Delete jobs in pending state after X duration (golang duration format, e.g 5m), 0 - never delete")

	legacyKeepSuccessHours := flag.Int64("keep-successful", 0, "Number of hours to keep successful jobs, -1 - forever, 0 - never (default), >0 number of hours")
	legacyKeepFailedHours := flag.Int64("keep-failures", -1, "Number of hours to keep faild jobs, -1 - forever (default) 0 - never, >0 number of hours")
	legacyKeepPendingHours := flag.Int64("keep-pending", -1, "Number of hours to keep pending jobs, -1 - forever (default) >0 number of hours")
	legacyMode := flag.Bool("legacy-mode", true, "Legacy mode: `true` - use old `keep-*` flags, `false` - enable new `delete-*-after` flags")

	dryRun := flag.Bool("dry-run", false, "Print only, do not delete anything.")
	flag.Parse()
	setupLogging()

	log.Println("Starting the application.")
	log.Printf(
		"Provided options: \n\t namespace: %s\n\t dry-run: %t\n\t delete-successful-after: %v\n\t delete-failed-after: %v\n\t delete-pending-after: %v\n\t delete-orphaned-after: %v\n",
		*namespace, *dryRun, *deleteSuccessAfter, *deleteFailedAfter, *deletePendingAfter, deleteOrphanedAfter,
	)

	var warning strings.Builder
	warning.WriteString("\n!!! DEPRECATION WARNING !!!\n")
	warning.WriteString("\t`keep-successful` is deprecated, use `delete-successful-after` instead\n")
	warning.WriteString("\t`keep-failures` is deprecated, use `delete-failed-after` instead\n")
	warning.WriteString("\t`keep-pending` is deprecated, use `delete-pending-after` instead\n")
	warning.WriteString("\tThese fields are going to be removed in the next version\n")
	fmt.Printf(warning.String())

	sigsCh := make(chan os.Signal, 1) // Create channel to receive OS signals
	stopCh := make(chan struct{})     // Create channel to receive stopCh signal

	signal.Notify(sigsCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigsCh channel to receieve SIGTERM

	wg := &sync.WaitGroup{}

	// Create clientset for interacting with the kubernetes cluster
	clientset, err := newClientSet(*runOutsideCluster)
	if err != nil {
		log.Fatal(err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		if *legacyMode {
			controller.NewPodController(
				ctx,
				clientset,
				*namespace,
				*dryRun,
				*legacyKeepSuccessHours,
				*legacyKeepFailedHours,
				*legacyKeepPendingHours,
			)
		} else {
			controller.NewKleaner(
				ctx,
				clientset,
				*namespace,
				*dryRun,
				*deleteSuccessAfter,
				*deleteFailedAfter,
				*deletePendingAfter,
				*deleteOrphanedAfter,
			).Run(stopCh)
		}
		wg.Done()
	}()
	log.Printf("Controller started...")

	<-sigsCh // Wait for signals (this hangs until a signal arrives)
	log.Printf("Shutting down...")
	cancel()
	close(stopCh) // Tell goroutines to stopCh themselves
	wg.Wait()     // Wait for all to be stopped
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
