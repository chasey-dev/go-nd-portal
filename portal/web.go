package portal

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// findInterfaceByIP find interface by given IP address
func findInterfaceByIP(ip net.IP) (*net.Interface, error) {
	logrus.Debugln("request ip: ", ip)
	ifaces, err := net.Interfaces()
	if err != nil {
		s := fmt.Sprintf("get interface list failed: %v", err)
		return nil, errors.New(s)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ipNet *net.IPNet
			var addrIP net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
				addrIP = v.IP
			case *net.IPAddr:
				ipNet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
				addrIP = v.IP
			}
			if ipNet != nil && addrIP.Equal(ip) {
				logrus.Debugln("ip bound to iface", iface)
				return &iface, nil
			}
		}
	}
	s := fmt.Sprintf("IP %s doesnt belong to any interface", ip.String())
	return nil, errors.New(s)
}

// dialerWithInterface returns a net.Dialer，
// which binds socket to specified interface when establishing connection
// linux only
func dialerWithInterface(iface string) *net.Dialer {
	return &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, _ string, c syscall.RawConn) error {
			var controlErr error
			// set SO_BINDTODEVICE for Control to bind to specified interface,
			// after socket is initialized
			err := c.Control(func(fd uintptr) {
				controlErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, iface)
			})
			if err != nil {
				return err
			}
			return controlErr
		},
	}
}

// requestDataWith request data with customized http header
func requestDataWith(ip net.IP, url, method, ua string) (data []byte, err error) {
	// get the actual iface first
	iface, err := findInterfaceByIP(ip)
	if err != nil {
		return nil, err
	}
	// get the dialer with bound iface
	dialer := dialerWithInterface(iface.Name)

	// specify DialContext with given dialer
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	// send req
	var request *http.Request
	request, err = http.NewRequest(method, url, nil)
	if err == nil {
		// add User-Agent to http header
		if ua != "" {
			request.Header.Add("User-Agent", ua)
		}
		var response *http.Response
		response, err = client.Do(request)
		if err == nil {
			if response.StatusCode != http.StatusOK {
				s := fmt.Sprintf("status code: %d", response.StatusCode)
				err = errors.New(s)
				return
			}
			data, err = io.ReadAll(response.Body)
			response.Body.Close()
		}
	}
	return
}
