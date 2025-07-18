/*
Copyright © 2021 MicroShift Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	tcpnet "net"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

var previousGatewayIP string = ""

type routeStruct struct {
	// Name of interface
	Iface string

	// big-endian hex string
	Gateway string
}

// Remember whether we have successfully found the hard-coded nodeIP
// on this host.
var foundHardCodedNodeIP bool

func GetHostIP(nodeIP string) (string, error) {
	var hostIP string
	var err error

	if nodeIP != "" {
		if !foundHardCodedNodeIP {
			foundHardCodedNodeIP = true
			klog.Infof("trying to find configured nodeIP %q on host", nodeIP)
		}
		hostIP, err = selectIPFromHostInterface(nodeIP)
		if err != nil {
			foundHardCodedNodeIP = false
			return "", fmt.Errorf("failed to find the configured nodeIP %q on host: %v", nodeIP, err)
		}
		goto found
	}

	if ip, err := net.ChooseHostInterface(); err == nil {
		hostIP = ip.String()
	} else {
		klog.Infof("failed to get host IP by default route: %v", err)
		if hostIP, err = selectIPFromHostInterface(""); err != nil {
			return "", err
		}
	}

found:
	if hostIP != previousGatewayIP {
		previousGatewayIP = hostIP
		klog.V(2).Infof("host gateway IP address: %s", hostIP)
	}

	return hostIP, nil
}

func RetryInsecureGet(ctx context.Context, url string) int {
	return RetryGet(ctx, url, "")
}

func RetryGet(ctx context.Context, url, additionalCAPath string) int {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		klog.Infof("Warning: Failed to load system CA certificates: %v. Creating an empty pool.", err)
		rootCAs = x509.NewCertPool()
	}
	if additionalCAPath != "" {
		caCert, err := os.ReadFile(additionalCAPath)
		if err != nil {
			klog.Errorf("failed to read CA certificate %s: %v", additionalCAPath, err)
			return 0
		}

		if !rootCAs.AppendCertsFromPEM(caCert) {
			klog.Errorf("failed to append CA certificate %s to pool", additionalCAPath)
			return 0
		}
	}
	status := 0
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 120*time.Second, false, func(ctx context.Context) (bool, error) {
		c := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    rootCAs,
					MinVersion: tls.VersionTLS12,
				},
			},
		}
		resp, err := c.Get(url)
		if err != nil {
			return false, nil
		}
		defer func() { _ = resp.Body.Close() }()
		status = resp.StatusCode
		return true, nil
	})

	if err != nil && err == context.DeadlineExceeded {
		klog.Warningf("Endpoint is not returning any status code")
	}

	return status
}

func RetryTCPConnection(ctx context.Context, host string, port string) bool {
	status := false
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 120*time.Second, false, func(ctx context.Context) (bool, error) {
		timeout := 30 * time.Second
		_, err := tcpnet.DialTimeout("tcp", tcpnet.JoinHostPort(host, port), timeout)

		if err == nil {
			status = true
			return true, nil
		}
		return false, nil
	})
	if err != nil && err == context.DeadlineExceeded {
		klog.Warningf("Endpoint is not returning any status code")
	}
	return status
}

func AddToNoProxyEnv(additionalEntries ...string) error {
	entries := map[string]struct{}{}

	// put both the NO_PROXY and no_proxy elements in a map to avoid duplicates
	addNoProxyEnvVarEntries(entries, "NO_PROXY")
	addNoProxyEnvVarEntries(entries, "no_proxy")

	for _, entry := range additionalEntries {
		entries[entry] = struct{}{}
	}

	noProxyEnv := strings.Join(mapKeys(entries), ",")

	// unset the lower-case one, and keep only upper-case
	_ = os.Unsetenv("no_proxy")
	if err := os.Setenv("NO_PROXY", noProxyEnv); err != nil {
		return fmt.Errorf("failed to update NO_PROXY: %w", err)
	}
	return nil
}

func mapKeys(entries map[string]struct{}) []string {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}

	// sort keys to avoid issues with map key ordering in go future versions on the unit-test side
	sort.Strings(keys)
	return keys
}

func addNoProxyEnvVarEntries(entries map[string]struct{}, envVar string) {
	noProxy := os.Getenv(envVar)

	if noProxy != "" {
		for _, entry := range strings.Split(noProxy, ",") {
			entries[strings.Trim(entry, " ")] = struct{}{}
		}
	}
}

func selectIPFromHostInterface(nodeIP string) (string, error) {
	ifaces, err := tcpnet.Interfaces()
	if err != nil {
		return "", err
	}
	// Sort all interfaces by index. Interfaces are listed in the same ordering as
	// in /proc/net/dev. The ordering is not based in interface index, and it is
	// not guaranteed to list system interfaces first. Veth interfaces from containers
	// could get listed first and retrieve the wrong node IP.
	// The index is assigned sequentially depending on the order of kmods loading
	// and interface creation. Since the interfaces we are looking for belong to
	// NetworkManager and the ones that could get a wrong node IP depend on a systemd
	// unit that requires system networking to be ready, we can assume the lower
	// indices are safe to use.
	slices.SortFunc(ifaces, func(a, b tcpnet.Interface) int {
		if a.Index > b.Index {
			return 1
		} else if a.Index < b.Index {
			return -1
		}
		return 0
	})
	// get list of interfaces
	for _, i := range ifaces {
		if i.Name == "br-ex" {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			klog.Warningf("failed to get IPs for interface %s: %v", i.Name, err)
			continue
		}

		for _, addr := range addrs {
			ip, _, err := tcpnet.ParseCIDR(addr.String())
			if err != nil {
				return "", fmt.Errorf("unable to parse CIDR for interface %q: %s", i.Name, err)
			}
			if ip.IsLoopback() {
				continue
			}
			if nodeIP != "" && nodeIP != ip.String() {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no interface with valid address found on host")
}

// ContainIPANetwork - will check if given IP address contained within list of networks
func ContainIPANetwork(ip tcpnet.IP, networks []string) bool {
	for _, netStr := range networks {
		_, netA, err := tcpnet.ParseCIDR(netStr)
		if err != nil {
			klog.Warningf("Could not parse CIDR %s, err: %v", netA, err)
			return false
		}
		if netA.Contains(ip) {
			return true
		}
	}
	return false
}

func GetHostIPv6(ipHint string) (string, error) {
	handle, err := netlink.NewHandle()
	if err != nil {
		return "", err
	}
	// Start by looking for the default route and using the dev
	// address.
	routeList, err := handle.RouteList(nil, netlink.FAMILY_V6)
	if err != nil {
		return "", err
	}
	defaultRouteLinkIndex := -1
	for _, route := range routeList {
		if route.Dst == nil {
			defaultRouteLinkIndex = route.LinkIndex
			break
		}
	}

	if defaultRouteLinkIndex != -1 {
		link, err := handle.LinkByIndex(defaultRouteLinkIndex)
		if err != nil {
			return "", err
		}
		addrList, err := handle.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return "", err
		}
		for _, addr := range addrList {
			if ipHint != "" && ipHint != addr.IP.String() {
				continue
			}
			return addr.IP.String(), nil
		}
	}

	// If there is no default route then pick the first ipv6
	// address that fits.
	addrList, err := handle.AddrList(nil, netlink.FAMILY_V6)
	if err != nil {
		return "", err
	}
	for _, addr := range addrList {
		ip, _, err := tcpnet.ParseCIDR(addr.String())
		if err != nil {
			return "", fmt.Errorf("unable to parse CIDR from address %q: %s", addr.String(), err)
		}
		if ip.IsLoopback() || ip.IsLinkLocalMulticast() {
			continue
		}
		if ipHint != "" && ipHint != ip.String() {
			continue
		}
		return ip.String(), nil
	}

	return "", fmt.Errorf("unable to find host IPv6 address")
}

func FindDefaultRouteMinMTU() (mtu int, err error) {
	ipFamilies := []int{netlink.FAMILY_V4, netlink.FAMILY_V6}

	mtu_slice := []int{}

	for _, ipFamily := range ipFamilies {
		new_mtu, err := FindDefaultRouteMTU(ipFamily)
		if err != nil {
			continue
		}
		mtu_slice = append(mtu_slice, new_mtu)
	}
	if len(mtu_slice) > 0 {
		return slices.Min(mtu_slice), nil
	}
	return 0, fmt.Errorf("could not find minimal MTU")
}

func FindDefaultRouteMTU(ipFamily int) (mtu int, err error) {
	link, err := FindDefaultRouteIface(ipFamily)
	if err != nil || link.MTU == 0 {
		return 0, err
	}

	klog.Infof("using IP %d on Interface %s with MTU %d ", ipFamily, link.Name, link.MTU)

	return link.MTU, nil
}

// Find the Default route Interface based on ipv4 or ipv6 routes.
func FindDefaultRouteIface(ipFamily int) (iface *tcpnet.Interface, err error) {
	parsedStruct, err := findDefaultRouteForFamily(ipFamily)
	if err != nil {
		return nil, err
	}

	iface, err = tcpnet.InterfaceByName(parsedStruct.Iface)
	if err != nil {
		return nil, err
	}

	return iface, nil
}

func findDefaultRouteForFamily(family int) (routeStruct, error) {
	handle, err := netlink.NewHandle()
	if err != nil {
		return routeStruct{}, err
	}

	routeList, err := handle.RouteList(nil, family)
	if err != nil {
		return routeStruct{}, err
	}

	for _, route := range routeList {
		//  for Default route the Destination should be nil (0)
		if route.Dst == nil {
			link, err := handle.LinkByIndex(route.LinkIndex)
			if err != nil {
				return routeStruct{}, err
			}
			return routeStruct{
				Iface:   link.Attrs().Name,
				Gateway: route.Gw.String(),
			}, nil
		}
	}
	return routeStruct{}, fmt.Errorf("no default gateway found")
}

// HasDefaultRoute returns whether the host has a default route for IPv4 or IPv6.
func HasDefaultRoute() (bool, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return false, fmt.Errorf("failed to get route list: %v", err)
	}
	for _, route := range routes {
		if route.Dst == nil || route.Dst.IP.Equal(tcpnet.IPv4zero) || route.Dst.IP.Equal(tcpnet.IPv6zero) {
			return true, nil
		}
	}
	return false, nil
}
