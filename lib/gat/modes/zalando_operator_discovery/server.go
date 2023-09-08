package zalando_operator_discovery

import (
	"context"
	"crypto/tls"

	acidzalando "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do"
	v1acid "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	acidv1informer "github.com/zalando/postgres-operator/pkg/generated/informers/externalversions/acid.zalan.do/v1"
	"github.com/zalando/postgres-operator/pkg/util/constants"
	"github.com/zalando/postgres-operator/pkg/util/k8sutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/util/flip"
	"pggat/lib/util/strutil"
)

type mapKey struct {
	User     string
	Database string
}

type Server struct {
	config *Config

	k8s k8sutil.KubernetesClient

	postgresqlInformer cache.SharedIndexInformer

	pools gat.PoolsMap
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
	T.k8s, err = k8sutil.NewFromConfig(T.config.Rest)
	if err != nil {
		return err
	}

	T.postgresqlInformer = acidv1informer.NewPostgresqlInformer(
		T.k8s.AcidV1ClientSet,
		T.config.Namespace,
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
		log.Print("removed users: ", oldPsql.Spec.Users)
	}
	if newPsql != nil {
		log.Print("added databases: ", newPsql.Spec.Databases)
		log.Print("added users: ", newPsql.Spec.Users)
		for user := range newPsql.Spec.Users {
			// TODO(garet) lookup config map to get this format (what a pain)
			secretName := user + "." + newPsql.Name + ".credentials." + v1acid.PostgresCRDResourceKind + "." + acidzalando.GroupName

			secret, err := T.k8s.Secrets(T.config.Namespace).Get(context.Background(), secretName, v1.GetOptions{})
			if err != nil {
				log.Printf("error getting secret: %v", err)
				return
			}

			password, ok := secret.Data["password"]
			if !ok {
				log.Println("failed to get password in secret :(")
				return
			}

			creds := credentials.Cleartext{
				Username: user,
				Password: string(password),
			}

			for database := range newPsql.Spec.Databases {
				d := dialer.Net{
					Network: "tcp",
					Address: newPsql.Name + "." + T.config.Namespace + ".svc.cluster.local:5432", // TODO(garet) lookup port from config map
					AcceptOptions: backends.AcceptOptions{
						SSLMode: bouncer.SSLModePrefer,
						SSLConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
						Credentials: creds,
						Database:    database,
					},
				}

				poolOptions := pool.Options{
					Credentials: creds,
				}
				p := transaction.NewPool(poolOptions)

				recipeOptions := recipe.Options{
					Dialer: d,
				}
				r := recipe.NewRecipe(recipeOptions)

				p.AddRecipe("service", r)

				T.pools.Add(user, database, p)
			}
		}
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

	bank.Queue(func() error {
		listen := ":5432" // TODO(garet) use port

		log.Printf("listening on %s", listen)

		return gat.ListenAndServe("tcp", listen, frontends.AcceptOptions{
			AllowedStartupOptions: []strutil.CIString{
				strutil.MakeCIString("client_encoding"),
				strutil.MakeCIString("datestyle"),
				strutil.MakeCIString("timezone"),
				strutil.MakeCIString("standard_conforming_strings"),
				strutil.MakeCIString("application_name"),
			},
			// TODO(garet)
		}, &T.pools)
	})

	return bank.Wait()
}
