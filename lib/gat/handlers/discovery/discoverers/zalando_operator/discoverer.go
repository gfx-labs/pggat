package zalando_operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
	acidzalando "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	acidv1informer "github.com/zalando/postgres-operator/pkg/generated/informers/externalversions/acid.zalan.do/v1"
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
	} else if T.OperatorConfigurationObject != "" {
		operatorConfig, err := T.k8s.OperatorConfigurations(T.Namespace).Get(ctx, T.OperatorConfigurationObject, metav1.GetOptions{})
		if err != nil {
			return err
		}

		T.op = new(config.Config)

		// why did they do this to me
		T.op.ClusterDomain = util.Coalesce(operatorConfig.Configuration.Kubernetes.ClusterDomain, "cluster.local")

		T.op.SecretNameTemplate = operatorConfig.Configuration.Kubernetes.SecretNameTemplate

		T.op.ConnectionPooler.NumberOfInstances = util.CoalesceInt32(
			operatorConfig.Configuration.ConnectionPooler.NumberOfInstances,
			k8sutil.Int32ToPointer(2))

		T.op.ConnectionPooler.Mode = util.Coalesce(
			operatorConfig.Configuration.ConnectionPooler.Mode,
			constants.ConnectionPoolerDefaultMode)

		T.op.ConnectionPooler.MaxDBConnections = util.CoalesceInt32(
			operatorConfig.Configuration.ConnectionPooler.MaxDBConnections,
			k8sutil.Int32ToPointer(constants.ConnectionPoolerMaxDBConnections))
	} else {
		// defaults
		T.op = config.NewFromMap(make(map[string]string))
	}

	T.informer = acidv1informer.NewPostgresqlInformer(
		T.k8s.AcidV1ClientSet,
		T.Namespace,
		0,
		cache.Indexers{},
	)

	T.added = make(chan discovery.Cluster)
	T.removed = make(chan string)

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
			T.added <- cluster
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
			T.added <- cluster
		},
		DeleteFunc: func(obj interface{}) {
			psql, ok := obj.(*acidv1.Postgresql)
			if !ok {
				return
			}
			T.removed <- string(psql.UID)
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
	close(T.done)
	return nil
}

func (T *Discoverer) postgresqlToCluster(cluster acidv1.Postgresql) (discovery.Cluster, error) {
	c := discovery.Cluster{
		ID:        string(cluster.UID),
		Primary:   fmt.Sprintf("%s.%s.svc.%s:5432", cluster.Name, T.Namespace, T.op.ClusterDomain),
		Databases: make([]string, 0, len(cluster.Spec.Databases)),
		Users:     make([]discovery.User, 0, len(cluster.Spec.Users)),
	}
	if cluster.Spec.NumberOfInstances > 1 {
		c.Replicas = make(map[string]string, 1)
		c.Replicas["repl"] = fmt.Sprintf("%s-repl.%s.svc.%s:5432", cluster.Name, T.Namespace, T.op.ClusterDomain)
	}

	for user := range cluster.Spec.Users {
		secretName := T.op.SecretNameTemplate.Format(
			"username", strings.Replace(user, "_", "-", -1),
			"cluster", cluster.Name,
			"tprkind", acidv1.PostgresCRDResourceKind,
			"tprgroup", acidzalando.GroupName,
		)

		// get secret
		secret, err := T.k8s.Secrets(T.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err != nil {
			return discovery.Cluster{}, err
		}

		password, ok := secret.Data["password"]
		if !ok {
			return discovery.Cluster{}, fmt.Errorf("no password in secret: %s", secretName)
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

var _ discovery.Discoverer = (*Discoverer)(nil)
var _ caddy.Module = (*Discoverer)(nil)
var _ caddy.Provisioner = (*Discoverer)(nil)
var _ caddy.CleanerUpper = (*Discoverer)(nil)
