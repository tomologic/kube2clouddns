package main

import (
	LogLib "log"
	"flag"
	"time"
	"k8s.io/client-go/1.5/tools/cache"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/tools/clientcmd"
	"os"
	"os/signal"
	"syscall"
	"io/ioutil"
)

var log = LogLib.New(os.Stderr, "kube2clouddns: ", LogLib.LstdFlags | LogLib.Lshortfile)

type DnsUpdater struct {
	dnsClient *CloudDnsClient
}

func (client *DnsUpdater) serviceCreated(obj interface{}) {
	service := obj.(*v1.Service)
	log.Println("Service created: " + service.Name)
	client.upsertService(service)
}
func (client *DnsUpdater) serviceDeleted(obj interface{}) {
	service := obj.(*v1.Service)
	log.Println("Service deleted: " + service.Name)
	client.deleteService(service)
}
func (client *DnsUpdater) serviceUpdated(oldObj, newObj interface{}) {
	oldService := oldObj.(*v1.Service)
	newService := newObj.(*v1.Service)
	log.Println("Service updated from: " + oldService.ObjectMeta.Name + " to: " + newService.ObjectMeta.Name)
	client.upsertService(newService)
	if oldService.Name != newService.Name {
		client.deleteService(oldService)
	}
}
func (client *DnsUpdater) upsertService(service *v1.Service) {
	externalDnsLabel, ok := service.Labels["external_dns"]
	if ok && externalDnsLabel == "true" {
		err := client.dnsClient.upsert(service.Name, service.Spec.ClusterIP, 60)
		if err != nil {
			log.Println(err)
		}
	}
}
func (client *DnsUpdater) deleteService(service *v1.Service) {
	externalDnsLabel, ok := service.Labels["external_dns"]
	if ok && externalDnsLabel == "true" {
		err := client.dnsClient.delete(service.Name)
		if err != nil {
			log.Println(err)
		}
	}
}

func watchServicesAndUpdateCloudDNS(kubeClientset *kubernetes.Clientset, dnsUpdater DnsUpdater, done chan struct{}) (cache.Store) {

	//Define what we want to look for (Services)
	watchlist := cache.NewListWatchFromClient(kubeClientset.Core().GetRESTClient(), "services", v1.NamespaceDefault, fields.Everything())
	resyncPeriod := 30 * time.Minute

	//Setup an informer to call functions when the watchlist changes
	servicesStore, eController := cache.NewInformer(
		watchlist,
		&v1.Service{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    dnsUpdater.serviceCreated,
			DeleteFunc: dnsUpdater.serviceDeleted,
			UpdateFunc: dnsUpdater.serviceUpdated,
		},
	)
	//Run the controller as a goroutine
	go eController.Run(done)

	return servicesStore
}

var (
	kubeconfig = flag.String("kubeconfig", "", "absolute path to a kubeconfig file")
	serviceaccount = flag.String("serviceaccount", "", "absolute path to a service account json file")
	project = flag.String("project", "", "The GCP project id in which the DNS zone for the domain is hosted")
	domain = flag.String("domain", "", "The domain that should host the service sub domains")
)

func main() {

	flag.Parse()

	// Setup Client for communication with Kubernetes API server
	// If a kubeconfig is supplied, use it, otherwise it assumes that we run in a cluster
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	kClient := kubernetes.NewForConfigOrDie(restConfig)

	//Setup Updater client for Cloud DNS
	serviceAccount, err := ioutil.ReadFile(*serviceaccount)
	if err != nil {
		log.Fatal(err)
	}
	// Cloud DNS Client needs a service account and some config (project and domain)
	dnsClient, err := NewDNSClient(serviceAccount, *domain, *project)
	if err != nil {
		log.Fatal(err)
	}
	dnsUpdater := DnsUpdater{dnsClient: dnsClient}

	// This channel is used to close the watch routine when the application exits
	doneChan := make(chan struct{})

	// Watch for events that add, modify, or delete services and process them asynchronously.
	log.Println("Watching for service events.")
	watchServicesAndUpdateCloudDNS(kClient, dnsUpdater, doneChan)

	// Stay alive until shutdown signal received
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-signalChan:
			log.Printf("Shutdown signal received, exiting...")
			close(doneChan)
			os.Exit(0)
		}
	}
}

type DNSConfig struct {
	projectId string
	domain    string
}
