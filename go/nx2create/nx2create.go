package main

import (
        "bufio"
        "bytes"
        "flag"
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "strconv"
        "strings"
        "text/template"
)

const proxyTemplate = `upstream {{.SiteHostName}} {
    server {{.IPHostName}}:{{.PortUpstream}};
}

server {
    listen 80;
    server_name {{.SiteName}};
    access_log /var/log/nginx/{{.SiteHostName}}_access.log;
    error_log /var/log/nginx/{{.SiteHostName}}_error.log;

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name {{.SiteName}};
    access_log /var/log/nginx/{{.SiteHostName}}_access.log;
    error_log /var/log/nginx/{{.SiteHostName}}_error.log;

    ssl_certificate "{{.FullchainPath}}";
    ssl_certificate_key "{{.PrivkeyPath}}";
    ssl_session_timeout 10m;
    ssl_ciphers 'ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA256';
    ssl_prefer_server_ciphers on;
    ssl_dhparam /etc/nginx/dhparam.pem;

    location / {
        proxy_pass {{.Protocol}}://{{.SiteHostName}};
        proxy_redirect off;
        proxy_buffering off;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`

const localTemplate = `server {
    listen 80;
    server_name {{.SiteName}};
    access_log /var/log/nginx/{{.SiteHostName}}_access.log;
    error_log /var/log/nginx/{{.SiteHostName}}_error.log;

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name {{.SiteName}};
    access_log /var/log/nginx/{{.SiteHostName}}_access.log;
    error_log /var/log/nginx/{{.SiteHostName}}_error.log;

    ssl_certificate "{{.FullchainPath}}";
    ssl_certificate_key "{{.PrivkeyPath}}";
    ssl_session_timeout 10m;
    ssl_ciphers 'ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA256';
    ssl_prefer_server_ciphers on;
    ssl_dhparam /etc/nginx/dhparam.pem;

    root /var/www/{{.SiteHostName}};
    index index.html index.htm;

    location / {
        try_files $uri $uri/ /index.html;
    }
}`

// ConfigData holds template variables.
type ConfigData struct {
        SiteName      string
        SiteHostName  string
        IPHostName    string
        PortUpstream  string
        FullchainPath string
        PrivkeyPath   string
        Protocol      string
}

// isIPAddress checks if the input is an IP address (IPv4 or IPv6).
func isIPAddress(input string) bool {
        // IPv4 pattern
        ipv4Pattern := `^(\d{1,3}\.){3}\d{1,3}$`
        // Simplified IPv6 pattern (not exhaustive, but sufficient for basic checks)
        ipv6Pattern := `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,7}:$|^::1$`
        return regexp.MustCompile(ipv4Pattern).MatchString(input) || regexp.MustCompile(ipv6Pattern).MatchString(input)
}

// validateParams checks if all parameters are valid for Nginx.
func validateParams(data ConfigData, siteType string) []string {
        var errors []string

        // Validate site name (basic domain format: at least one dot, no spaces)
        if data.SiteName == "" {
                errors = append(errors, "Site name is empty")
        } else if !regexp.MustCompile(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(data.SiteName) {
                errors = append(errors, "Invalid site name format (must be a valid domain, e.g., www.example.com)")
        }

        // Validate site hostname (must not be empty)
        if data.SiteHostName == "" {
                errors = append(errors, "Site hostname could not be extracted from site name")
        }

        // Validate site type
        if siteType != "proxy" && siteType != "local" {
                errors = append(errors, "Site type must be 'proxy' or 'local'")
        }

        // Validate proxy-specific parameters
        if siteType == "proxy" {
                if data.IPHostName == "" {
                        errors = append(errors, "Upstream hostname or IP is empty")
                } else if strings.Contains(data.IPHostName, " ") {
                        errors = append(errors, "Upstream hostname or IP contains invalid characters (spaces)")
                }

                if data.PortUpstream == "" {
                        errors = append(errors, "Upstream port is empty")
                } else if port, err := strconv.Atoi(data.PortUpstream); err != nil || port < 1 || port > 65535 {
                        errors = append(errors, "Upstream port must be a number between 1 and 65535")
                }

                if data.Protocol != "http" && data.Protocol != "https" {
                        errors = append(errors, "Protocol must be 'http' or 'https'")
                }
        }

        // Check certificate files
        if data.FullchainPath == "" {
                errors = append(errors, "Certificate path is empty")
        } else if _, err := os.Stat(data.FullchainPath); os.IsNotExist(err) {
                errors = append(errors, fmt.Sprintf("Certificate file %s does not exist", data.FullchainPath))
        } else if info, err := os.Stat(data.FullchainPath); err != nil || info.IsDir() {
                errors = append(errors, fmt.Sprintf("Certificate path %s is not a valid file", data.FullchainPath))
        }

        if data.PrivkeyPath == "" {
                errors = append(errors, "Private key path is empty")
        } else if _, err := os.Stat(data.PrivkeyPath); os.IsNotExist(err) {
                errors = append(errors, fmt.Sprintf("Private key file %s does not exist", data.PrivkeyPath))
        } else if info, err := os.Stat(data.PrivkeyPath); err != nil || info.IsDir() {
                errors = append(errors, fmt.Sprintf("Private key path %s is not a valid file", data.PrivkeyPath))
        }

        return errors
}

func main() {
        // Define flags
        help := flag.Bool("help", false, "Display usage information")
        configDir := flag.String("config-dir", "/etc/nginx", "Path to Nginx configuration directory")
        siteNameFlag := flag.String("site-name", "", "Full site name (e.g., www.example.com)")
        siteTypeFlag := flag.String("site-type", "", "Site type (proxy or local)")
        upstreamHostFlag := flag.String("upstream-host", "", "Upstream hostname or IP for proxy")
        upstreamPortFlag := flag.String("upstream-port", "", "Upstream port for proxy")
        proxyProtocolFlag := flag.String("proxy-protocol", "", "Proxy protocol for proxy (http or https)")
        fullchainPathFlag := flag.String("fullchain-path", "", "Path to fullchain certificate")
        privkeyPathFlag := flag.String("privkey-path", "", "Path to private key")
        flag.Parse()

        // Display help if --help is passed
        if *help {
                fmt.Println("Usage: nx2createsite [--config-dir=<path>] [--site-name=<name>] [--site-type=<type>] [--upstream-host=<host>] [--upstream-port=<port>] [--proxy-protocol=<protocol>] [--fullchain-path=<path>] [--privkey-path=<path>]")
                fmt.Println("Create an Nginx site configuration in sites-available using the hostname (e.g., teste.conf for teste.tjap.jus.br).")
                fmt.Println("\nOptions:")
                fmt.Println("  --config-dir=<path>      Specify the Nginx configuration directory (default: /etc/nginx)")
                fmt.Println("  --site-name=<name>      Full site name (e.g., www.example.com)")
                fmt.Println("  --site-type=<type>      Site type (proxy or local)")
                fmt.Println("  --upstream-host=<host>  Upstream hostname or IP for proxy sites")
                fmt.Println("  --upstream-port=<port>  Upstream port for proxy sites")
                fmt.Println("  --proxy-protocol=<protocol>  Proxy protocol for proxy sites (http or https)")
                fmt.Println("  --fullchain-path=<path> Path to fullchain certificate file (default: /opt/certs/fullchain.pem)")
                fmt.Println("  --privkey-path=<path>   Path to private key file (default: /opt/certs/privkey.pem)")
                fmt.Println("  --help                  Display this help message")
                fmt.Println("\nInteractive Mode Example:")
                fmt.Println("  nx2createsite --config-dir=/custom/nginx")
                fmt.Println("  # Prompts for site name, type, upstream details, certificate paths, and reload")
                fmt.Println("\nNon-Interactive Mode Examples:")
                fmt.Println("  # Proxy site:")
                fmt.Println("  nx2createsite --site-name=teste.tjap.jus.br --site-type=proxy --upstream-host=192.168.1.100 --upstream-port=8080 --proxy-protocol=https --fullchain-path=/opt/certs/teste.pem --privkey-path=/opt/certs/teste.key")
                fmt.Println("  # Local site with defaults for certs:")
                fmt.Println("  nx2createsite --site-name=local.tjap.jus.br --site-type=local")
                fmt.Println("\nNotes:")
                fmt.Println("  - Config file uses hostname (e.g., teste.conf for teste.tjap.jus.br).")
                fmt.Println("  - For proxy sites, ensure upstream hostname is resolvable via /etc/hosts or DNS.")
                fmt.Println("  - Reload only occurs if all parameters are valid and certificates exist, followed by nx2ensite.")
                fmt.Println("  - If the config file already exists, interactive mode prompts to overwrite; non-interactive mode fails.")
                fmt.Println("  - In interactive mode, press Enter to use default certificate paths.")
                os.Exit(0)
        }

        // Check if configuration directory exists
        if _, err := os.Stat(*configDir); os.IsNotExist(err) {
                fmt.Printf("Error: Configuration directory %s does not exist.\n", *configDir)
                os.Exit(1)
        }

        // Initialize config data
        data := ConfigData{
                Protocol: "http", // Default for proxy
        }

        // Use flags if provided, otherwise prompt
        scanner := bufio.NewScanner(os.Stdin)
        var siteType string
        if *siteNameFlag != "" {
                data.SiteName = *siteNameFlag
        } else {
                fmt.Print("Enter the full site name (e.g., www.example.com): ")
                scanner.Scan()
                data.SiteName = strings.TrimSpace(scanner.Text())
        }

        // Extract hostname (e.g., teste from teste.tjap.jus.br)
        if data.SiteName != "" {
                parts := strings.Split(data.SiteName, ".")
                if len(parts) >= 2 {
                        data.SiteHostName = parts[0] // Use first part as hostname
                }
        }

        // Check if config file already exists
        configPath := filepath.Join(*configDir, "sites-available", data.SiteHostName+".conf")
        if _, err := os.Stat(configPath); err == nil {
                // File exists, check if non-interactive mode
                nonInteractive := *siteNameFlag != "" && *siteTypeFlag != "" && (*fullchainPathFlag != "" || *siteTypeFlag != "proxy") && (*privkeyPathFlag != "" || *siteTypeFlag != "proxy")
                if nonInteractive {
                        fmt.Printf("Error: Configuration file %s already exists. Remove it or choose a different site name.\n", configPath)
                        os.Exit(2)
                }

                // Interactive mode: prompt to overwrite
                fmt.Printf("Configuration file %s already exists. Do you want to reconfigure it? (y/n): ", configPath)
                scanner.Scan()
                if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
                        fmt.Println("Operation canceled. No changes made.")
                        os.Exit(0)
                }
        }

        if *siteTypeFlag != "" {
                siteType = strings.ToLower(*siteTypeFlag)
        } else {
                fmt.Print("Is this a proxy or local site? (proxy/local): ")
                scanner.Scan()
                siteType = strings.ToLower(strings.TrimSpace(scanner.Text()))
        }

        if siteType == "proxy" {
                if *upstreamHostFlag != "" {
                        data.IPHostName = *upstreamHostFlag
                } else {
                        fmt.Print("Enter the upstream hostname or IP: ")
                        scanner.Scan()
                        data.IPHostName = strings.TrimSpace(scanner.Text())
                }

                // Check if IPHostName is a hostname (not an IP)
                if data.IPHostName != "" && !isIPAddress(data.IPHostName) {
                        fmt.Printf("Warning: Upstream hostname %s must be resolvable. Update /etc/hosts or configure DNS for the site to function properly.\n", data.IPHostName)
                }

                if *upstreamPortFlag != "" {
                        data.PortUpstream = *upstreamPortFlag
                } else {
                        fmt.Print("Enter the upstream port: ")
                        scanner.Scan()
                        data.PortUpstream = strings.TrimSpace(scanner.Text())
                }

                if *proxyProtocolFlag != "" {
                        data.Protocol = strings.ToLower(*proxyProtocolFlag)
                } else {
                        fmt.Print("Use http or https for proxy_pass? (http/https): ")
                        scanner.Scan()
                        data.Protocol = strings.ToLower(strings.TrimSpace(scanner.Text()))
                }
        }

        if *fullchainPathFlag != "" {
                data.FullchainPath = *fullchainPathFlag
        } else {
                fmt.Print("Enter the fullchain certificate path (default: /opt/certs/fullchain.pem): ")
                scanner.Scan()
                data.FullchainPath = strings.TrimSpace(scanner.Text())
                if data.FullchainPath == "" {
                        data.FullchainPath = "/opt/certs/fullchain.pem"
                }
        }

        if *privkeyPathFlag != "" {
                data.PrivkeyPath = *privkeyPathFlag
        } else {
                fmt.Print("Enter the private key path (default: /opt/certs/privkey.pem): ")
                scanner.Scan()
                data.PrivkeyPath = strings.TrimSpace(scanner.Text())
                if data.PrivkeyPath == "" {
                        data.PrivkeyPath = "/opt/certs/privkey.pem"
                }
        }

        // Validate parameters
        errors := validateParams(data, siteType)
        if len(errors) > 0 {
                // Write config file even if there are errors
                tmplContent := localTemplate
                if siteType == "proxy" {
                        tmplContent = proxyTemplate
                }

                tmpl, err := template.New("nginx").Parse(tmplContent)
                if err != nil {
                        fmt.Printf("Error: Failed to parse template: %v\n", err)
                        os.Exit(3)
                }

                file, err := os.Create(configPath)
                if err != nil {
                        fmt.Printf("Error: Failed to create config file %s: %v\n", configPath, err)
                        os.Exit(4)
                }
                defer file.Close()

                if err := tmpl.Execute(file, data); err != nil {
                        fmt.Printf("Error: Failed to write config file: %v\n", err)
                        os.Exit(4)
                }
                fmt.Printf("Configuration file created: %s\n", configPath)

                fmt.Println("Cannot reload Nginx due to the following issues:")
                for _, err := range errors {
                        fmt.Printf("- %s\n", err)
                }
                fmt.Println("Please fix the parameters and use nx2ensite to enable the site.")
                os.Exit(0)
        }

        // Write config file
        tmplContent := localTemplate
        if siteType == "proxy" {
                tmplContent = proxyTemplate
        }

        tmpl, err := template.New("nginx").Parse(tmplContent)
        if err != nil {
                fmt.Printf("Error: Failed to parse template: %v\n", err)
                os.Exit(3)
        }

        file, err := os.Create(configPath)
        if err != nil {
                fmt.Printf("Error: Failed to create config file %s: %v\n", configPath, err)
                os.Exit(4)
        }
        defer file.Close()

        if err := tmpl.Execute(file, data); err != nil {
                fmt.Printf("Error: Failed to write config file: %v\n", err)
                os.Exit(4)
        }
        fmt.Printf("Configuration file created: %s\n", configPath)

        // Prompt for reload (only in interactive mode)
        nonInteractive := *siteNameFlag != "" && *siteTypeFlag != "" && (*fullchainPathFlag != "" || siteType != "proxy") && (*privkeyPathFlag != "" || siteType != "proxy")
        if !nonInteractive {
                fmt.Print("Do you want to enable the site and reload Nginx? (y/n): ")
                scanner.Scan()
                if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
                        fmt.Println("Nginx reload skipped. Use nx2ensite to enable the site.")
                        os.Exit(0)
                }
        }

        // Run nx2ensite to enable the site
        cmd := exec.Command("nx2ensite", data.SiteHostName)
        var stderr bytes.Buffer
        cmd.Stderr = &stderr
        if err := cmd.Run(); err != nil {
                fmt.Printf("Error: Failed to enable site with nx2ensite: %v\n", err)
                fmt.Println(stderr.String())
                os.Exit(5)
        }
        fmt.Printf("Site %s enabled successfully.\n", data.SiteHostName)

        // Test Nginx configuration
        cmd = exec.Command("nginx", "-t")
        stderr.Reset()
        cmd.Stderr = &stderr
        if err := cmd.Run(); err != nil {
                fmt.Println("Error: Nginx configuration test failed. Details:")
                fmt.Println(stderr.String())
                os.Exit(6)
        }

        // Reload Nginx
        cmd = exec.Command("systemctl", "reload", "nginx")
        if err := cmd.Run(); err != nil {
                fmt.Printf("Error: Failed to reload Nginx: %v\n", err)
                os.Exit(7)
        }

        fmt.Println("Nginx reloaded successfully.")
}
