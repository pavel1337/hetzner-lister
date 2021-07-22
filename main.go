package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

func main() {
	configPath := flag.String("c", "", " path to config file")
	savePath := flag.String("s", "", " path to save ips")
	flag.Parse()

	c, err := parseConfig(*configPath)
	if err != nil {
		panic(err)
	}

	var ips []string

	for _, ctoken := range c.CloudTokens {
		cloudIPs, err := CloudIps(ctoken)
		if err != nil {
			panic(err)
		}
		ips = append(ips, cloudIPs...)
	}

	for _, rcreds := range c.RobotCreds {
		robotIPs, err := RobotIps(rcreds.User, rcreds.Password)
		if err != nil {
			panic(err)
		}
		ips = append(ips, robotIPs...)
	}
	if *savePath != "" {
		err := SaveIps(ips, *savePath)
		if err != nil {
			panic(err)
		}
	}
	for _, ip := range ips {
		fmt.Println(ip)
	}
}

func SaveIps(ips []string, savePath string) error {
	f, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, ip := range ips {
		f.WriteString(ip + "\n")
	}
	return nil
}

func CloudIps(token string) ([]string, error) {
	var ips []string
	client := hcloud.NewClient(hcloud.WithToken(token))

	// take public ips from all servers
	servers, err := client.Server.All(context.Background())
	if err != nil {
		return ips, err
	}
	for _, s := range servers {
		ips = append(ips, s.PublicNet.IPv4.IP.String())
	}

	// take floating ips from all servers
	fips, err := client.FloatingIP.All(context.Background())
	if err != nil {
		return ips, err
	}
	for _, fip := range fips {
		ips = append(ips, fip.IP.String())
	}

	// take ips from all loadbalancers
	lbips, err := client.LoadBalancer.All(context.Background())
	if err != nil {
		return ips, err
	}
	for _, lbip := range lbips {
		ips = append(ips, lbip.PublicNet.IPv4.IP.String())
	}

	return ips, nil
}

type Server struct {
	Server struct {
		ServerIP     string   `json:"server_ip"`
		ServerNumber int      `json:"server_number"`
		ServerName   string   `json:"server_name"`
		Product      string   `json:"product"`
		Dc           string   `json:"dc"`
		Traffic      string   `json:"traffic"`
		Flatrate     bool     `json:"flatrate"`
		Status       string   `json:"status"`
		Throttled    bool     `json:"throttled"`
		Cancelled    bool     `json:"cancelled"`
		PaidUntil    string   `json:"paid_until"`
		IP           []string `json:"ip"`
		Subnet       []struct {
			IP   string `json:"ip"`
			Mask string `json:"mask"`
		} `json:"subnet"`
	} `json:"server"`
}

func RobotIps(user, password string) ([]string, error) {
	var ips []string

	req, err := http.NewRequest("GET", "https://robot-ws.your-server.de/server", nil)
	if err != nil {
		return ips, err
	}
	req.SetBasicAuth(user, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ips, err
	}
	defer resp.Body.Close()

	var servers []Server
	err = json.NewDecoder(resp.Body).Decode(&servers)
	if err != nil {
		return ips, err
	}
	for _, s := range servers {
		ips = append(ips, s.Server.IP...)

		// add all server's subnets' addresses
		for _, subnet := range s.Server.Subnet {
			if !validIPv4Address(subnet.IP) {
				continue
			}
			ipsfs, err := ipsFromSubnet(
				fmt.Sprintf("%s/%s", subnet.IP, subnet.Mask),
			)
			if err != nil {
				fmt.Println(err)
			}
			ips = append(ips, ipsfs...)
		}
	}
	return ips, nil
}

func ipsFromSubnet(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// remove network address and broadcast address
	lenIPs := len(ips)
	switch {
	case lenIPs < 2:
		return ips, nil

	default:
		return ips[1 : len(ips)-1], nil
	}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func validIPv4Address(ip string) bool {
	return net.ParseIP(ip).To4() != nil
}
