package config

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all backend environment configuration
type Config struct {
	Host             string
	Port             int
	DBDriver         string // "mysql" or "sqlite"
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBPath           string // SQLite database file path fallback
	SystemOneURL     string
	VibeasikCloudURL string // vibeasik.com cloud API endpoint
}

var globalConfig *Config

// LoadConfig loads environment variables from .env file or system environment
func LoadConfig() *Config {
	// 1. Try loading .env in current working directory
	_ = godotenv.Load(".env")

	// 2. Fallback check home directory ~/.vibeasik/.env or ~/.projectmanager/.env
	if home, err := os.UserHomeDir(); err == nil {
		_ = godotenv.Load(home + "/.vibeasik/.env")
		_ = godotenv.Load(home + "/.projectmanager/.env")
	}

	defaultDBPath := "data/projects.db"
	if _, err := os.Stat("data"); err != nil {
		if home, err := os.UserHomeDir(); err == nil {
			defaultDBPath = filepath.Join(home, ".vibeasik", "projects.db")
		}
	}

	cfg := &Config{
		Host:             getEnv("HOST", "127.0.0.1"),                             // bawaan hanya localhost; LAN harus diaktifkan sadar
		Port:             getEnvAsInt("VIBEASIK_PORT", getEnvAsInt("PORT", 8765)), // VIBEASIK_PORT diutamakan: PORT generik di shell/.env proyek lain sering bentrok
		DBDriver:         strings.ToLower(getEnv("DB_DRIVER", "sqlite")),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "3306"),
		DBUser:           getEnv("DB_USER", "root"),
		DBPassword:       getEnv("DB_PASSWORD", "root"),
		DBName:           getEnv("DB_NAME", "projectmanager_db"), // nama lama dipertahankan agar DB yang ada tetap terbaca
		DBPath:           getEnv("DB_PATH", defaultDBPath),
		SystemOneURL:     getEnv("SYSTEM_ONE_URL", "http://localhost:8750/v1/systemone"),
		VibeasikCloudURL: getEnv("VIBEASIK_CLOUD_URL", "http://localhost:8770"),
	}

	globalConfig = cfg
	return cfg
}

// GetConfig returns current global config or loads it if nil
func GetConfig() *Config {
	if globalConfig == nil {
		return LoadConfig()
	}
	return globalConfig
}

// GetLocalIPs returns list of non-loopback IPv4 addresses on the host for LAN access display
func GetLocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
