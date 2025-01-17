package v1alpha1

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-k8s/api/common"
	capi "github.com/hashicorp/consul/api"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type MeshGatewayMode string

// Expose describes HTTP paths to expose through Envoy outside of Connect.
// Users can expose individual paths and/or all HTTP/GRPC paths for checks.
type Expose struct {
	// Checks defines whether paths associated with Consul checks will be exposed.
	// This flag triggers exposing all HTTP and GRPC check paths registered for the service.
	Checks bool `json:"checks,omitempty"`

	// Paths is the list of paths exposed through the proxy.
	Paths []ExposePath `json:"paths,omitempty"`
}

type ExposePath struct {
	// ListenerPort defines the port of the proxy's listener for exposed paths.
	ListenerPort int `json:"listenerPort,omitempty"`

	// Path is the path to expose through the proxy, ie. "/metrics".
	Path string `json:"path,omitempty"`

	// LocalPathPort is the port that the service is listening on for the given path.
	LocalPathPort int `json:"localPathPort,omitempty"`

	// Protocol describes the upstream's service protocol.
	// Valid values are "http" and "http2", defaults to "http".
	Protocol string `json:"protocol,omitempty"`
}

type TransparentProxy struct {
	// The port of the listener where outbound application traffic is being redirected to.
	OutboundListenerPort int `json:"outboundListenerPort,omitempty"`
}

// MeshGateway controls how Mesh Gateways are used for upstream Connect
// services
type MeshGateway struct {
	// Mode is the mode that should be used for the upstream connection.
	// One of none, local, or remote.
	Mode string `json:"mode,omitempty"`
}

type ProxyMode string

func (in MeshGateway) toConsul() capi.MeshGatewayConfig {
	mode := capi.MeshGatewayMode(in.Mode)
	switch mode {
	case capi.MeshGatewayModeLocal, capi.MeshGatewayModeRemote, capi.MeshGatewayModeNone:
		return capi.MeshGatewayConfig{
			Mode: mode,
		}
	default:
		return capi.MeshGatewayConfig{
			Mode: capi.MeshGatewayModeDefault,
		}
	}
}

func (in MeshGateway) validate(path *field.Path) *field.Error {
	modes := []string{"remote", "local", "none", ""}
	if !sliceContains(modes, in.Mode) {
		return field.Invalid(path.Child("mode"), in.Mode, notInSliceMessage(modes))
	}
	return nil
}

func (in Expose) toConsul() capi.ExposeConfig {
	var paths []capi.ExposePath
	for _, path := range in.Paths {
		paths = append(paths, capi.ExposePath{
			ListenerPort:  path.ListenerPort,
			Path:          path.Path,
			LocalPathPort: path.LocalPathPort,
			Protocol:      path.Protocol,
		})
	}
	return capi.ExposeConfig{
		Checks: in.Checks,
		Paths:  paths,
	}
}

func (in Expose) validate(path *field.Path) field.ErrorList {
	var errs field.ErrorList
	protocols := []string{"http", "http2"}
	for i, pathCfg := range in.Paths {
		indexPath := path.Child("paths").Index(i)
		if invalidPathPrefix(pathCfg.Path) {
			errs = append(errs, field.Invalid(
				indexPath.Child("path"),
				pathCfg.Path,
				`must begin with a '/'`))
		}
		if pathCfg.Protocol != "" && !sliceContains(protocols, pathCfg.Protocol) {
			errs = append(errs, field.Invalid(
				indexPath.Child("protocol"),
				pathCfg.Protocol,
				notInSliceMessage(protocols)))
		}
	}
	return errs
}

func (in *TransparentProxy) toConsul() *capi.TransparentProxyConfig {
	if in == nil {
		return &capi.TransparentProxyConfig{OutboundListenerPort: 0}
	}
	return &capi.TransparentProxyConfig{OutboundListenerPort: in.OutboundListenerPort}
}

func (in *TransparentProxy) validate(path *field.Path) *field.Error {
	if in != nil {
		return field.Invalid(path, in, "use the annotation `consul.hashicorp.com/transparent-proxy-outbound-listener-port` to configure the Outbound Listener Port")
	}
	return nil
}

func (in *ProxyMode) validate(path *field.Path) *field.Error {
	if in != nil {
		return field.Invalid(path, in, "use the annotation `consul.hashicorp.com/transparent-proxy` to configure the Transparent Proxy Mode")
	}
	return nil
}

func notInSliceMessage(slice []string) string {
	return fmt.Sprintf(`must be one of "%s"`, strings.Join(slice, `", "`))
}

func sliceContains(slice []string, entry string) bool {
	for _, s := range slice {
		if entry == s {
			return true
		}
	}
	return false
}

func invalidPathPrefix(path string) bool {
	return path != "" && !strings.HasPrefix(path, "/")
}

func meta(datacenter string) map[string]string {
	return map[string]string{
		common.SourceKey:     common.SourceValue,
		common.DatacenterKey: datacenter,
	}
}
