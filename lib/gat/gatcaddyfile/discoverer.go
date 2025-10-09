package gatcaddyfile

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/cloudnative_pg"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/digitalocean"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/google_cloud_sql"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/zalando_operator"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"strconv"
)

func init() {
	RegisterDirective(Discoverer, "digitalocean", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := digitalocean.Discoverer{}

		if d.NextArg() {
			module.APIKey = d.Val()
		} else {
			if !d.NextBlock(d.Nesting()) {
				return nil, d.ArgErr()
			}

			for d.Val() != "}" {

				directive := d.Val()
				switch directive {
				case "token":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.APIKey = d.Val()
				case "private":
					if d.NextArg() {
						switch d.Val() {
						case "true":
							module.Private = true
						case "false":
							module.Private = false
						default:
							return nil, d.ArgErr()
						}
					} else {
						module.Private = true
					}
				case "filter":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.Filter, err = UnmarshalDirectiveJSONModuleObject(
						d,
						DigitaloceanFilter,
						"filter",
						warnings,
					)
					if err != nil {
						return nil, err
					}
				default:
					return nil, d.ArgErr()
				}

				if !d.NextLine() {
					return nil, d.EOFErr()
				}
			}
		}

		return &module, nil
	})
	RegisterDirective(Discoverer, "google_cloud_sql", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := google_cloud_sql.Discoverer{
			Config: google_cloud_sql.Config{
				IpAddressType: "PRIMARY",
			},
		}

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.Project = d.Val()

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.AuthUser = d.Val()

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.AuthPassword = d.Val()

		return &module, nil
	})
	RegisterDirective(Discoverer, "zalando_operator", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := zalando_operator.Discoverer{
			Config: zalando_operator.Config{
				Namespace: "default",
			},
		}

		if d.NextArg() {
			module.OperatorConfigurationObject = d.Val()
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			directive := d.Val()
			switch directive {
			case "namespace":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.Namespace = d.Val()
			case "operator_configuration_object":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.OperatorConfigurationObject = d.Val()
			case "config_map_name":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.ConfigMapName = d.Val()
			case "config":
				{
					for nesting := d.Nesting(); d.NextBlock(nesting); {
						directive := d.Val()
						switch directive {
						case "cluster_domain":
							if !d.NextArg() {
								return nil, d.ArgErr()
							}
							module.ClusterDomain = d.Val()
						case "secret_name_template":
							if !d.NextArg() {
								return nil, d.ArgErr()
							}
							module.SecretNameTemplate = d.Val()
						case "connection_pooler_number_of_instances":
							if !d.NextArg() {
								return nil, d.ArgErr()
							}
							val, err := strconv.ParseInt(d.Val(), 10, 32)
							if err != nil {
								return nil, fmt.Errorf("error parsing connection_pooler_number_of_instances: %v", err)
							}
							i32 := int32(val)
							module.ConnectionPoolerNumberOfInstances = &i32
						case "connection_pooler_mode":
							if !d.NextArg() {
								return nil, d.ArgErr()
							}
							module.ConnectionPoolerMode = d.Val()
						case "connection_pooler_max_db_connections":
							if !d.NextArg() {
								return nil, d.ArgErr()
							}
							val, err := strconv.ParseInt(d.Val(), 10, 32)
							if err != nil {
								return nil, fmt.Errorf("error parsing connection_pooler_max_db_instances: %v", err)
							}
							i32 := int32(val)
							module.ConnectionPoolerMaxDBConnections = &i32
						default:
							return nil, d.ArgErr()
						}
					}
				}
			default:
				return nil, d.ArgErr()
			}
		}

		return &module, nil
	})
	RegisterDirective(Discoverer, "cloudnative_pg", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := cloudnative_pg.Discoverer{
			Config: cloudnative_pg.Config{
				ClusterDomain:          "cluster.local",
				ReadWriteServiceSuffix: "-rw",
				ReadOnlyServiceSuffix:  "-ro",
				Port:                   5432,
				SecretSuffix:           "-app",
			},
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			directive := d.Val()
			switch directive {
			case "namespace":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.Namespace = d.Val()
			case "cluster_domain":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.ClusterDomain = d.Val()
			case "read_write_service_suffix":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.ReadWriteServiceSuffix = d.Val()
			case "read_only_service_suffix":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.ReadOnlyServiceSuffix = d.Val()
			case "port":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				port, err := strconv.Atoi(d.Val())
				if err != nil {
					return nil, fmt.Errorf("error parsing port: %v", err)
				}
				module.Port = port
			case "secret_suffix":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}
				module.SecretSuffix = d.Val()
			case "include_superuser":
				if d.NextArg() {
					switch d.Val() {
					case "true":
						module.IncludeSuperuser = true
					case "false":
						module.IncludeSuperuser = false
					default:
						return nil, d.ArgErr()
					}
				} else {
					module.IncludeSuperuser = true
				}
			default:
				return nil, d.ArgErr()
			}
		}

		return &module, nil
	})
}
