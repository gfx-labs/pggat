package eddy

import (
	v1acid "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	acidv1informer "github.com/zalando/postgres-operator/pkg/generated/informers/externalversions/acid.zalan.do/v1"
	"github.com/zalando/postgres-operator/pkg/util/constants"
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
	"k8s.io/client-go/tools/cache"
	"tuxpa.in/a/zlog/log"

	"pggat2/lib/util/flip"
)

type Server struct {
	config *Config

	k8s k8sutil.KubernetesClient

	postgresqlInformer cache.SharedIndexInformer
}

func NewServer(config *Config) (*Server, error) {
	srv := &Server{
		config: config,
	}
	if err := srv.init(); err != nil {
		return nil, err
	}
	return srv, nil
}

func (T *Server) init() error {
	var err error
	T.k8s, err = k8sutil.NewFromConfig(T.config.rest)
	if err != nil {
		return err
	}

	T.postgresqlInformer = acidv1informer.NewPostgresqlInformer(
		T.k8s.AcidV1ClientSet,
		"default",
		constants.QueueResyncPeriodTPR,
		cache.Indexers{})

	_, err = T.postgresqlInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			psql, ok := obj.(*v1acid.Postgresql)
			if !ok {
				return
			}
			T.addPostgresql(psql)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPsql, ok := oldObj.(*v1acid.Postgresql)
			if !ok {
				return
			}
			newPsql, ok := newObj.(*v1acid.Postgresql)
			if !ok {
				return
			}
			T.updatePostgresql(oldPsql, newPsql)
		},
		DeleteFunc: func(obj interface{}) {
			psql, ok := obj.(*v1acid.Postgresql)
			if !ok {
				return
			}
			T.deletePostgresql(psql)
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (T *Server) addPostgresql(psql *v1acid.Postgresql) {
	T.updatePostgresql(nil, psql)
}

func (T *Server) updatePostgresql(oldPsql *v1acid.Postgresql, newPsql *v1acid.Postgresql) {
	if oldPsql != nil {
		log.Print("removed databases: ", oldPsql.Spec.Databases)
		log.Print("removed users: ", newPsql.Spec.Users)
	}
	if newPsql != nil {
		log.Print("added databases: ", newPsql.Spec.Databases)
		log.Print("added users: ", newPsql.Spec.Users)
	}
}

func (T *Server) deletePostgresql(psql *v1acid.Postgresql) {
	T.updatePostgresql(psql, nil)
}

func (T *Server) ListenAndServe() error {
	var bank flip.Bank

	bank.Queue(func() error {
		T.postgresqlInformer.Run(make(chan struct{}))
		return nil
	})

	return bank.Wait()
}
