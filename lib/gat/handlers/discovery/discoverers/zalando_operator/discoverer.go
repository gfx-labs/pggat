package zalando_operator

import (
	"context"
	"fmt"
	"strings"

	acidv1informer "github.com/zalando/postgres-operator/pkg/generated/informers/externalversions/acid.zalan.do/v1"

	"github.com/caddyserver/caddy/v2"
	acidzalando "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"github.com/zalando/postgres-operator/pkg/util"
	"github.com/zalando/postgres-operator/pkg/util/config"
	"github.com/zalando/postgres-operator/pkg/util/constants"
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"
)

func init() {
	caddy.RegisterModule((*Discoverer)(nil))
}

type Discoverer struct {
	Config

	rest *rest.Config

	op *config.Config

	informer cache.SharedIndexInformer

	k8s k8sutil.KubernetesClient

	added   chan discovery.Cluster
	removed chan string

	done chan struct{}

	log *zap.Logger
}

func (T *Discoverer) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.discovery.discoverers.zalando_operator",
		New: func() caddy.Module {
			return new(Discoverer)
		},
	}
}

func (T *Discoverer) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger().With(zap.String("discoverer", "zalando_operator"))

	var err error
	T.rest, err = rest.InClusterConfig()
	if err != nil {
		return err
	}

	T.k8s, err = k8sutil.NewFromConfig(T.rest)
	if err != nil {
		return err
	}

	if T.ConfigMapName != "" {
		operatorConfig, err := T.k8s.ConfigMaps(T.Namespace.Namespace).Get(ctx, T.ConfigMapName, metav1.GetOptions{})
		if err != nil {
			return nil
		}

		T.op = config.NewFromMap(operatorConfig.Data)
	} else {
		// defaults
		T.op = new(config.Config)

		// from Caddyfile
		T.op.ClusterDomain = util.Coalesce(T.ClusterDomain, "cluster.local")

		T.op.SecretNameTemplate = config.StringTemplate(T.SecretNameTemplate)

		T.op.NumberOfInstances = util.CoalesceInt32(
			T.ConnectionPoolerNumberOfInstances,
			k8sutil.Int32ToPointer(2))

		T.op.Mode = util.Coalesce(
			T.ConnectionPoolerMode,
			constants.ConnectionPoolerDefaultMode)

		T.op.MaxDBConnections = util.CoalesceInt32(
			T.ConnectionPoolerMaxDBConnections,
			k8sutil.Int32ToPointer(constants.ConnectionPoolerMaxDBConnections))

		// from external config
		if T.OperatorConfigurationObject != "" {
			operatorConfig, err := T.k8s.OperatorConfigurations(T.Namespace.Namespace).Get(ctx, T.OperatorConfigurationObject, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// why did they do this to me
			T.op.ClusterDomain = util.Coalesce(operatorConfig.Configuration.Kubernetes.ClusterDomain, T.op.ClusterDomain)

			T.op.SecretNameTemplate = config.StringTemplate(util.Coalesce(
				string(operatorConfig.Configuration.Kubernetes.SecretNameTemplate),
				string(T.op.SecretNameTemplate)))

			T.op.NumberOfInstances = util.CoalesceInt32(
				operatorConfig.Configuration.ConnectionPooler.NumberOfInstances,
				T.op.NumberOfInstances)

			T.op.Mode = util.Coalesce(
				operatorConfig.Configuration.ConnectionPooler.Mode,
				T.op.Mode)

			T.op.MaxDBConnections = util.CoalesceInt32(
				operatorConfig.Configuration.ConnectionPooler.MaxDBConnections,
				T.op.MaxDBConnections)
		}
	}

	T.informer = acidv1informer.NewPostgresqlInformer(
		T.k8s.AcidV1ClientSet,
		T.Namespace.Namespace,
		0,
		cache.Indexers{},
	)

	// Initialize channels with larger buffers to handle many clusters
	T.added = make(chan discovery.Cluster, 200)
	T.removed = make(chan string, 200)

	_, err = T.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			psql, ok := obj.(*acidv1.Postgresql)
			if !ok {
				return
			}
			// Filter by namespace labels if configured
			if !T.namespaceMatches(psql.Namespace) {
				return
			}
			cluster, err := T.postgresqlToCluster(*psql)
			if err != nil {
				return
			}
			select {
			case T.added <- cluster:
			default:
				T.log.Error("dropped add event - added channel full",
					zap.String("cluster", psql.Name),
					zap.String("namespace", psql.Namespace),
					zap.String("uid", string(psql.UID)))
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			psql, ok := obj.(*acidv1.Postgresql)
			if !ok {
				return
			}
			// Filter by namespace labels if configured
			if !T.namespaceMatches(psql.Namespace) {
				return
			}
			cluster, err := T.postgresqlToCluster(*psql)
			if err != nil {
				return
			}
			select {
			case T.added <- cluster:
			default:
				T.log.Error("dropped update event - added channel full",
					zap.String("cluster", psql.Name),
					zap.String("namespace", psql.Namespace),
					zap.String("uid", string(psql.UID)))
			}
		},
		DeleteFunc: func(obj interface{}) {
			psql, ok := obj.(*acidv1.Postgresql)
			if !ok {
				return
			}
			select {
			case T.removed <- string(psql.UID):
			default:
				T.log.Error("dropped delete event - removed channel full",
					zap.String("cluster", psql.Name),
					zap.String("namespace", psql.Namespace),
					zap.String("uid", string(psql.UID)))
			}
		},
	})
	if err != nil {
		return err
	}

	T.done = make(chan struct{})
	go T.informer.Run(T.done)

	return nil
}

func (T *Discoverer) Cleanup() error {
	if T.done == nil {
		return nil
	}
	close(T.done)
	return nil
}

// namespaceMatches checks if a namespace matches the label filter requirements
func (T *Discoverer) namespaceMatches(namespaceName string) bool {
	// If no label filters, always match
	if len(T.Namespace.Labels) == 0 {
		return T.Namespace.MatchesNamespace(namespaceName)
	}

	// Fetch namespace to check its labels
	ns, err := T.k8s.Namespaces().Get(context.Background(), namespaceName, metav1.GetOptions{})
	if err != nil {
		T.log.Warn("failed to get namespace",
			zap.String("namespace", namespaceName),
			zap.Error(err))
		return false
	}

	return T.Namespace.MatchesNamespace(namespaceName) && T.Namespace.MatchesNamespaceLabels(ns.Labels)
}

func (T *Discoverer) postgresqlToCluster(cluster acidv1.Postgresql) (discovery.Cluster, error) {
	c := discovery.Cluster{
		ID: string(cluster.UID),
		Primary: discovery.Node{
			Address: fmt.Sprintf("%s.%s.svc.%s:5432", cluster.Name, cluster.Namespace, T.op.ClusterDomain),
		},
		Databases: make([]string, 0, len(cluster.Spec.Databases)),
		Users:     make([]discovery.User, 0, len(cluster.Spec.Users)),
	}
	if cluster.Spec.NumberOfInstances > 1 {
		c.Replicas = make(map[string]discovery.Node, 1)
		c.Replicas["repl"] = discovery.Node{
			Address: fmt.Sprintf("%s-repl.%s.svc.%s:5432", cluster.Name, cluster.Namespace, T.op.ClusterDomain),
		}
	}

	for user := range cluster.Spec.Users {
		secretName := T.op.SecretNameTemplate.Format(
			"username", strings.ReplaceAll(user, "_", "-"),
			"cluster", cluster.Name,
			"tprkind", acidv1.PostgresCRDResourceKind,
			"tprgroup", acidzalando.GroupName,
		)

		// get secret - use cluster's namespace, not discoverer's namespace
		secret, err := T.k8s.Secrets(cluster.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err != nil {
			T.log.Warn("failed to get secret for user",
				zap.String("user", user),
				zap.String("secret", secretName),
				zap.String("cluster", cluster.Name),
				zap.String("namespace", cluster.Namespace),
				zap.Error(err))
			continue
		}

		password, ok := secret.Data["password"]
		if !ok {
			T.log.Warn("no password in secret",
				zap.String("secret", secretName),
				zap.String("user", user),
				zap.String("cluster", cluster.Name),
				zap.String("namespace", cluster.Namespace))
			continue
		}

		c.Users = append(c.Users, discovery.User{
			Username: user,
			Password: string(password),
		})
	}

	for database := range cluster.Spec.Databases {
		c.Databases = append(c.Databases, database)
	}
	return c, nil
}

func (T *Discoverer) Clusters() ([]discovery.Cluster, error) {
	listOpts := metav1.ListOptions{}
	// Note: TweakListOptions adds label filters but those are for cluster labels, not namespace labels
	// We filter by namespace labels separately below

	clusters, err := T.k8s.Postgresqls(T.Namespace.Namespace).List(context.Background(), listOpts)
	if err != nil {
		return nil, err
	}

	res := make([]discovery.Cluster, 0, len(clusters.Items))
	for _, cluster := range clusters.Items {
		// Filter by namespace labels
		if !T.namespaceMatches(cluster.Namespace) {
			continue
		}
		r, err := T.postgresqlToCluster(cluster)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (T *Discoverer) Added() <-chan discovery.Cluster {
	return T.added
}

func (T *Discoverer) Removed() <-chan string {
	return T.removed
}

var (
	_ discovery.Discoverer = (*Discoverer)(nil)
	_ caddy.Module         = (*Discoverer)(nil)
	_ caddy.Provisioner    = (*Discoverer)(nil)
	_ caddy.CleanerUpper   = (*Discoverer)(nil)
)
