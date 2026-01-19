package database

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// InitDatabaseFromConfig 根据配置文件初始化数据库
func InitDatabaseFromConfig(logger *zap.Logger) error {
	// 从配置文件获取数据库路径
	dbPath := viper.GetString("database.path")
	if dbPath == "" {
		// 默认使用项目根目录下的db.sqlite
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dbPath = filepath.Join(wd, "db.sqlite")
	}

	logger.Info("初始化数据库", zap.String("path", dbPath))

	if err := InitDB(dbPath); err != nil {
		logger.Error("数据库初始化失败", zap.Error(err))
		return err
	}

	logger.Info("数据库初始化成功")
	return nil
}