package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// checkSiteEnabled verifies if the site is already enabled by checking the symbolic link.
func checkSiteEnabled(sitesAvailable, sitesEnabled string) (bool, error) {
	// Check if the symbolic link exists
	if _, err := os.Lstat(sitesEnabled); os.IsNotExist(err) {
		return false, nil
	}

	// Read the symbolic link's target
	target, err := os.Readlink(sitesEnabled)
	if err != nil {
		return false, fmt.Errorf("failed to read symbolic link: %v", err)
	}

	// Compare the target with the sites-available path
	return filepath.Clean(target) == filepath.Clean(sitesAvailable), nil
}

// promptReload asks the user if they want to reload Nginx.
func promptReload(site string) bool {
	fmt.Printf("Site %s is already enabled. Do you want to reload Nginx? (y/n): ", site)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

func main() {
	// Define flags
	help := flag.Bool("help", false, "Display usage information")
	configDir := flag.String("config-dir", "/etc/nginx", "Path to Nginx configuration directory")
	force := flag.Bool("f", false, "Force Nginx reload without prompting if site is already enabled")
	flag.Parse()

	// Display help if --help is passed or no arguments are provided
	if *help || len(flag.Args()) == 0 {
		fmt.Println("Usage: nx2ensite [--config-dir=<path>] [-f] <site_name>")
		fmt.Println("Enable a site in Nginx by creating a symbolic link from sites-available to sites-enabled.")
		fmt.Println("\nOptions:")
		fmt.Println("  --config-dir=<path>  Specify the Nginx configuration directory (default: /etc/nginx)")
		fmt.Println("  -f                   Force Nginx reload without prompting if site is already enabled")
		fmt.Println("  --help               Display this help message")
		fmt.Println("\nExample:")
		fmt.Println("  nx2ensite example")
		fmt.Println("  nx2ensite --config-dir=/custom/nginx -f example")
		os.Exit(0)
	}

	// Get site name from arguments
	site := flag.Args()[0]

	// Construct paths
	sitesAvailable := filepath.Join(*configDir, "sites-available", site+".conf")
	sitesEnabled := filepath.Join(*configDir, "sites-enabled", site+".conf")

	// Check if configuration directory exists
	if _, err := os.Stat(*configDir); os.IsNotExist(err) {
		fmt.Printf("Error: Configuration directory %s does not exist.\n", *configDir)
		os.Exit(1)
	}

	// Check if site configuration file exists
	if _, err := os.Stat(sitesAvailable); os.IsNotExist(err) {
		fmt.Printf("Error: Configuration file %s not found.\n", sitesAvailable)
		os.Exit(2)
	}

	// Test Nginx configuration
	cmd := exec.Command("nginx", "-t")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error: Nginx configuration test failed. Details:")
		fmt.Println(stderr.String())
		os.Exit(3)
	}

	// Check if site is already enabled
	isEnabled, err := checkSiteEnabled(sitesAvailable, sitesEnabled)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(4)
	}

	if isEnabled {
		fmt.Printf("Warning: Site %s is already enabled in sites-enabled.\n", site)
		if !*force {
			if !promptReload(site) {
				fmt.Println("Nginx reload skipped.")
				os.Exit(0)
			}
		}
	} else {
		// Create symbolic link
		if err := os.Symlink(sitesAvailable, sitesEnabled); err != nil {
			fmt.Printf("Error: Failed to create symbolic link: %v\n", err)
			os.Exit(4)
		}
		fmt.Printf("Site %s enabled successfully.\n", site)
	}

	// Test Nginx configuration again before reloading
	cmd = exec.Command("nginx", "-t")
	stderr.Reset()
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error: Nginx configuration test failed after enabling site. Details:")
		fmt.Println(stderr.String())
		os.Exit(5)
	}

	// Reload Nginx
	cmd = exec.Command("systemctl", "reload", "nginx")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: Failed to reload Nginx: %v\n", err)
		os.Exit(6)
	}

	fmt.Println("Nginx reloaded successfully.")
}