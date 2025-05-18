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

// findInterfaceByIP 根据给定的 IP 查询所属的网络接口
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

// dialerWithInterface 返回一个自定义的 net.Dialer，该 dialer 在建立连接时将 socket 绑定到指定的网络接口（仅限 Linux）
func dialerWithInterface(iface string) *net.Dialer {
	return &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, _ string, c syscall.RawConn) error {
			var controlErr error
			// 在 socket 建立后，通过 Control 回调设置 SO_BINDTODEVICE 选项绑定到指定接口
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

// requestDataWith 使用自定义请求头获取数据
func requestDataWith(ip net.IP, url, method, ua string) (data []byte, err error) {
	iface, err := findInterfaceByIP(ip)
	if err != nil {
		return nil, err
	}

	dialer := dialerWithInterface(iface.Name)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	// 提交请求
	var request *http.Request
	request, err = http.NewRequest(method, url, nil)
	if err == nil {
		// 增加header选项
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
