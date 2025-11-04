package zalando_operator

import (
	"context"
	"fmt"
	acidv1informer "github.com/zalando/postgres-operator/pkg/generated/informers/externalversions/acid.zalan.do/v1"
	"strings"

	"github.com/caddyserver/caddy/v2"
	acidzalando "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"github.com/zalando/postgres-operator/pkg/util"
	"github.com/zalando/postgres-operator/pkg/util/config"
	"github.com/zalando/postgres-operator/pkg/util/constants"
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
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
		operatorConfig, err := T.k8s.ConfigMaps(T.Namespace).Get(ctx, T.ConfigMapName, metav1.GetOptions{})
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
			operatorConfig, err := T.k8s.OperatorConfigurations(T.Namespace).Get(ctx, T.OperatorConfigurationObject, metav1.GetOptions{})
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
		T.Namespace,
		0,
		cache.Indexers{},
	)

	T.added = make(chan discovery.Cluster, 10)
	T.removed = make(chan string, 10)

	_, err = T.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			psql, ok := obj.(*acidv1.Postgresql)
			if !ok {
				return
			}
			cluster, err := T.postgresqlToCluster(*psql)
			if err != nil {
				return
			}
			select {
			case T.added <- cluster:
			default:
				fmt.Printf("ERROR: Dropped add event for cluster %s (namespace: %s, UID: %s) - added channel full\n",
					psql.Name, psql.Namespace, psql.UID)
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			psql, ok := obj.(*acidv1.Postgresql)
			if !ok {
				return
			}
			cluster, err := T.postgresqlToCluster(*psql)
			if err != nil {
				return
			}
			select {
			case T.added <- cluster:
			default:
				fmt.Printf("ERROR: Dropped update event for cluster %s (namespace: %s, UID: %s) - added channel full\n",
					psql.Name, psql.Namespace, psql.UID)
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
				fmt.Printf("ERROR: Dropped delete event for cluster %s (namespace: %s, UID: %s) - removed channel full\n",
					psql.Name, psql.Namespace, psql.UID)
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

func (T *Discoverer) postgresqlToCluster(cluster acidv1.Postgresql) (discovery.Cluster, error) {
	c := discovery.Cluster{
		ID: string(cluster.UID),
		Primary: discovery.Node{
			Address: fmt.Sprintf("%s.%s.svc.%s:5432", cluster.Name, T.Namespace, T.op.ClusterDomain),
		},
		Databases: make([]string, 0, len(cluster.Spec.Databases)),
		Users:     make([]discovery.User, 0, len(cluster.Spec.Users)),
	}
	if cluster.Spec.NumberOfInstances > 1 {
		c.Replicas = make(map[string]discovery.Node, 1)
		c.Replicas["repl"] = discovery.Node{
			Address: fmt.Sprintf("%s-repl.%s.svc.%s:5432", cluster.Name, T.Namespace, T.op.ClusterDomain),
		}
	}

	for user := range cluster.Spec.Users {
		secretName := T.op.SecretNameTemplate.Format(
			"username", strings.ReplaceAll(user, "_", "-"),
			"cluster", cluster.Name,
			"tprkind", acidv1.PostgresCRDResourceKind,
			"tprgroup", acidzalando.GroupName,
		)

		// get secret
		secret, err := T.k8s.Secrets(T.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Failed to get secret for user %s: %v\n", user, err)
			continue
		}

		password, ok := secret.Data["password"]
		if !ok {
			fmt.Printf("No password in secret: %s\n", secretName)
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
	clusters, err := T.k8s.Postgresqls(T.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := make([]discovery.Cluster, 0, len(clusters.Items))
	for _, cluster := range clusters.Items {
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
