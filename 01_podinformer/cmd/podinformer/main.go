// Building a simple custome controller for Kubernetes
// Based on the great work by https://github.com/aaronlevy/kube-controller-demo
package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/embano1/pairprogramming/01_podinformer/common"
	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

type podWatcher struct {
	kclient  kubernetes.Interface
	informer cache.SharedInformer
	stopCh   chan struct{}
	doneCh   chan struct{}
}

type pwopts struct {
	resync     time.Duration
	kubeconfig string
}

func main() {
	opts := &pwopts{}
	// When running as a pod in-cluster, a kubeconfig is not needed. Instead this will make use of the service account injected into the pod.
	// However, allow the use of a local kubeconfig as this can make local development & testing easier.
	flag.StringVar(&opts.kubeconfig, "kubeconfig", "", "Path to a kubeconfig file")
	flag.DurationVar(&opts.resync, "r", 10*time.Second, "How often to refresh the cache")

	// We log to stderr because glog will default to logging to a file.
	// By setting this debugging is easier via `kubectl logs`
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Build the client config - optionally using a provided kubeconfig file.
	config, err := common.GetClientConfig(opts.kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to load client config: %v", err)
	}

	// Construct the Kubernetes client
	kclient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	ver, err := kclient.DiscoveryClient.ServerVersion()
	if err != nil {
		glog.Fatalf("Failed to connect to API server: %v", err)
	}

	glog.Infof("Connected to API server (version: %s)", ver)

	pw := newPodWatcher(kclient, *opts)
	err = pw.informer.AddEventHandler(addHandlers())
	if err != nil {
		glog.Fatalf("Failed to add resource handlers: %v", err)
	}

	// Run the informer
	go pw.Run()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer close(sigs)

	// Which signal did we receive
	s := <-sigs
	glog.Infof("Got signal %v, trying graceful shutdown", s)

	// Tell the controller to stop
	close(pw.stopCh)
	// Wait for the controller to stop
	<-pw.doneCh

	glog.Infof("Done")

}

func newPodWatcher(kclient kubernetes.Interface, opts pwopts) *podWatcher {
	pw := &podWatcher{
		kclient: kclient,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	lw := cache.NewListWatchFromClient(kclient.CoreV1().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())

	informer := cache.NewSharedInformer(
		// ListWatch
		lw,

		// The types of objects this informer will return
		&v1.Pod{},

		// The resync period of this object. This will force a re-queue of all cached objects at this interval.
		// Every object will trigger the `Updatefunc` even if there have been no actual updates triggered.
		opts.resync,
	)

	pw.informer = informer
	return pw
}

func (c *podWatcher) Run() {
	glog.Info("Starting PodWatcher")
	go c.informer.Run(c.stopCh)

	// Wait for all caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(c.stopCh, c.informer.HasSynced) {
		glog.Error("Timed out waiting for caches to sync")
		return
	}

	<-c.stopCh
	glog.Info("Stopping PodWatcher")
	// Signal that we´re ok to stop
	c.doneCh <- struct{}{}
}

func addHandlers() cache.ResourceEventHandler {
	// Callback Functions to trigger on add/update/delete
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Type assertion
			pod, ok := obj.(*v1.Pod)
			if !ok {
				glog.Infoln("This is no pod!")
				return
			}
			glog.Info("Pod created: ", pod.Name)

		},
		UpdateFunc: func(old, new interface{}) {
			// no-op()
		},
		DeleteFunc: func(obj interface{}) {
			// Type assertion

			pod, ok := obj.(*v1.Pod)
			if !ok {
				glog.Infoln("This is no pod!")
				return
			}
			glog.Info("Pod deleted: ", pod.Name)
		},
	}
}
