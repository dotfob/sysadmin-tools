package main

import (
        "bytes"
        "flag"
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
)

// checkSiteEnabled verifies if the site is enabled by checking the symbolic link.
func checkSiteEnabled(sitesEnabled string) (bool, error) {
        // Check if the symbolic link exists
        if _, err := os.Lstat(sitesEnabled); os.IsNotExist(err) {
                return false, nil
        } else if err != nil {
                return false, fmt.Errorf("failed to check symbolic link: %v", err)
        }
        return true, nil
}

func main() {
        // Define flags
        help := flag.Bool("help", false, "Display usage information")
        configDir := flag.String("config-dir", "/etc/nginx", "Path to Nginx configuration directory")
        flag.Parse()

        // Display help if --help is passed or no arguments are provided
        if *help || len(flag.Args()) == 0 {
                fmt.Println("Usage: nx2dissite2 [--config-dir=<path>] <site_name>")
                fmt.Println("Disable a site in Nginx by removing its symbolic link from sites-enabled.")
                fmt.Println("\nOptions:")
                fmt.Println("  --config-dir=<path>  Specify the Nginx configuration directory (default: /etc/nginx)")
                fmt.Println("  --help               Display this help message")
                fmt.Println("\nExample:")
                fmt.Println("  nx2dissite2 example")
                fmt.Println("  nx2dissite2 --config-dir=/custom/nginx example")
                os.Exit(0)
        }

        // Get site name from arguments
        site := flag.Args()[0]

        // Construct paths
        sitesEnabled := filepath.Join(*configDir, "sites-enabled", site+".conf")

        // Check if configuration directory exists
        if _, err := os.Stat(*configDir); os.IsNotExist(err) {
                fmt.Printf("Error: Configuration directory %s does not exist.\n", *configDir)
                os.Exit(1)
        }

        // Check if site is enabled
        isEnabled, err := checkSiteEnabled(sitesEnabled)
        if err != nil {
                fmt.Printf("Error: %v\n", err)
                os.Exit(2)
        }
        if !isEnabled {
                fmt.Printf("Error: Site %s is not enabled or the symbolic link does not exist.\n", site)
                os.Exit(2)
        }

        // Remove the symbolic link
        if err := os.Remove(sitesEnabled); err != nil {
                fmt.Printf("Error: Failed to remove symbolic link: %v\n", err)
                os.Exit(3)
        }
        fmt.Printf("Site %s disabled successfully.\n", site)

        // Test Nginx configuration
        cmd := exec.Command("nginx", "-t")
        var stderr bytes.Buffer
        cmd.Stderr = &stderr
        if err := cmd.Run(); err != nil {
                fmt.Println("Error: Nginx configuration test failed. Details:")
                fmt.Println(stderr.String())
                os.Exit(4)
        }

        // Reload Nginx
        cmd = exec.Command("systemctl", "reload", "nginx")
        if err := cmd.Run(); err != nil {
                fmt.Printf("Error: Failed to reload Nginx: %v\n", err)
                os.Exit(5)
        }

        fmt.Println("Nginx reloaded successfully.")
}