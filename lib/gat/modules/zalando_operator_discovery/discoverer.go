package zalando_operator_discovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	acidzalando "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	acidv1informer "github.com/zalando/postgres-operator/pkg/generated/informers/externalversions/acid.zalan.do/v1"
	"github.com/zalando/postgres-operator/pkg/util"
	"github.com/zalando/postgres-operator/pkg/util/config"
	"github.com/zalando/postgres-operator/pkg/util/constants"
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"pggat/lib/gat/modules/discovery"
)

type Discoverer struct {
	config Config

	op *config.Config

	informer cache.SharedIndexInformer

	k8s k8sutil.KubernetesClient

	added   chan discovery.Cluster
	updated chan discovery.Cluster
	removed chan string
}

func NewDiscoverer(conf Config) (*Discoverer, error) {
	d := &Discoverer{
		config: conf,
	}
	if err := d.init(); err != nil {
		return nil, err
	}
	return d, nil
}

func (T *Discoverer) init() error {
	var err error
	T.k8s, err = k8sutil.NewFromConfig(T.config.Rest)
	if err != nil {
		return err
	}

	if T.config.ConfigMapName != "" {
		operatorConfig, err := T.k8s.ConfigMaps(T.config.Namespace).Get(context.Background(), T.config.ConfigMapName, metav1.GetOptions{})
		if err != nil {
			return nil
		}

		T.op = config.NewFromMap(operatorConfig.Data)
	} else if T.config.OperatorConfigurationObject != "" {
		operatorConfig, err := T.k8s.OperatorConfigurations(T.config.Namespace).Get(context.Background(), T.config.OperatorConfigurationObject, metav1.GetOptions{})
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
		T.config.Namespace,
		5*time.Minute,
		cache.Indexers{},
	)

	T.added = make(chan discovery.Cluster)
	T.updated = make(chan discovery.Cluster)
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
			T.updated <- cluster
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

	go T.informer.Run(nil)

	return nil
}

func (T *Discoverer) postgresqlToCluster(cluster acidv1.Postgresql) (discovery.Cluster, error) {
	c := discovery.Cluster{
		ID: string(cluster.UID),
		Primary: discovery.Endpoint{
			Network: "tcp",
			Address: fmt.Sprintf("%s.%s.svc.%s:5432", cluster.Name, T.config.Namespace, T.op.ClusterDomain),
		},
		Databases: make([]string, 0, len(cluster.Spec.Databases)),
		Users:     make([]discovery.User, 0, len(cluster.Spec.Users)),
	}
	if cluster.Spec.NumberOfInstances > 1 {
		c.Replicas = make(map[string]discovery.Endpoint, 1)
		c.Replicas["repl"] = discovery.Endpoint{
			Network: "tcp",
			Address: fmt.Sprintf("%s-repl.%s.svc.%s:5432", cluster.Name, T.config.Namespace, T.op.ClusterDomain),
		}
	}

	for user := range cluster.Spec.Users {
		secretName := T.op.SecretNameTemplate.Format(
			"username", strings.Replace(user, "_", "-", -1),
			"cluster", cluster.Name,
			"tprkind", acidv1.PostgresCRDResourceKind,
			"tprgroup", acidzalando.GroupName,
		)

		// get secret
		secret, err := T.k8s.Secrets(T.config.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
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
	return nil, nil
}

func (T *Discoverer) Added() <-chan discovery.Cluster {
	return T.added
}

func (T *Discoverer) Updated() <-chan discovery.Cluster {
	return T.updated
}

func (T *Discoverer) Removed() <-chan string {
	return T.removed
}

var _ discovery.Discoverer = (*Discoverer)(nil)
