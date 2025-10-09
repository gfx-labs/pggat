package cloudnative_pg

import (
	"context"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"
)

func init() {
	caddy.RegisterModule((*Discoverer)(nil))
}

// Discoverer discovers CloudNativePG PostgreSQL clusters in Kubernetes
type Discoverer struct {
	Config

	restConfig *rest.Config
	k8sClient  kubernetes.Interface
	dynClient  dynamic.Interface

	informer cache.SharedIndexInformer
	stopCh   chan struct{}

	added   chan discovery.Cluster
	removed chan string
}

func (d *Discoverer) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.discovery.discoverers.cloudnative_pg",
		New: func() caddy.Module {
			return new(Discoverer)
		},
	}
}

func (d *Discoverer) Provision(ctx caddy.Context) error {
	// Set defaults
	if d.ClusterDomain == "" {
		d.ClusterDomain = "cluster.local"
	}
	if d.ReadWriteServiceSuffix == "" {
		d.ReadWriteServiceSuffix = "-rw"
	}
	if d.ReadOnlyServiceSuffix == "" {
		d.ReadOnlyServiceSuffix = "-ro"
	}
	if d.Port == 0 {
		d.Port = 5432
	}
	if d.SecretSuffix == "" {
		d.SecretSuffix = "-app"
	}

	// Initialize Kubernetes clients
	var err error
	d.restConfig, err = rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	d.k8sClient, err = kubernetes.NewForConfig(d.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	d.dynClient, err = dynamic.NewForConfig(d.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create informer for CloudNativePG clusters
	gvr := schema.GroupVersionResource{
		Group:    cnpgv1.SchemeGroupVersion.Group,
		Version:  cnpgv1.SchemeGroupVersion.Version,
		Resource: "clusters",
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		d.dynClient,
		0, // No resync
		d.Namespace,
		nil,
	)

	d.informer = factory.ForResource(gvr).Informer()

	// Initialize channels
	d.added = make(chan discovery.Cluster, 10)
	d.removed = make(chan string, 10)
	d.stopCh = make(chan struct{})

	// Set up event handlers
	_, err = d.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			d.handleClusterEvent(obj, watch.Added)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			d.handleClusterEvent(newObj, watch.Modified)
		},
		DeleteFunc: func(obj interface{}) {
			d.handleClusterEvent(obj, watch.Deleted)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start the informer
	go d.informer.Run(d.stopCh)

	// Wait for cache sync
	if !cache.WaitForCacheSync(ctx.Done(), d.informer.HasSynced) {
		return fmt.Errorf("failed to sync cache")
	}

	return nil
}

func (d *Discoverer) Cleanup() error {
	if d.stopCh != nil {
		close(d.stopCh)
	}
	if d.added != nil {
		close(d.added)
	}
	if d.removed != nil {
		close(d.removed)
	}
	return nil
}

func (d *Discoverer) handleClusterEvent(obj interface{}, eventType watch.EventType) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	var cluster cnpgv1.Cluster
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &cluster)
	if err != nil {
		fmt.Printf("Failed to convert unstructured to Cluster: %v\n", err)
		return
	}

	switch eventType {
	case watch.Added, watch.Modified:
		discoveryCluster, err := d.clusterToDiscoveryCluster(cluster)
		if err != nil {
			fmt.Printf("Failed to convert cluster %s: %v\n", cluster.Name, err)
			return
		}
		select {
		case d.added <- discoveryCluster:
		default:
			// Channel full, drop the event
		}
	case watch.Deleted:
		select {
		case d.removed <- string(cluster.UID):
		default:
			// Channel full, drop the event
		}
	}
}

func (d *Discoverer) clusterToDiscoveryCluster(cluster cnpgv1.Cluster) (discovery.Cluster, error) {
	namespace := cluster.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Build service endpoints
	primaryEndpoint := fmt.Sprintf("%s%s.%s.svc.%s:%d",
		cluster.Name, d.ReadWriteServiceSuffix, namespace, d.ClusterDomain, d.Port)

	readOnlyEndpoint := fmt.Sprintf("%s%s.%s.svc.%s:%d",
		cluster.Name, d.ReadOnlyServiceSuffix, namespace, d.ClusterDomain, d.Port)

	discoveryCluster := discovery.Cluster{
		ID: string(cluster.UID),
		Primary: discovery.Node{
			Address: primaryEndpoint,
		},
		Databases: make([]string, 0),
		Users:     make([]discovery.User, 0),
	}

	// Add replicas if instances > 1
	if cluster.Spec.Instances > 1 {
		discoveryCluster.Replicas = map[string]discovery.Node{
			"read-only": {
				Address: readOnlyEndpoint,
			},
		}
	}

	// Get database and user information from bootstrap config
	if cluster.Spec.Bootstrap != nil && cluster.Spec.Bootstrap.InitDB != nil {
		initDB := cluster.Spec.Bootstrap.InitDB

		// Add database
		database := initDB.Database
		if database == "" {
			database = "app"
		}
		discoveryCluster.Databases = append(discoveryCluster.Databases, database)

		// Add application user
		owner := initDB.Owner
		if owner == "" {
			owner = database
		}

		// Try to get the password from the specified secret or default pattern
		var secretName string
		if initDB.Secret != nil && initDB.Secret.Name != "" {
			secretName = initDB.Secret.Name
		} else {
			// Try default secret name pattern
			secretName = cluster.Name + d.SecretSuffix
		}

		if secretName != "" {
			secret, err := d.k8sClient.CoreV1().Secrets(namespace).Get(
				context.Background(), secretName, metav1.GetOptions{})
			if err == nil {
				if password, ok := secret.Data["password"]; ok {
					discoveryCluster.Users = append(discoveryCluster.Users, discovery.User{
						Username: owner,
						Password: string(password),
					})
				}
			} else {
				fmt.Printf("Failed to get secret %s for cluster %s: %v\n",
					secretName, cluster.Name, err)
			}
		}
	}

	// Optionally include superuser
	if d.IncludeSuperuser && cluster.Spec.SuperuserSecret != nil && cluster.Spec.SuperuserSecret.Name != "" {
		secret, err := d.k8sClient.CoreV1().Secrets(namespace).Get(
			context.Background(), cluster.Spec.SuperuserSecret.Name, metav1.GetOptions{})
		if err == nil {
			if password, ok := secret.Data["password"]; ok {
				discoveryCluster.Users = append(discoveryCluster.Users, discovery.User{
					Username: "postgres",
					Password: string(password),
				})
			}
		} else {
			fmt.Printf("Failed to get superuser secret %s for cluster %s: %v\n",
				cluster.Spec.SuperuserSecret.Name, cluster.Name, err)
		}
	}

	// Add postgres database (always exists)
	discoveryCluster.Databases = append(discoveryCluster.Databases, "postgres")

	return discoveryCluster, nil
}

func (d *Discoverer) Clusters() ([]discovery.Cluster, error) {
	// Get all items from the informer's cache
	items := d.informer.GetStore().List()

	clusters := make([]discovery.Cluster, 0, len(items))
	for _, item := range items {
		unstructuredObj, ok := item.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		var cluster cnpgv1.Cluster
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			unstructuredObj.UnstructuredContent(), &cluster)
		if err != nil {
			continue
		}

		discoveryCluster, err := d.clusterToDiscoveryCluster(cluster)
		if err != nil {
			continue
		}

		clusters = append(clusters, discoveryCluster)
	}

	return clusters, nil
}

func (d *Discoverer) Added() <-chan discovery.Cluster {
	return d.added
}

func (d *Discoverer) Removed() <-chan string {
	return d.removed
}

// Interface assertions
var (
	_ discovery.Discoverer = (*Discoverer)(nil)
	_ caddy.Module         = (*Discoverer)(nil)
	_ caddy.Provisioner    = (*Discoverer)(nil)
	_ caddy.CleanerUpper   = (*Discoverer)(nil)
)
