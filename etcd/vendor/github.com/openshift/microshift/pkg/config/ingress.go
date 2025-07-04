package config

import (
	"fmt"
	"net"
	"regexp"
	"slices"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NamespaceOwnershipStrict  NamespaceOwnershipEnum   = "Strict"
	NamespaceOwnershipAllowed NamespaceOwnershipEnum   = "InterNamespaceAllowed"
	StatusManaged             IngressStatusEnum        = "Managed"
	StatusRemoved             IngressStatusEnum        = "Removed"
	DefaultHttpVersionV1      DefaultHttpVersionPolicy = 1
	DefaultHttpVersionV2      DefaultHttpVersionPolicy = 2
	WildcardPolicyAllowed     WildcardPolicy           = "WildcardsAllowed"
	WildcardPolicyDisallowed  WildcardPolicy           = "WildcardsDisallowed"

	AccessLoggingEnabled  AccessLoggingStatusEnum = "Enabled"
	AccessLoggingDisabled AccessLoggingStatusEnum = "Disabled"
)

var (
	headerNamePattern = regexp.MustCompile(`^[-!#$%&'*+.0-9A-Z^_` + "`" + `a-z|~]+$`)
	allowedFacilities = []string{
		"kern", "user", "mail", "daemon", "auth", "syslog", "lpr", "news",
		"uucp", "cron", "auth2", "ftp", "ntp", "audit", "alert", "cron2",
		"local0", "local1", "local2", "local3", "local4", "local5", "local6",
		"local7",
	}
)

type AccessLoggingStatusEnum string

type NamespaceOwnershipEnum string
type IngressStatusEnum string
type DefaultHttpVersionPolicy int32
type WildcardPolicy string

type IngressConfig struct {
	// Default router status, can be Managed or Removed.
	// +kubebuilder:default=Managed
	Status          IngressStatusEnum              `json:"status"`
	AdmissionPolicy RouteAdmissionPolicy           `json:"routeAdmissionPolicy"`
	Ports           IngressPortsConfig             `json:"ports"`
	TuningOptions   IngressControllerTuningOptions `json:"tuningOptions"`
	// List of IP addresses and NIC names where the router will be listening. The NIC
	// names get translated to all their configured IPs dynamically. Defaults to the
	// configured IPs in the host at MicroShift start.
	ListenAddress      []string `json:"listenAddress"`
	ServingCertificate []byte   `json:"-"`
	ServingKey         []byte   `json:"-"`

	// ServingCertificateSecret references a kubernetes.io/tls type secret containing the TLS cert info for serving secure traffic.
	// The secret must exist in the openshift-ingress namespace and contain the following required fields:
	// - Secret.Data["tls.key"] - TLS private key.
	// - Secret.Data["tls.crt"] - TLS certificate.
	//
	// +optional
	// +kubebuilder:default:="router-certs-default"
	ServingCertificateSecret string `json:"certificateSecret"`

	// logEmptyRequests specifies how connections on which no request is
	// received should be logged.  Typically, these empty requests come from
	// load balancers' health probes or Web browsers' speculative
	// connections ("preconnect"), in which case logging these requests may
	// be undesirable.  However, these requests may also be caused by
	// network errors, in which case logging empty requests may be useful
	// for diagnosing the errors.  In addition, these requests may be caused
	// by port scans, in which case logging empty requests may aid in
	// detecting intrusion attempts.  Allowed values for this field are
	// "Log" and "Ignore".  The default value is "Log".
	//
	// +optional
	// +kubebuilder:default:="Log"
	LogEmptyRequests operatorv1.LoggingPolicy `json:"logEmptyRequests,omitempty"`

	// forwardedHeaderPolicy specifies when and how ingress router
	// sets the Forwarded, X-Forwarded-For, X-Forwarded-Host,
	// X-Forwarded-Port, X-Forwarded-Proto, and X-Forwarded-Proto-Version
	// HTTP headers.  The value may be one of the following:
	//
	// * "Append", which specifies that ingress router appends the
	//   headers, preserving existing headers.
	//
	// * "Replace", which specifies that ingress router sets the
	//   headers, replacing any existing Forwarded or X-Forwarded-* headers.
	//
	// * "IfNone", which specifies that ingress router sets the
	//   headers if they are not already set.
	//
	// * "Never", which specifies that ingress router never sets the
	//   headers, preserving any existing headers.
	//
	// By default, the policy is "Append".
	//
	// +optional
	ForwardedHeaderPolicy operatorv1.IngressControllerHTTPHeaderPolicy `json:"forwardedHeaderPolicy,omitempty"`

	// httpEmptyRequestsPolicy describes how HTTP connections should be
	// handled if the connection times out before a request is received.
	// Allowed values for this field are "Respond" and "Ignore".  If the
	// field is set to "Respond", the ingress controller sends an HTTP 400
	// or 408 response, logs the connection (if access logging is enabled),
	// and counts the connection in the appropriate metrics.  If the field
	// is set to "Ignore", the ingress controller closes the connection
	// without sending a response, logging the connection, or incrementing
	// metrics.  The default value is "Respond".
	//
	// Typically, these connections come from load balancers' health probes
	// or Web browsers' speculative connections ("preconnect") and can be
	// safely ignored.  However, these requests may also be caused by
	// network errors, and so setting this field to "Ignore" may impede
	// detection and diagnosis of problems.  In addition, these requests may
	// be caused by port scans, in which case logging empty requests may aid
	// in detecting intrusion attempts.
	//
	// +optional
	// +kubebuilder:default:="Respond"
	HTTPEmptyRequestsPolicy operatorv1.HTTPEmptyRequestsPolicy `json:"httpEmptyRequestsPolicy,omitempty"`

	// httpCompression defines a policy for HTTP traffic compression.
	// By default, there is no HTTP compression.
	//
	// +optional
	HTTPCompressionPolicy operatorv1.HTTPCompressionPolicy `json:"httpCompression,omitempty"`

	// Determines default http version should be used for the ingress backends
	// By default,  using version 1.
	//
	// +optional
	// +kubebuilder:default:="1"
	DefaultHttpVersionPolicy DefaultHttpVersionPolicy `json:"defaultHTTPVersion,omitempty"`

	// tlsSecurityProfile specifies settings for TLS connections for ingresscontrollers.
	//
	// If unset, the default is based on the apiservers.config.openshift.io/cluster resource.
	//
	// Note that when using the Old, Intermediate, and Modern profile types, the effective
	// profile configuration is subject to change between releases. For example, given
	// a specification to use the Intermediate profile deployed on release X.Y.Z, an upgrade
	// to release X.Y.Z+1 may cause a new profile configuration to be applied to the ingress
	// controller, resulting in a rollout.
	//
	// +optional
	TLSSecurityProfile *configv1.TLSSecurityProfile `json:"tlsSecurityProfile,omitempty"`

	// clientTLS specifies settings for requesting and verifying client
	// certificates, which can be used to enable mutual TLS for
	// edge-terminated and reencrypt routes.
	//
	// +optional
	ClientTLS operatorv1.ClientTLS `json:"clientTLS,omitempty"`

	// httpErrorCodePages specifies a configmap with custom error pages.
	// The administrator must create this configmap in the openshift-config namespace.
	// This configmap should have keys in the format "error-page-<error code>.http",
	// where <error code> is an HTTP error code.
	// For example, "error-page-503.http" defines an error page for HTTP 503 responses.
	// Currently only error pages for 503 and 404 responses can be customized.
	// Each value in the configmap should be the full response, including HTTP headers.
	// Eg- https://raw.githubusercontent.com/openshift/router/fadab45747a9b30cc3f0a4b41ad2871f95827a93/images/router/haproxy/conf/error-page-503.http
	// If this field is empty, the ingress controller uses the default error pages.
	// +kubebuilder:validation:Optional
	HttpErrorCodePages configv1.ConfigMapNameReference `json:"httpErrorCodePages,omitempty"`

	// accessLogging describes how the client requests should be logged.
	AccessLogging AccessLogging `json:"accessLogging,omitempty"`
}

// IngressControllerTuningOptions specifies options for tuning the performance
// of ingress controller pods
type IngressControllerTuningOptions struct {
	// headerBufferBytes describes how much memory should be reserved
	// (in bytes) for IngressController connection sessions.
	// Note that this value must be at least 16384 if HTTP/2 is
	// enabled for the IngressController (https://tools.ietf.org/html/rfc7540).
	// If this field is empty, the IngressController will use a default value
	// of 32768 bytes.
	//
	// Setting this field is generally not recommended as headerBufferBytes
	// values that are too small may break the IngressController and
	// headerBufferBytes values that are too large could cause the
	// IngressController to use significantly more memory than necessary.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=16384
	// +optional
	HeaderBufferBytes int32 `json:"headerBufferBytes,omitempty"`

	// headerBufferMaxRewriteBytes describes how much memory should be reserved
	// (in bytes) from headerBufferBytes for HTTP header rewriting
	// and appending for IngressController connection sessions.
	// Note that incoming HTTP requests will be limited to
	// (headerBufferBytes - headerBufferMaxRewriteBytes) bytes, meaning
	// headerBufferBytes must be greater than headerBufferMaxRewriteBytes.
	// If this field is empty, the IngressController will use a default value
	// of 8192 bytes.
	//
	// Setting this field is generally not recommended as
	// headerBufferMaxRewriteBytes values that are too small may break the
	// IngressController and headerBufferMaxRewriteBytes values that are too
	// large could cause the IngressController to use significantly more memory
	// than necessary.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=4096
	// +optional
	HeaderBufferMaxRewriteBytes int32 `json:"headerBufferMaxRewriteBytes,omitempty"`

	// threadCount defines the number of threads created per HAProxy process.
	// Creating more threads allows each ingress controller pod to handle more
	// connections, at the cost of more system resources being used. HAProxy
	// currently supports up to 64 threads. If this field is empty, the
	// IngressController will use the default value.  The current default is 4
	// threads, but this may change in future releases.
	//
	// Setting this field is generally not recommended. Increasing the number
	// of HAProxy threads allows ingress controller pods to utilize more CPU
	// time under load, potentially starving other pods if set too high.
	// Reducing the number of threads may cause the ingress controller to
	// perform poorly.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=64
	// +optional
	ThreadCount int32 `json:"threadCount,omitempty"`

	// clientTimeout defines how long a connection will be held open while
	// waiting for a client response.
	//
	// If unset, the default timeout is 30s
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="30s"
	// +optional
	ClientTimeout *metav1.Duration `json:"clientTimeout,omitempty"`

	// clientFinTimeout defines how long a connection will be held open while
	// waiting for the client response to the server/backend closing the
	// connection.
	//
	// If unset, the default timeout is 1s
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="1s"
	// +optional
	ClientFinTimeout *metav1.Duration `json:"clientFinTimeout,omitempty"`

	// serverTimeout defines how long a connection will be held open while
	// waiting for a server/backend response.
	//
	// If unset, the default timeout is 30s
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="30s"
	// +optional
	ServerTimeout *metav1.Duration `json:"serverTimeout,omitempty"`

	// serverFinTimeout defines how long a connection will be held open while
	// waiting for the server/backend response to the client closing the
	// connection.
	//
	// If unset, the default timeout is 1s
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="1s"
	// +optional
	ServerFinTimeout *metav1.Duration `json:"serverFinTimeout,omitempty"`

	// tunnelTimeout defines how long a tunnel connection (including
	// websockets) will be held open while the tunnel is idle.
	//
	// If unset, the default timeout is 1h
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="1h"
	// +optional
	TunnelTimeout *metav1.Duration `json:"tunnelTimeout,omitempty"`

	// tlsInspectDelay defines how long the router can hold data to find a
	// matching route.
	//
	// Setting this too short can cause the router to fall back to the default
	// certificate for edge-terminated or reencrypt routes even when a better
	// matching certificate could be used.
	//
	// If unset, the default inspect delay is 5s
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="5s"
	// +optional
	TLSInspectDelay *metav1.Duration `json:"tlsInspectDelay,omitempty"`

	// healthCheckInterval defines how long the router waits between two consecutive
	// health checks on its configured backends.  This value is applied globally as
	// a default for all routes, but may be overridden per-route by the route annotation
	// "router.openshift.io/haproxy.health.check.interval".
	//
	// Expects an unsigned duration string of decimal numbers, each with optional
	// fraction and a unit suffix, eg "300ms", "1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs" U+00B5 or "μs" U+03BC), "ms", "s", "m", "h".
	//
	// Setting this to less than 5s can cause excess traffic due to too frequent
	// TCP health checks and accompanying SYN packet storms.  Alternatively, setting
	// this too high can result in increased latency, due to backend servers that are no
	// longer available, but haven't yet been detected as such.
	//
	// An empty or zero healthCheckInterval means no opinion and IngressController chooses
	// a default, which is subject to change over time.
	// Currently the default healthCheckInterval value is 5s.
	//
	// Currently the minimum allowed value is 1s and the maximum allowed value is
	// 2147483647ms (24.85 days).  Both are subject to change over time.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^(0|([0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h))+)$
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:default:="5s"
	// +optional
	HealthCheckInterval *metav1.Duration `json:"healthCheckInterval,omitempty"`

	// maxConnections defines the maximum number of simultaneous
	// connections that can be established per HAProxy process.
	// Increasing this value allows each ingress controller pod to
	// handle more connections but at the cost of additional
	// system resources being consumed.
	//
	// Permitted values are: empty, 0, -1, and the range
	// 2000-2000000.
	//
	// If this field is empty or 0, the IngressController will use
	// the default value of 50000, but the default is subject to
	// change in future releases.
	//
	// If the value is -1 then HAProxy will dynamically compute a
	// maximum value based on the available ulimits in the running
	// container. Selecting -1 (i.e., auto) will result in a large
	// value being computed (~520000 on OpenShift >=4.10 clusters)
	// and therefore each HAProxy process will incur significant
	// memory usage compared to the current default of 50000.
	//
	// Setting a value that is greater than the current operating
	// system limit will prevent the HAProxy process from
	// starting.
	//
	// If you choose a discrete value (e.g., 750000) and the
	// router pod is migrated to a new node, there's no guarantee
	// that new node has identical ulimits configured. In
	// such a scenario the pod would fail to start. If you have
	// nodes with different ulimits configured (e.g., different
	// tuned profiles) and you choose a discrete value then the
	// guidance is to use -1 and let the value be computed
	// dynamically at runtime.
	//
	// You can monitor memory usage for router containers with the
	// following metric:
	// 'container_memory_working_set_bytes{container="router",namespace="openshift-ingress"}'.
	//
	// You can monitor memory usage of individual HAProxy
	// processes in router containers with the following metric:
	// 'container_memory_working_set_bytes{container="router",namespace="openshift-ingress"}/container_processes{container="router",namespace="openshift-ingress"}'.
	//
	// +kubebuilder:validation:Optional
	// +optional
	MaxConnections int32 `json:"maxConnections,omitempty"`
}

type RouteAdmissionPolicy struct {
	// Describes how host name claims across namespaces should be handled.
	//
	// Value must be one of:
	//
	// - Strict: Do not allow routes in different namespaces to claim the same host.
	//
	// - InterNamespaceAllowed: Allow routes to claim different paths of the same
	//   host name across namespaces.
	//
	// If empty, the default is InterNamespaceAllowed.
	// +kubebuilder:default="InterNamespaceAllowed"
	NamespaceOwnership NamespaceOwnershipEnum `json:"namespaceOwnership"`
	// wildcardPolicy describes how routes with wildcard policies should
	// be handled for the ingress controller. WildcardPolicy controls use
	// of routes [1] exposed by the ingress controller based on the route's
	// wildcard policy.
	//
	// [1] https://github.com/openshift/api/blob/master/route/v1/types.go
	//
	// Note: Updating WildcardPolicy from WildcardsAllowed to WildcardsDisallowed
	// will cause admitted routes with a wildcard policy of Subdomain to stop
	// working. These routes must be updated to a wildcard policy of None to be
	// readmitted by the ingress controller.
	//
	// WildcardPolicy supports WildcardsAllowed and WildcardsDisallowed values.
	//
	// If empty, defaults to "WildcardsDisallowed".
	//
	WildcardPolicy WildcardPolicy `json:"wildcardPolicy,omitempty"`
}

type IngressPortsConfig struct {
	// Default router http port. Must be in range 1-65535.
	// +kubebuilder:default=80
	Http *int `json:"http"`
	// Default router https port. Must be in range 1-65535.
	// +kubebuilder:default=443
	Https *int `json:"https"`
}

type AccessLogging struct {
	// Status of the access logging. If set to "Enabled", the router will
	// log all requests to the access log. If set to "Disabled", the router
	// will not log any requests to the access log.
	//+kubebuilder:default=Disabled
	//+kubebuilder:validation:Enum=Disabled;Enabled
	Status AccessLoggingStatusEnum `json:"status"`

	// destination is where access logs go.
	//
	// +required
	Destination operatorv1.LoggingDestination `json:"destination"`

	// httpLogFormat specifies the format of the log message for an HTTP
	// request.
	//
	// If this field is empty, log messages use the implementation's default
	// HTTP log format.  For HAProxy's default HTTP log format, see the
	// HAProxy documentation:
	// http://cbonte.github.io/haproxy-dconv/2.0/configuration.html#8.2.3
	//
	// Note that this format only applies to cleartext HTTP connections
	// and to secure HTTP connections for which the ingress controller
	// terminates encryption (that is, edge-terminated or reencrypt
	// connections).  It does not affect the log format for TLS passthrough
	// connections.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:default:=""
	// +optional
	HttpLogFormat string `json:"httpLogFormat,omitempty"`

	// httpCaptureHeaders defines HTTP headers that should be captured in
	// access logs.  If this field is empty, no headers are captured.
	//
	// Note that this option only applies to cleartext HTTP connections
	// and to secure HTTP connections for which the ingress controller
	// terminates encryption (that is, edge-terminated or reencrypt
	// connections).  Headers cannot be captured for TLS passthrough
	// connections.
	//
	// +optional
	HttpCaptureHeaders operatorv1.IngressControllerCaptureHTTPHeaders `json:"httpCaptureHeaders,omitempty"`

	// httpCaptureCookies specifies HTTP cookies that should be captured in
	// access logs.  If this field is empty, no cookies are captured.
	//
	// +nullable
	// +optional
	// +kubebuilder:validation:MaxItems=1
	// +listType=atomic
	HttpCaptureCookies []operatorv1.IngressControllerCaptureHTTPCookie `json:"httpCaptureCookies,omitempty"`
}

func (a *AccessLogging) Validate() error {
	if a.Status != AccessLoggingEnabled && a.Status != AccessLoggingDisabled {
		return fmt.Errorf("invalid access logging status: %s", a.Status)
	}

	if a.Status == AccessLoggingDisabled {
		return nil
	}

	switch a.Destination.Type {
	case operatorv1.ContainerLoggingDestinationType:
		if a.Destination.Container != nil {
			if a.Destination.Container.MaxLength > 0 && (a.Destination.Container.MaxLength < 480 || a.Destination.Container.MaxLength > 8192) {
				return fmt.Errorf("invalid container maxLength: %d. Must be between 480 and 8192", a.Destination.Container.MaxLength)
			}
		}
	case operatorv1.SyslogLoggingDestinationType:
		if a.Destination.Syslog == nil {
			return fmt.Errorf("destination syslog is required")
		}
		if a.Destination.Syslog.Address == "" {
			return fmt.Errorf("destination syslog address is required")
		}
		if net.ParseIP(a.Destination.Syslog.Address) == nil {
			return fmt.Errorf("invalid syslog address: %s. Must be a valid IPv4 or IPv6 address", a.Destination.Syslog.Address)
		}
		if a.Destination.Syslog.Port < 1 || a.Destination.Syslog.Port > 65535 {
			return fmt.Errorf("invalid syslog port: %d. Must be between 1 and 65535", a.Destination.Syslog.Port)
		}
		if a.Destination.Syslog.Facility != "" && !slices.Contains(allowedFacilities, a.Destination.Syslog.Facility) {
			return fmt.Errorf("invalid syslog facility: %s. Must be one of %v", a.Destination.Syslog.Facility, allowedFacilities)
		}
		if a.Destination.Syslog.MaxLength > 0 && (a.Destination.Syslog.MaxLength < 480 || a.Destination.Syslog.MaxLength > 8192) {
			return fmt.Errorf("invalid syslog maxLength: %d. Must be between 480 and 8192", a.Destination.Syslog.MaxLength)
		}
	default:
		return fmt.Errorf("invalid access logging destination type: %s", a.Destination.Type)
	}

	if len(a.HttpCaptureCookies) > 1 {
		return fmt.Errorf("invalid number of capture cookies: %d. Must be 1 at most", len(a.HttpCaptureCookies))
	}
	if len(a.HttpCaptureCookies) == 1 {
		cookie := a.HttpCaptureCookies[0]
		if isDefaultCookie(&cookie) {
			a.HttpCaptureCookies = nil
		} else if err := validateCookie(&cookie); err != nil {
			return err
		}
	}
	if len(a.HttpCaptureHeaders.Request) > 0 {
		filteredList := make([]operatorv1.IngressControllerCaptureHTTPHeader, 0)
		for _, h := range a.HttpCaptureHeaders.Request {
			if isDefaultHeader(&h) {
				continue
			} else if err := validateHeader(&h); err != nil {
				return fmt.Errorf("invalid request header: %s", err)
			}
			filteredList = append(filteredList, h)
		}
		a.HttpCaptureHeaders.Request = filteredList
	}
	if len(a.HttpCaptureHeaders.Response) > 0 {
		filteredList := make([]operatorv1.IngressControllerCaptureHTTPHeader, 0)
		for _, h := range a.HttpCaptureHeaders.Response {
			if isDefaultHeader(&h) {
				continue
			} else if err := validateHeader(&h); err != nil {
				return fmt.Errorf("invalid response header: %s", err)
			}
			filteredList = append(filteredList, h)
		}
		a.HttpCaptureHeaders.Response = filteredList
	}
	return nil
}

func isDefaultHeader(h *operatorv1.IngressControllerCaptureHTTPHeader) bool {
	return h.MaxLength == 0 && h.Name == ""
}

func validateHeader(h *operatorv1.IngressControllerCaptureHTTPHeader) error {
	if h.Name == "" {
		return fmt.Errorf("header name is required")
	}
	if !headerNamePattern.MatchString(h.Name) {
		return fmt.Errorf("header name '%s' contains invalid characters", h.Name)
	}
	if h.MaxLength < 1 {
		return fmt.Errorf("header '%s' maxLength must be at least 1", h.Name)
	}
	return nil
}

func isDefaultCookie(c *operatorv1.IngressControllerCaptureHTTPCookie) bool {
	return c.MatchType == "" && c.MaxLength == 0 && c.Name == "" && c.NamePrefix == ""
}

func validateCookie(c *operatorv1.IngressControllerCaptureHTTPCookie) error {
	if c.MatchType != "Exact" && c.MatchType != "Prefix" {
		return fmt.Errorf("invalid cookie match type: %s. Must be `Exact` or `Prefix`", c.MatchType)
	}
	if c.MaxLength < 1 || c.MaxLength > 1024 {
		return fmt.Errorf("invalid cookie maxLength: %d. Must be between 1 and 1024", c.MaxLength)
	}
	if c.MatchType == "Exact" {
		if len(c.Name) > 1024 {
			return fmt.Errorf("invalid cookie name length: %d. Must be less than 1024", len(c.Name))
		}
		if !headerNamePattern.MatchString(c.Name) {
			return fmt.Errorf("cookie name '%s' contains invalid characters", c.Name)
		}
		return nil
	}
	if len(c.NamePrefix) > 1024 {
		return fmt.Errorf("invalid cookie namePrefix length: %d. Must be less than 1024", len(c.NamePrefix))
	}
	if !headerNamePattern.MatchString(c.NamePrefix) {
		return fmt.Errorf("cookie namePrefix '%s' contains invalid characters", c.NamePrefix)
	}
	return nil
}
