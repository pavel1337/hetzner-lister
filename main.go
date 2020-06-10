package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

	servers, err := client.Server.All(context.Background())
	if err != nil {
		return ips, err
	}
	for _, s := range servers {
		ips = append(ips, s.PublicNet.IPv4.IP.String())
	}

	fips, err := client.FloatingIP.All(context.Background())
	if err != nil {
		return ips, err
	}
	for _, fip := range fips {
		ips = append(ips, fip.IP.String())
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
	}
	return ips, nil
}
