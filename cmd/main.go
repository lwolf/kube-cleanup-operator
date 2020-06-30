package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/VictoriaMetrics/metrics"
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
	listenAddr := flag.String("listen-addr", "0.0.0.0:7000", "Address to expose metrics.")

	deleteSuccessAfter := flag.Duration("delete-successful-after", 15*time.Minute, "Delete jobs and pods in successful state after X duration (golang duration format, e.g 5m), 0 - never delete")
	deleteFailedAfter := flag.Duration("delete-failed-after", 0, "Delete jobs and pods in failed state after X duration (golang duration format, e.g 5m), 0 - never delete")
	deleteOrphanedAfter := flag.Duration("delete-orphaned-pods-after", 1*time.Hour, "Delete orphaned pods. Pods without an owner in non-running state (golang duration format, e.g 5m), 0 - never delete")
	deleteEvictedAfter := flag.Duration("delete-evicted-pods-after", 15*time.Minute, "Delete pods in evicted state (golang duration format, e.g 5m), 0 - never delete")
	deletePendingAfter := flag.Duration("delete-pending-pods-after", 0, "Delete pods in pending state after X duration (golang duration format, e.g 5m), 0 - never delete")

	legacyKeepSuccessHours := flag.Int64("keep-successful", 0, "Number of hours to keep successful jobs, -1 - forever, 0 - never (default), >0 number of hours")
	legacyKeepFailedHours := flag.Int64("keep-failures", -1, "Number of hours to keep faild jobs, -1 - forever (default) 0 - never, >0 number of hours")
	legacyKeepPendingHours := flag.Int64("keep-pending", -1, "Number of hours to keep pending jobs, -1 - forever (default) >0 number of hours")
	legacyMode := flag.Bool("legacy-mode", true, "Legacy mode: `true` - use old `keep-*` flags, `false` - enable new `delete-*-after` flags")

	dryRun := flag.Bool("dry-run", false, "Print only, do not delete anything.")
	flag.Parse()
	setupLogging()

	log.Println("Starting the application.")
	var optsInfo strings.Builder
	optsInfo.WriteString("Provided options: \n")
	optsInfo.WriteString(fmt.Sprintf("\tnamespace: %s\n", *namespace))
	optsInfo.WriteString(fmt.Sprintf("\tdry-run: %v\n", *dryRun))
	optsInfo.WriteString(fmt.Sprintf("\tdelete-successful-after: %s\n", *deleteSuccessAfter))
	optsInfo.WriteString(fmt.Sprintf("\tdelete-failed-after: %s\n", *deleteFailedAfter))
	optsInfo.WriteString(fmt.Sprintf("\tdelete-pending-after: %s\n", *deletePendingAfter))
	optsInfo.WriteString(fmt.Sprintf("\tdelete-orphaned-after: %s\n", *deleteOrphanedAfter))
	optsInfo.WriteString(fmt.Sprintf("\tdelete-evicted-after: %s\n", *deleteEvictedAfter))

	optsInfo.WriteString(fmt.Sprintf("\n\tlegacy-mode: %v\n", *legacyMode))
	optsInfo.WriteString(fmt.Sprintf("\tkeep-successful: %d\n", *legacyKeepSuccessHours))
	optsInfo.WriteString(fmt.Sprintf("\tkeep-failures: %d\n", *legacyKeepFailedHours))
	optsInfo.WriteString(fmt.Sprintf("\tkeep-pending: %d\n", *legacyKeepPendingHours))
	log.Println(optsInfo.String())

	if *legacyMode {
		var warning strings.Builder
		warning.WriteString("\n!!! DEPRECATION WARNING !!!\n")
		warning.WriteString("\t Operator is running in `legacy` mode. Using old format of arguments. Please change the settings.\n")
		warning.WriteString("\t`keep-successful` is deprecated, use `delete-successful-after` instead\n")
		warning.WriteString("\t`keep-failures` is deprecated, use `delete-failed-after` instead\n")
		warning.WriteString("\t`keep-pending` is deprecated, use `delete-pending-after` instead\n")
		warning.WriteString(" These fields are going to be removed in the next version\n")
		log.Println(warning.String())
	}

	sigsCh := make(chan os.Signal, 1) // Create channel to receive OS signals
	stopCh := make(chan struct{})     // Create channel to receive stopCh signal

	signal.Notify(sigsCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Register the sigsCh channel to receieve SIGTERM

	wg := &sync.WaitGroup{}

	// Create clientset for interacting with the kubernetes cluster
	clientset, err := newClientSet(*runOutsideCluster)
	if err != nil {
		log.Fatal(err.Error())
	}
	ctx := context.Background()

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
				stopCh,
			).Run()
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
				*deleteEvictedAfter,
				stopCh,
			).Run()
		}
		wg.Done()
	}()
	log.Printf("Controller started...")

	server := http.Server{Addr: *listenAddr}
	wg.Add(1)
	go func() {
		// Expose the registered metrics at `/metrics` path.
		http.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
			metrics.WritePrometheus(w, true)
		})
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("failed to ListenAndServe metrics server: %v\n", err)
		}
		wg.Done()
	}()
	log.Printf("Listening at %s", *listenAddr)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			<-stopCh
			log.Println("shutting http server down")
			err := server.Shutdown(ctx)
			if err != nil {
				log.Printf("failed to shutdown metrics server: %v\n", err)
			}
			break
		}
	}()

	<-sigsCh // Wait for signals (this hangs until a signal arrives)
	log.Printf("got termination signal...")
	close(stopCh) // Tell goroutines to stopCh themselves
	wg.Wait()     // Wait for all to be stopped
}

func newClientSet(runOutsideCluster bool) (*kubernetes.Clientset, error) {
	kubeConfigLocation := ""

	if runOutsideCluster {
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
