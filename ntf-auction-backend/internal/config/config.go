package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type App struct {
	Server      ServerConfig      `mapstructure:"server" yaml:"server"`
	Database    DatabaseConfig    `mapstructure:"database" yaml:"database"`
	JWT         JWTConfig         `mapstructure:"jwt" yaml:"jwt"`
	DefaultUser DefaultUserConfig `mapstructure:"default_user" yaml:"default_user"`
	Ethereum    EthereumConfig    `mapstructure:"ethereum" yaml:"ethereum"`
}

type ServerConfig struct {
	Port int `mapstructure:"port" yaml:"port"`
}

type DatabaseConfig struct {
	Driver     string `mapstructure:"driver" yaml:"driver"`
	SQLitePath string `mapstructure:"sqlite_path" yaml:"sqlite_path"`
	MySQLDSN   string `mapstructure:"mysql_dsn" yaml:"mysql_dsn"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret" yaml:"secret"`
	ExpireHours int    `mapstructure:"expire_hours" yaml:"expire_hours"`
}

type DefaultUserConfig struct {
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
}

type EthereumConfig struct {
	RPCURL                string   `mapstructure:"rpc_url" yaml:"rpc_url"`
	TxSignerPrivateKeyHex string   `mapstructure:"tx_signer_private_key" yaml:"tx_signer_private_key"`
	WatchEvents           []string `mapstructure:"watch_events" yaml:"watch_events"`
}

func Load(path string) (App, error) {
	if path == "" {
		path = "config/app.yaml"
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	setDefaults(v)
	bindEnvs(v)

	if err := v.ReadInConfig(); err != nil {
		return App{}, fmt.Errorf("read config: %w", err)
	}

	cfg := defaultConfig()
	if err := v.Unmarshal(&cfg); err != nil {
		return App{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.sqlite_path", "data/app.db")
	v.SetDefault("jwt.secret", "change-this-secret")
	v.SetDefault("jwt.expire_hours", 24)
	v.SetDefault("default_user.username", "admin")
	v.SetDefault("default_user.password", "admin123")
	v.SetDefault("ethereum.watch_events", []string{
		"NFTAuctionListed",
		"NFTAuctionBidPlaced",
		"NFTAuctionEnded",
		"NFTAuctionCancelled",
		"NFTAuctionDelisted",
		"NFTListed",
		"NFTDelisted",
		"NFTPurchased",
		"NFTListingPriceUpdated",
	})
}

func bindEnvs(v *viper.Viper) {
	_ = v.BindEnv("ethereum.rpc_url", "ETH_RPC_URL")
	_ = v.BindEnv("ethereum.tx_signer_private_key", "ETH_TX_SIGNER_PRIVATE_KEY")
}

func defaultConfig() App {
	return App{
		Server: ServerConfig{Port: 8080},
		Database: DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: "data/app.db",
		},
		JWT: JWTConfig{
			Secret:      "change-this-secret",
			ExpireHours: 24,
		},
		DefaultUser: DefaultUserConfig{
			Username: "admin",
			Password: "admin123",
		},
		Ethereum: EthereumConfig{},
	}
}
