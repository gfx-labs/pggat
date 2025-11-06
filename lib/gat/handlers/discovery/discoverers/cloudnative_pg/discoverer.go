package cloudnative_pg

import (
	"context"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"go.uber.org/zap"
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

	log *zap.Logger
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
	d.log = ctx.Logger()

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

	// Use NamespaceMatcher to configure label filtering
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		d.dynClient,
		0, // No resync
		d.Namespace.Namespace,
		d.Namespace.TweakListOptions,
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

// namespaceMatches checks if a namespace matches the label filter requirements
func (d *Discoverer) namespaceMatches(ctx context.Context, namespaceName string) bool {
	// If no label filters, always match
	if len(d.Namespace.Labels) == 0 {
		return d.Namespace.MatchesNamespace(namespaceName)
	}

	// Fetch namespace to check its labels
	ns, err := d.k8sClient.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		d.log.Error("failed to get namespace", zap.String("namespace", namespaceName), zap.Error(err))
		return false
	}

	return d.Namespace.MatchesNamespace(namespaceName) && d.Namespace.MatchesNamespaceLabels(ns.Labels)
}

func (d *Discoverer) handleClusterEvent(obj interface{}, eventType watch.EventType) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	var cluster cnpgv1.Cluster
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &cluster)
	if err != nil {
		d.log.Error("failed to convert unstructured to Cluster", zap.Error(err))
		return
	}

	// Filter by namespace labels if configured
	if !d.namespaceMatches(context.Background(), cluster.Namespace) {
		return
	}

	switch eventType {
	case watch.Added, watch.Modified:
		discoveryCluster, err := d.clusterToDiscoveryCluster(context.Background(), cluster)
		if err != nil {
			d.log.Error("failed to convert cluster", zap.String("cluster", cluster.Name), zap.Error(err))
			return
		}
		select {
		case d.added <- discoveryCluster:
		default:
			d.log.Error("dropped add/update event - added channel full",
				zap.String("cluster", cluster.Name),
				zap.String("namespace", cluster.Namespace),
				zap.String("uid", string(cluster.UID)))
		}
	case watch.Deleted:
		select {
		case d.removed <- string(cluster.UID):
		default:
			d.log.Error("dropped delete event - removed channel full",
				zap.String("cluster", cluster.Name),
				zap.String("namespace", cluster.Namespace),
				zap.String("uid", string(cluster.UID)))
		}
	}
}

func (d *Discoverer) clusterToDiscoveryCluster(ctx context.Context, cluster cnpgv1.Cluster) (discovery.Cluster, error) {
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
			database = "postgres"
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
				ctx, secretName, metav1.GetOptions{})
			if err == nil {
				if password, ok := secret.Data["password"]; ok {
					discoveryCluster.Users = append(discoveryCluster.Users, discovery.User{
						Username: owner,
						Password: string(password),
					})
				}
			} else {
				d.log.Warn("failed to get secret",
					zap.String("secret", secretName),
					zap.String("cluster", cluster.Name),
					zap.Error(err))
			}
		}
	}

	// Optionally include superuser
	if d.IncludeSuperuser && cluster.Spec.SuperuserSecret != nil && cluster.Spec.SuperuserSecret.Name != "" {
		secret, err := d.k8sClient.CoreV1().Secrets(namespace).Get(
			ctx, cluster.Spec.SuperuserSecret.Name, metav1.GetOptions{})
		if err == nil {
			if password, ok := secret.Data["password"]; ok {
				discoveryCluster.Users = append(discoveryCluster.Users, discovery.User{
					Username: "postgres",
					Password: string(password),
				})
			}
		} else {
			d.log.Warn("failed to get superuser secret",
				zap.String("secret", cluster.Spec.SuperuserSecret.Name),
				zap.String("cluster", cluster.Name),
				zap.Error(err))
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

		// Filter by namespace labels
		if !d.namespaceMatches(context.Background(), cluster.Namespace) {
			continue
		}

		discoveryCluster, err := d.clusterToDiscoveryCluster(context.Background(), cluster)
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
