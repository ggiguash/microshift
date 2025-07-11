package ovn

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"github.com/openshift/microshift/pkg/util"
	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

const (
	ovnConfigFileName           = "ovn.yaml"
	OVNGatewayInterface         = "br-ex"
	defaultMTU                  = 1500
	OVNKubernetesV4MasqueradeIP = "169.254.169.2"
	OVNKubernetesV6MasqueradeIP = "fd69::2"

	// used for multinode ovn database transport
	OVN_NB_PORT = "9641"
	OVN_SB_PORT = "9642"

	// geneve header length for IPv4
	GeneveHeaderLengthIPv4 = 58
	// geneve header length for IPv6
	GeneveHeaderLengthIPv6 = GeneveHeaderLengthIPv4 + 20
)

type OVNKubernetesConfig struct {
	// MTU to use for the pod interface. Default is 1500.
	MTU int `json:"mtu,omitempty"`
}

func NewOVNKubernetesConfigFromFileOrDefault(dir string, multinode bool, ipFamily int) (*OVNKubernetesConfig, error) {
	path := filepath.Join(dir, ovnConfigFileName)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			klog.Infof("OVNKubernetes config file not found, assuming default values")
			return new(OVNKubernetesConfig).withDefaults(multinode, ipFamily), nil
		}
		return nil, fmt.Errorf("failed to get OVNKubernetes config file: %v", err)
	}

	o, err := newOVNKubernetesConfigFromFile(path, multinode, ipFamily)
	if err == nil {
		return o, nil
	}
	return nil, fmt.Errorf("getting OVNKubernetes config: %v", err)
}

func (o *OVNKubernetesConfig) Validate() error {
	// br-ex is required to run ovn-kubernetes
	err := o.validateOVSBridge()
	if err != nil {
		return fmt.Errorf("failed to validate OVS bridge: %w", err)
	}
	return nil
}

// validateOVSBridge validates the existence of ovn-kubernetes br-ex bridge
func (o *OVNKubernetesConfig) validateOVSBridge() error {
	_, err := net.InterfaceByName(OVNGatewayInterface)
	if err != nil {
		return fmt.Errorf("failed to find OVN gateway interface %q: %w", OVNGatewayInterface, err)
	}
	return nil
}

// getClusterMTU retrieves MTU from the default route network interface,
// and falls back to use 1500 when unable to get the mtu or ipFamily than 0.
func (o *OVNKubernetesConfig) getClusterMTU(multinode bool, ipFamily int) {
	klog.Infof("getClusterMTU: finding default route interface")
	o.MTU = defaultMTU

	// if configure both IPV4 and IPV6 check the smallest
	//nolint:nestif
	if ipFamily == netlink.FAMILY_ALL {
		mtu, err := util.FindDefaultRouteMinMTU()

		if err == nil {
			o.MTU = mtu
		} else {
			klog.Infof("getClusterMTU: error %s.", err)
		}
	} else {
		mtu, err := util.FindDefaultRouteMTU(ipFamily)
		if err == nil {
			o.MTU = mtu
		} else {
			klog.Infof("getClusterMTU: error %s.", err)
		}
	}

	if multinode {
		o.MTU = o.MTU - GeneveHeaderLengthIPv6
	}
}

// withDefaults returns the default values when ovn.yaml is not provided
func (o *OVNKubernetesConfig) withDefaults(multinode bool, ipFamily int) *OVNKubernetesConfig {
	o.getClusterMTU(multinode, ipFamily)
	return o
}

func newOVNKubernetesConfigFromFile(path string, multinode bool, ipFamily int) (*OVNKubernetesConfig, error) {
	o := new(OVNKubernetesConfig)
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(buf, o)
	if err != nil {
		return nil, fmt.Errorf("parsing OVNKubernetes config: %v", err)
	}
	// in case mtu is not defined
	if o.MTU == 0 {
		o.getClusterMTU(multinode, ipFamily)
	}
	klog.Infof("parsed OVNKubernetes config from file %q: %+v", path, o)

	return o, nil
}

func ExcludeOVNKubernetesMasqueradeIPs(addrs []net.Addr) []net.Addr {
	var netAddrs []net.Addr
	for _, a := range addrs {
		ipNet, _, _ := net.ParseCIDR(a.String())
		if ipNet.String() != OVNKubernetesV4MasqueradeIP && ipNet.String() != OVNKubernetesV6MasqueradeIP {
			netAddrs = append(netAddrs, a)
		}
	}
	return netAddrs
}

func IsOVNKubernetesInternalInterface(name string) bool {
	excludedInterfacesRegexp := regexp.MustCompile(
		"^[A-Fa-f0-9]{15}|" + // OVN pod interfaces
			"ovn.*|" + // OVN ovn-k8s-mp0 and similar interfaces
			"br-int|" + // OVN integration bridge
			"veth.*|cni.*|" + // Interfaces used in bridge-cni or flannel
			"ovs-system$") // Internal OVS interface

	return excludedInterfacesRegexp.MatchString(name)
}
