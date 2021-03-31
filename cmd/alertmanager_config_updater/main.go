package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/sputnik-systems/alertmanager_bot/pkg/alertmanager"
)

var kubeconfig, namespace, secret, config, url string

func main() {
	flag.StringVar(&kubeconfig, "kube.config", "", "path to kubeconfig")
	flag.StringVar(&namespace, "kube.namespace", "default", "namespace name")
	flag.StringVar(&secret, "kube.secret-name", "vmalertmanager-test", "secret name")
	flag.StringVar(&config, "alertmanager.config", "/etc/alertmanager/alertmanager.yaml", "path to alertmanager config")
	flag.StringVar(&url, "alertmanager.url", "http://localhost:9093", "alertmanager url")
	flag.Parse()

	if kubeconfig == "" {
		klog.Fatal("kubeconfig path should be specified")
	}

	c, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatal("failed create config object with given kubeconfig: %s", err)
	}

	k, err := kubernetes.NewForConfig(c)
	if err != nil {
		klog.Fatal("failed create kube client: %s", err)
	}

	watcher := cache.NewListWatchFromClient(k.CoreV1().RESTClient(), "secrets", namespace, fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	indexer, informer := cache.NewIndexerInformer(watcher, &v1.Secret{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewController(queue, indexer, informer)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	select {}
}

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()

	klog.Info("starting secret controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
	klog.Info("stoppping secret controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	if key == fmt.Sprintf("%s/%s", namespace, secret) {
		err := c.saveConfigToFile(key.(string))

		// Handle the error if something went wrong during the execution of the business logic
		c.handleErr(err, key)
	}

	return true
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	klog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

func (c *Controller) saveConfigToFile(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Pod, so that we will see a delete for one pod
		klog.Infof("secret %s does not exist anymore\n", key)
	} else {
		var value []byte
		var ok bool

		if value, ok = obj.(*v1.Secret).Data["alertmanager.yaml"]; !ok {
			klog.Error("alertmanager.yaml no found in secret")
		} else {
			klog.Info("overwriting alertmanager.yaml from secret")
			f, err := os.OpenFile(config, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed open config file: %s", err)
			}

			_, err = f.Write(value)
			if err != nil {
				return fmt.Errorf("failed write to file: %s", err)
			}

			_, err = alertmanager.Reload(url)
			if err != nil {
				return fmt.Errorf("failed to reload alertmanager: %s", err)
			}

			klog.Info("alertmanager successfully reloaded")
		}
	}

	return nil
}
