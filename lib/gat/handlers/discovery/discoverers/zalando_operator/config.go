package zalando_operator

type Config struct {
	Namespace                         string `json:"namespace"`
	ConfigMapName                     string `json:"config_map_name"`
	OperatorConfigurationObject       string `json:"operator_configuration_object"`
	ClusterDomain                     string `json:"cluster_domain"`
	SecretNameTemplate                string `json:"secret_name_template"`
	ConnectionPoolerNumberOfInstances *int32 `json:"connection_pooler_number_of_instances"`
	ConnectionPoolerMode              string `json:"connection_pooler_mode"`
	ConnectionPoolerMaxDBConnections  *int32 `json:"connection_pooler_max_db_connections"`
}
