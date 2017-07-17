// Building a simple custome controller for Kubernetes
// Based on the great work by https://github.com/aaronlevy/kube-controller-demo
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/embano1/tompodinformer/common"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	lister_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type podWatcher struct {
	kclient   kubernetes.Interface
	podLister lister_v1.PodLister
	informer  cache.Controller
	stopCh    chan struct{}
	doneCh    chan struct{}
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

	pw := newPodWatcher(kclient, *opts)
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
	rc := &podWatcher{
		kclient: kclient,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	indexer, informer := cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(lo meta_v1.ListOptions) (runtime.Object, error) {
				// We do not add any selectors because we want to watch all pods.
				return kclient.Core().Pods(meta_v1.NamespaceDefault).List(lo)
			},
			WatchFunc: func(lo meta_v1.ListOptions) (watch.Interface, error) {
				return kclient.Core().Pods(meta_v1.NamespaceDefault).Watch(lo)
			},
		},
		// The types of objects this informer will return
		&v1.Pod{},
		// The resync period of this object. This will force a re-queue of all cached objects at this interval.
		// Every object will trigger the `Updatefunc` even if there have been no actual updates triggered.
		// In some cases you can set this to a very high interval - as you can assume you will see periodic updates in normal operation.
		// The interval is set low here for demo purposes.
		opts.resync,
		// Callback Functions to trigger on add/update/delete
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// Type assertion
				pod, ok := obj.(*v1.Pod)
				if !ok {
					glog.Infoln("This is no pod!")
					return
				}
				fmt.Println("Pod created:", pod.Name)

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
				fmt.Println("Pod deleted:", pod.Name)
			},
		},
		cache.Indexers{},
	)

	rc.informer = informer
	// NodeLister avoids some boilerplate code (e.g. convert runtime.Object to *v1.node)
	rc.podLister = lister_v1.NewPodLister(indexer)

	return rc
}

func (c *podWatcher) Run() {
	glog.Info("Starting RebootController")

	go c.informer.Run(c.stopCh)

	// Wait for all caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(c.stopCh, c.informer.HasSynced) {
		glog.Error(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	<-c.stopCh
	glog.Info("Stopping Reboot Controller")
	// Signal that we´re ok to stop
	c.doneCh <- struct{}{}
}
