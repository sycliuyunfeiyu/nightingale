package tools

import (
	"fmt"
	"github.com/ccfos/nightingale/v6/conf"
	"github.com/ccfos/nightingale/v6/storage"
)

var RedisClient storage.Redis

func InitTools(configDir string, cryptoKey string) error {
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return fmt.Errorf("failed to init config: %v", err)
	}
	RedisClient, err = storage.NewRedis(config.Redis)
	return err
}
