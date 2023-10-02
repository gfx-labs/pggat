package google_cloud_sql

type Config struct {
	Project       string `json:"project"`
	IpAddressType string `json:"ip_address_type"`
	AuthUser      string `json:"auth_user"`
	AuthPassword  string `json:"auth_password"`
}
