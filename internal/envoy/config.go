package envoy

import (
	"fmt"
	"sort"

	routingv1alpha1 "github.com/seu-user/header-route-controller/api/v1alpha1"
)

// Config represents the Envoy configuration structure
type Config struct {
	StaticResources StaticResources `json:"static_resources"`
}

type StaticResources struct {
	Listeners []Listener `json:"listeners"`
	Clusters  []Cluster  `json:"clusters"`
}

type Listener struct {
	Name    string  `json:"name"`
	Address Address `json:"address"`
	FilterChains []FilterChain `json:"filter_chains"`
}

type Address struct {
	SocketAddress SocketAddress `json:"socket_address"`
}

type SocketAddress struct {
	Address   string `json:"address"`
	PortValue int    `json:"port_value"`
}

type FilterChain struct {
	Filters []Filter `json:"filters"`
}

type Filter struct {
	Name        string      `json:"name"`
	TypedConfig TypedConfig `json:"typed_config"`
}

type TypedConfig struct {
	Type        string      `json:"@type"`
	StatPrefix  string      `json:"stat_prefix"`
	CodecType   string      `json:"codec_type,omitempty"`
	RouteConfig RouteConfig `json:"route_config"`
	HttpFilters []HttpFilter `json:"http_filters"`
}

type RouteConfig struct {
	Name         string        `json:"name"`
	VirtualHosts []VirtualHost `json:"virtual_hosts"`
}

type VirtualHost struct {
	Name    string   `json:"name"`
	Domains []string `json:"domains"`
	Routes  []Route  `json:"routes"`
}

type Route struct {
	Match RouteMatch `json:"match"`
	Route RouteAction `json:"route,omitempty"`
	DirectResponse *DirectResponse `json:"direct_response,omitempty"`
}

type RouteMatch struct {
	Prefix  string        `json:"prefix"`
	Headers []HeaderMatch `json:"headers,omitempty"`
}

type HeaderMatch struct {
	Name       string `json:"name"`
	ExactMatch string `json:"exact_match"`
}

type RouteAction struct {
	Cluster string `json:"cluster"`
}

type DirectResponse struct {
	Status int    `json:"status"`
	Body   Body   `json:"body"`
}

type Body struct {
	InlineString string `json:"inline_string"`
}

type HttpFilter struct {
	Name        string            `json:"name"`
	TypedConfig HttpFilterConfig  `json:"typed_config,omitempty"`
}

type HttpFilterConfig struct {
	Type string `json:"@type"`
}

type Cluster struct {
	Name           string         `json:"name"`
	ConnectTimeout string         `json:"connect_timeout"`
	Type           string         `json:"type"`
	LbPolicy       string         `json:"lb_policy"`
	LoadAssignment LoadAssignment `json:"load_assignment"`
}

type LoadAssignment struct {
	ClusterName string     `json:"cluster_name"`
	Endpoints   []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	LbEndpoints []LbEndpoint `json:"lb_endpoints"`
}

type LbEndpoint struct {
	Endpoint EndpointAddress `json:"endpoint"`
}

type EndpointAddress struct {
	Address Address `json:"address"`
}

// DefaultBackend represents the fallback service configuration
type DefaultBackend struct {
	ServiceName string
	Namespace   string
	Port        int32
}

// GenerateEnvoyConfig generates Envoy configuration from HeaderRoute resources
func GenerateEnvoyConfig(routes []routingv1alpha1.HeaderRoute, defaultBackend *DefaultBackend) Config {
	// Sort routes by priority (higher priority first)
	sortedRoutes := make([]routingv1alpha1.HeaderRoute, len(routes))
	copy(sortedRoutes, routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].Spec.Priority > sortedRoutes[j].Spec.Priority
	})

	// Build routes and collect unique clusters
	envoyRoutes := make([]Route, 0, len(sortedRoutes)+1)
	clusters := make([]Cluster, 0)
	clusterSet := make(map[string]bool)

	for _, hr := range sortedRoutes {
		clusterName := generateClusterName(hr.Spec.Backend.Name, hr.Spec.Backend.Namespace, hr.Namespace)
		
		// Add route
		envoyRoutes = append(envoyRoutes, Route{
			Match: RouteMatch{
				Prefix: "/",
				Headers: []HeaderMatch{
					{
						Name:       hr.Spec.HeaderName,
						ExactMatch: hr.Spec.HeaderValue,
					},
				},
			},
			Route: RouteAction{
				Cluster: clusterName,
			},
		})

		// Add cluster if not already added
		if !clusterSet[clusterName] {
			clusterSet[clusterName] = true
			namespace := hr.Spec.Backend.Namespace
			if namespace == "" {
				namespace = hr.Namespace
			}
			clusters = append(clusters, createCluster(
				clusterName,
				hr.Spec.Backend.Name,
				namespace,
				hr.Spec.Backend.Port,
			))
		}
	}

	// Add default backend route (fallback / 404)
	if defaultBackend != nil {
		defaultClusterName := generateClusterName(defaultBackend.ServiceName, defaultBackend.Namespace, defaultBackend.Namespace)
		
		envoyRoutes = append(envoyRoutes, Route{
			Match: RouteMatch{
				Prefix: "/",
			},
			Route: RouteAction{
				Cluster: defaultClusterName,
			},
		})

		if !clusterSet[defaultClusterName] {
			clusters = append(clusters, createCluster(
				defaultClusterName,
				defaultBackend.ServiceName,
				defaultBackend.Namespace,
				defaultBackend.Port,
			))
		}
	} else {
		// If no default backend, return 404
		envoyRoutes = append(envoyRoutes, Route{
			Match: RouteMatch{
				Prefix: "/",
			},
			DirectResponse: &DirectResponse{
				Status: 404,
				Body: Body{
					InlineString: `{"error": "no matching route", "message": "No HeaderRoute rule matched the request"}`,
				},
			},
		})
	}

	return Config{
		StaticResources: StaticResources{
			Listeners: []Listener{
				{
					Name: "listener_0",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   "0.0.0.0",
							PortValue: 8080,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []Filter{
								{
									Name: "envoy.filters.network.http_connection_manager",
									TypedConfig: TypedConfig{
										Type:       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
										StatPrefix: "ingress_http",
										CodecType:  "AUTO",
										RouteConfig: RouteConfig{
											Name: "local_route",
											VirtualHosts: []VirtualHost{
												{
													Name:    "backend",
													Domains: []string{"*"},
													Routes:  envoyRoutes,
												},
											},
										},
										HttpFilters: []HttpFilter{
											{
												Name: "envoy.filters.http.router",
												TypedConfig: HttpFilterConfig{
													Type: "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Clusters: clusters,
		},
	}
}

func generateClusterName(serviceName, namespace, defaultNamespace string) string {
	ns := namespace
	if ns == "" {
		ns = defaultNamespace
	}
	return fmt.Sprintf("%s_%s", ns, serviceName)
}

func createCluster(name, serviceName, namespace string, port int32) Cluster {
	// Use Kubernetes DNS for service discovery
	serviceHost := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)
	
	return Cluster{
		Name:           name,
		ConnectTimeout: "5s",
		Type:           "STRICT_DNS",
		LbPolicy:       "ROUND_ROBIN",
		LoadAssignment: LoadAssignment{
			ClusterName: name,
			Endpoints: []Endpoint{
				{
					LbEndpoints: []LbEndpoint{
						{
							Endpoint: EndpointAddress{
								Address: Address{
									SocketAddress: SocketAddress{
										Address:   serviceHost,
										PortValue: int(port),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
