package zalando_operator_discovery

import (
	"context"
	"fmt"
	"strings"

	acidzalando "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"github.com/zalando/postgres-operator/pkg/util"
	"github.com/zalando/postgres-operator/pkg/util/config"
	"github.com/zalando/postgres-operator/pkg/util/constants"
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"pggat/lib/gat/modules/discovery"
)

type Discoverer struct {
	config Config

	op *config.Config

	k8s k8sutil.KubernetesClient
}

func NewDiscoverer(conf Config) (*Discoverer, error) {
	k8s, err := k8sutil.NewFromConfig(conf.Rest)
	if err != nil {
		return nil, err
	}

	var op *config.Config
	if conf.ConfigMapName != "" {
		operatorConfig, err := k8s.ConfigMaps(conf.Namespace).Get(context.Background(), conf.ConfigMapName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		op = config.NewFromMap(operatorConfig.Data)
	} else if conf.OperatorConfigurationObject != "" {
		operatorConfig, err := k8s.OperatorConfigurations(conf.Namespace).Get(context.Background(), conf.OperatorConfigurationObject, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		op = new(config.Config)

		// why did they do this to me
		op.ClusterDomain = util.Coalesce(operatorConfig.Configuration.Kubernetes.ClusterDomain, "cluster.local")

		op.SecretNameTemplate = operatorConfig.Configuration.Kubernetes.SecretNameTemplate

		op.ConnectionPooler.NumberOfInstances = util.CoalesceInt32(
			operatorConfig.Configuration.ConnectionPooler.NumberOfInstances,
			k8sutil.Int32ToPointer(2))

		op.ConnectionPooler.Mode = util.Coalesce(
			operatorConfig.Configuration.ConnectionPooler.Mode,
			constants.ConnectionPoolerDefaultMode)

		op.ConnectionPooler.MaxDBConnections = util.CoalesceInt32(
			operatorConfig.Configuration.ConnectionPooler.MaxDBConnections,
			k8sutil.Int32ToPointer(constants.ConnectionPoolerMaxDBConnections))
	} else {
		// defaults
		op = config.NewFromMap(make(map[string]string))
	}

	return &Discoverer{
		config: conf,
		op:     op,
		k8s:    k8s,
	}, nil
}

func (T *Discoverer) Clusters() ([]discovery.Cluster, error) {
	clusters, err := T.k8s.Postgresqls(T.config.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := make([]discovery.Cluster, 0, len(clusters.Items))
	for _, cluster := range clusters.Items {
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
				return nil, err
			}

			password, ok := secret.Data["password"]
			if !ok {
				return nil, fmt.Errorf("no password in secret: %s", secretName)
			}

			c.Users = append(c.Users, discovery.User{
				Username: user,
				Password: string(password),
			})
		}

		for database := range cluster.Spec.Databases {
			c.Databases = append(c.Databases, database)
		}

		res = append(res, c)
	}

	return res, nil
}

func (T *Discoverer) Added() <-chan discovery.Cluster {
	return nil // TODO(garet)
}

func (T *Discoverer) Updated() <-chan discovery.Cluster {
	return nil // TODO(garet)
}

func (T *Discoverer) Removed() <-chan string {
	return nil // TODO(garet)
}

var _ discovery.Discoverer = (*Discoverer)(nil)
