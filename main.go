package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type GcoreResponse struct {
	Addresses   []string `json:"addresses"`
	AddressesV6 []string `json:"addresses_v6"`
}

func fetchGcoreIPs(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var gcoreResp GcoreResponse
	err = json.Unmarshal(body, &gcoreResp)
	if err != nil {
		return nil, err
	}

	return append(gcoreResp.Addresses, gcoreResp.AddressesV6...), nil
}

func fetchIPs(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ips := strings.Split(string(body), "\n")
	return ips, nil
}

func writeConfig(ips []string, filepath string) (int, error) {
	var builder strings.Builder
	builder.WriteString("# NGINX IP Whitelist\n")
	for _, ip := range ips {
		if ip != "" {
			builder.WriteString(fmt.Sprintf("allow %s;\n", ip))
		}
	}
	builder.WriteString("deny all;\n")

	content := builder.String()
	err := ioutil.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		return 0, err
	}

	// Count the number of lines in the file
	lines := strings.Count(content, "\n")
	return lines, nil
}

func reloadNginx() error {
	cmd := exec.Command("nginx", "-s", "reload")
	return cmd.Run()
}

func executeTask(filepath string) {
	gcoreIPs, err := fetchGcoreIPs("https://api.gcore.com/cdn/public-ip-list")
	if err != nil {
		fmt.Println("Error fetching Gcore IPs:", err)
		return
	}

	cloudflareIPv4, err := fetchIPs("https://www.cloudflare.com/ips-v4")
	if err != nil {
		fmt.Println("Error fetching Cloudflare IPv4 IPs:", err)
		return
	}

	cloudflareIPv6, err := fetchIPs("https://www.cloudflare.com/ips-v6")
	if err != nil {
		fmt.Println("Error fetching Cloudflare IPv6 IPs:", err)
		return
	}

	allIPs := append(gcoreIPs, cloudflareIPv4...)
	allIPs = append(allIPs, cloudflareIPv6...)

	lines, err := writeConfig(allIPs, filepath)
	if err != nil {
		fmt.Println("Error writing config:", err)
		return
	}

	fmt.Printf("NGINX whitelist configuration has been written to %s with %d lines\n", filepath, lines)

	err = reloadNginx()
	if err != nil {
		fmt.Println("Error reloading NGINX:", err)
	} else {
		fmt.Println("NGINX configuration reloaded successfully")
	}
}

func main() {
	// Define command-line flags
	location := flag.String("location", "/etc/nginx/conf.d", "Directory to save the allow.conf file")
	filename := flag.String("filename", "allow.conf", "Name of the allow.conf file")
	hour := flag.Int("hour", 3, "Hour of the day to run the task (0-23)")
	help := flag.Bool("help", false, "Show help message")

	// Parse command-line flags
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	filepath := fmt.Sprintf("%s/%s", *location, *filename)

	// Execute task immediately on startup
	executeTask(filepath)

	// Schedule task to run at the specified hour every day
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), *hour, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))

		executeTask(filepath)
	}
}
