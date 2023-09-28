package zalando_operator

type Config struct {
	Namespace                   string `json:"namespace"`
	ConfigMapName               string `json:"config_map_name"`
	OperatorConfigurationObject string `json:"operator_configuration_object"`
}
