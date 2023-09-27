package caddy

type Server struct {
	Listen  []ServerSlug   `json:"listen,omitempty"`
	Modules []ServerModule `json:"modules,omitempty"`
}
