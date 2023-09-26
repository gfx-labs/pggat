package caddy

type Server struct {
	Listen []ServerSlug `json:"listen,omitempty"`
}
