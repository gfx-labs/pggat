package digitalocean_discovery

import (
	"time"

	"github.com/google/uuid"
)

type Connection struct {
	URI      string `env:"uri"`
	Database string `env:"database"`
	Host     string `env:"host"`
	Port     int    `env:"port"`
	User     string `env:"user"`
	Password string `env:"password"`
	SSL      bool   `env:"ssl"`
}

type User struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type MaintenanceWindow struct {
	Day         string   `json:"day"`
	Hour        string   `json:"hour"`
	Pending     bool     `json:"pending"`
	Description []string `json:"description"`
}

type Database struct {
	ID                       uuid.UUID  `json:"id"`
	Name                     string     `json:"name"`
	Engine                   string     `json:"engine"`
	Version                  string     `json:"version"`
	Connection               Connection `json:"connection"`
	PrivateConnection        Connection `json:"private_connection"`
	Users                    []User     `json:"users"`
	DBNames                  []string   `json:"db_names"`
	NumNodes                 int        `json:"num_nodes"`
	Region                   string     `json:"region"`
	Status                   string     `json:"online"`
	CreatedAt                time.Time  `json:"created_at"`
	Size                     string     `json:"size"`
	Tags                     []string   `json:"tags"`
	PrivateNetworkUUID       uuid.UUID  `json:"private_network_uuid"`
	VersionEndOfLife         time.Time  `json:"version_end_of_life"`
	VersionEndOfAvailability time.Time  `json:"version_end_of_availability"`
}

type ListClustersResponse struct {
	Databases []Database `json:"databases"`
}
