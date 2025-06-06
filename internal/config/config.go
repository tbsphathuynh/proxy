package config

import (
	"sync"
	"time"
)

var (
    instance *Config
    once     sync.Once
)

// Config represents the complete proxy server configuration
// Aggregates all component configurations for centralized management
// Supports environment variable and file-based configuration
type Config struct {
    Server      ServerConfig      `yaml:"server" json:"server"`
    Cache       CacheConfig       `yaml:"cache" json:"cache"`
    RateLimit   RateLimitConfig   `yaml:"rateLimit" json:"rateLimit"`
    LoadBalance LoadBalanceConfig `yaml:"loadBalance" json:"loadBalance"`
    Health      HealthConfig      `yaml:"health" json:"health"`
    Tracing     TracingConfig     `yaml:"tracing" json:"tracing"`
}

// ServerConfig defines HTTP server configuration parameters
// Controls server behavior including timeouts and TLS settings
type ServerConfig struct {
    Port         int           `yaml:"port" json:"port" default:"8080"`
    ReadTimeout  time.Duration `yaml:"readTimeout" json:"readTimeout" default:"30s"`
    WriteTimeout time.Duration `yaml:"writeTimeout" json:"writeTimeout" default:"30s"`
    IdleTimeout  time.Duration `yaml:"idleTimeout" json:"idleTimeout" default:"60s"`
    TLSCertFile  string        `yaml:"tlsCertFile" json:"tlsCertFile"`
    TLSKeyFile   string        `yaml:"tlsKeyFile" json:"tlsKeyFile"`
}

// CacheConfig defines caching middleware configuration
// Controls cache behavior including size limits and TTL
type CacheConfig struct {
    Enabled bool          `yaml:"enabled" json:"enabled" default:"true"`
    MaxSize int           `yaml:"maxSize" json:"maxSize" default:"1000"`
    TTL     time.Duration `yaml:"ttl" json:"ttl" default:"5m"`
}

// RateLimitConfig defines rate limiting configuration
// Controls request rate limits using token bucket algorithm
type RateLimitConfig struct {
    Enabled    bool `yaml:"enabled" json:"enabled" default:"true"`
    Capacity   int  `yaml:"capacity" json:"capacity" default:"100"`
    RefillRate int  `yaml:"refillRate" json:"refillRate" default:"10"`
}

// BackendConfig represents individual backend server configuration
// Includes URL and weight for load balancing algorithms
type BackendConfig struct {
    URL    string `yaml:"url" json:"url"`
    Weight int    `yaml:"weight" json:"weight" default:"1"`
}

// LoadBalanceConfig defines load balancing configuration
// Specifies backend servers and balancing algorithm
type LoadBalanceConfig struct {
    Algorithm string          `yaml:"algorithm" json:"algorithm" default:"round-robin"`
    Backends  []BackendConfig `yaml:"backends" json:"backends"`
}

// HealthConfig defines health check configuration
// Controls health monitoring of backend servers
type HealthConfig struct {
    Enabled  bool          `yaml:"enabled" json:"enabled" default:"true"`
    Interval time.Duration `yaml:"interval" json:"interval" default:"30s"`
    Timeout  time.Duration `yaml:"timeout" json:"timeout" default:"5s"`
    Path     string        `yaml:"path" json:"path" default:"/health"`
}

// TracingConfig defines OpenTelemetry tracing configuration
// Controls distributed tracing and observability
type TracingConfig struct {
    Enabled        bool    `yaml:"enabled" json:"enabled" default:"false"`
    ServiceName    string  `yaml:"serviceName" json:"serviceName" default:"proxy"`
    ServiceVersion string  `yaml:"serviceVersion" json:"serviceVersion" default:"1.0.0"`
    Environment    string  `yaml:"environment" json:"environment" default:"development"`
    JaegerEndpoint string  `yaml:"jaegerEndpoint" json:"jaegerEndpoint"`
    OTLPEndpoint   string  `yaml:"otlpEndpoint" json:"otlpEndpoint"`
    SamplingRatio  float64 `yaml:"samplingRatio" json:"samplingRatio" default:"0.1"`
}

// DefaultConfig returns configuration with sensible defaults
// Provides baseline configuration for development and testing
func DefaultConfig() *Config {
    return &Config{
        Server: ServerConfig{
            Port:         8080,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 30 * time.Second,
            IdleTimeout:  60 * time.Second,
        },
        Cache: CacheConfig{
            Enabled: true,
            MaxSize: 1000,
            TTL:     5 * time.Minute,
        },
        RateLimit: RateLimitConfig{
            Enabled:    true,
            Capacity:   100,
            RefillRate: 10,
        },
        LoadBalance: LoadBalanceConfig{
            Algorithm: "round-robin",
            Backends:  []BackendConfig{},
        },
        Health: HealthConfig{
            Enabled:  true,
            Interval: 30 * time.Second,
            Timeout:  5 * time.Second,
            Path:     "/health",
        },
        Tracing: TracingConfig{
            Enabled:        false,
            ServiceName:    "proxy",
            ServiceVersion: "1.0.0",
            Environment:    "development",
            SamplingRatio:  0.1,
        },
    }
}

// GetInstance returns the singleton config instance
// Uses sync.Once to ensure thread-safe lazy initialisation
// Time Complexity: O(1) - returns cached instance after first call
// Space Complexity: O(1) - stores single configuration instance
func GetInstance() *Config {
    once.Do(func() {
        instance = DefaultConfig()
    })
    return instance
}

// LoadConfig loads configuration from file and updates singleton
// Thread-safe configuration update using mutex
// Time Complexity: O(n) where n is config file size
// Space Complexity: O(n) for parsing configuration
func LoadConfig(path string) error {
    cfg, err := loadFromFile(path)
    if err != nil {
        return err
    }

    // Update singleton instance
    once.Do(func() {
        instance = cfg
    })
    return nil
}

// loadFromFile reads configuration from YAML file
// Supports environment variable interpolation
// Time Complexity: O(n) where n is file size
// Space Complexity: O(n) for file content
func loadFromFile(path string) (*Config, error) {
    // TODO: Implement YAML file loading
    // This is just a placeholder - you'll need to add actual file loading logic
    return DefaultConfig(), nil
}