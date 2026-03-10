package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

var (
	titleColor   = color.New(color.FgHiCyan, color.Bold)
	infoColor    = color.New(color.FgWhite)
	successColor = color.New(color.FgHiGreen, color.Bold)
	errorColor   = color.New(color.FgHiRed, color.Bold)
	shardColors  = []*color.Color{
		color.New(color.FgHiBlue, color.Bold),
		color.New(color.FgHiGreen, color.Bold),
		color.New(color.FgHiYellow, color.Bold),
		color.New(color.FgHiMagenta, color.Bold),
	}
)

func main() {
	// Command line flags
	mode := flag.String("mode", "", "Mode: 'gateway' or 'node'")
	shardID := flag.Int("shard", 0, "Shard ID (1-4) for node mode")
	showConfig := flag.Bool("config", false, "Show current configuration from .env")
	flag.Parse()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Try loading from parent directory
		if err := godotenv.Load("../.env"); err != nil {
			errorColor.Println("Warning: No .env file found, using defaults")
		}
	}

	printBanner()

	if *showConfig {
		showConfiguration()
		return
	}

	if *mode == "" {
		showUsage()
		return
	}

	switch *mode {
	case "gateway":
		startGateway()
	case "node":
		if *shardID < 1 || *shardID > 4 {
			errorColor.Println("Error: Shard ID must be between 1 and 4")
			os.Exit(1)
		}
		startNode(*shardID)
	default:
		errorColor.Printf("Unknown mode: %s\n", *mode)
		showUsage()
		os.Exit(1)
	}
}

func printBanner() {
	titleColor.Println("\n╔══════════════════════════════════════════════════════════╗")
	titleColor.Println("║     GIZZARD - Distributed Database Sharding Framework    ║")
	titleColor.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func showUsage() {
	infoColor.Println("Usage:")
	fmt.Println("  ./launcher -mode=gateway              Start the API Gateway")
	fmt.Println("  ./launcher -mode=node -shard=1        Start Shard Node 1")
	fmt.Println("  ./launcher -mode=node -shard=2        Start Shard Node 2")
	fmt.Println("  ./launcher -config                    Show current configuration")
	fmt.Println()
	infoColor.Println("The launcher reads configuration from .env file.")
	fmt.Println("Update .env with the IP addresses of your shard nodes.")
	fmt.Println()
}

func showConfiguration() {
	titleColor.Println("Current Configuration (from .env):")
	fmt.Println()

	// Gateway
	gatewayPort := getEnvInt("GATEWAY_PORT", 8000)
	infoColor.Printf("  Gateway Port: %d\n", gatewayPort)
	fmt.Println()

	// Shards
	infoColor.Println("  Shard Nodes:")
	for i := 1; i <= 4; i++ {
		host := getEnv(fmt.Sprintf("SHARD%d_HOST", i), "localhost")
		port := getEnvInt(fmt.Sprintf("SHARD%d_PORT", i), 8000+i)
		shardColors[(i-1)%4].Printf("    [SHARD %d] %s:%d\n", i, host, port)
	}
	fmt.Println()

	// Local IP
	localIP := getLocalIP()
	successColor.Printf("  Your Local IP: %s\n", localIP)
	infoColor.Println("\n  Share this IP with other laptops to connect to your shard.")
	fmt.Println()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if num, err := strconv.Atoi(value); err == nil {
			return num
		}
	}
	return fallback
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}

func startGateway() {
	gatewayPort := getEnvInt("GATEWAY_PORT", 8000)

	// Build node addresses from .env
	var nodes []string
	for i := 1; i <= 4; i++ {
		host := getEnv(fmt.Sprintf("SHARD%d_HOST", i), "localhost")
		port := getEnvInt(fmt.Sprintf("SHARD%d_PORT", i), 8000+i)
		nodes = append(nodes, fmt.Sprintf("%s:%d", host, port))
	}

	titleColor.Println("Starting API Gateway...")
	fmt.Println()
	infoColor.Printf("  Gateway Port: %d\n", gatewayPort)
	infoColor.Println("  Connected Shards:")
	for i, node := range nodes {
		shardColors[i%4].Printf("    [SHARD %d] %s\n", i+1, node)
	}
	fmt.Println()

	// Build and run gateway
	cmd := exec.Command("./bin/gateway",
		fmt.Sprintf("-port=%d", gatewayPort),
		fmt.Sprintf("-node1=%s", nodes[0]),
		fmt.Sprintf("-node2=%s", nodes[1]),
		fmt.Sprintf("-node3=%s", nodes[2]),
		fmt.Sprintf("-node4=%s", nodes[3]),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println()
		infoColor.Println("Shutting down gateway...")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	successColor.Println("Gateway started! Press Ctrl+C to stop.")
	fmt.Println()
	if err := cmd.Run(); err != nil {
		errorColor.Printf("Gateway exited: %v\n", err)
	}
}

func startNode(shardID int) {
	port := getEnvInt(fmt.Sprintf("SHARD%d_PORT", shardID), 8000+shardID)
	localIP := getLocalIP()

	titleColor.Printf("Starting Shard Node %d...\n", shardID)
	fmt.Println()
	shardColors[(shardID-1)%4].Printf("  [SHARD %d]\n", shardID)
	infoColor.Printf("  Port: %d\n", port)
	infoColor.Printf("  Local IP: %s\n", localIP)
	infoColor.Printf("  Data Directory: ./data/shard%d.db\n", shardID)
	fmt.Println()
	successColor.Println("Share this with the gateway machine:")
	fmt.Printf("  SHARD%d_HOST=%s\n", shardID, localIP)
	fmt.Printf("  SHARD%d_PORT=%d\n", shardID, port)
	fmt.Println()

	// Build and run node
	cmd := exec.Command("./bin/node",
		fmt.Sprintf("-shard=%d", shardID),
		fmt.Sprintf("-port=%d", port),
		"-data=./data",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println()
		infoColor.Printf("Shutting down shard %d...\n", shardID)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	successColor.Printf("Shard %d started! Press Ctrl+C to stop.\n", shardID)
	fmt.Println()
	if err := cmd.Run(); err != nil {
		errorColor.Printf("Shard exited: %v\n", err)
	}
}
