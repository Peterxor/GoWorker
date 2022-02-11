package utils

import (
	"dishrank-go-worker/structs"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var EnvConfig *structs.EnviromentModel

type EnvService struct{}

func (e *EnvService) InitEnv() {
	e.loadConfig()
	e.configToModel()
}

func (e *EnvService) loadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			// 進到這邊代表找不到 config.yml
			// 找不到 config.yml 的話就抓取環境變數
			viper.AutomaticEnv()
			viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		} else {

			// 有找到 config.yml 但是發生了其他未知的錯誤
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}
	return
}

func (e *EnvService) configToModel() {
	var config structs.EnviromentModel
	config.Database.Client = viper.GetString("database.client")
	config.Database.Host = viper.GetString("database.host")
	config.Database.User = viper.GetString("database.user")
	config.Database.Password = viper.GetString("database.password")
	config.Database.Db = viper.GetString("database.name")
	config.Database.MaxIdle = uint(viper.GetInt("database.max_idle"))
	config.Database.MaxOpenConn = uint(viper.GetInt("database.max_open_conn"))
	config.Database.MaxLifeTime = viper.GetString("database.max_life_time")
	config.Database.Params = viper.GetString("database.params")
	config.Database.Port = viper.GetString("database.port")
	config.Database.LogEnable = viper.GetInt("database.log_enable")
	config.ConcurrentAmount = viper.GetInt("concurrentAmount")
	config.RabbitMQ.Domain = viper.GetString("rabbitmq.domain")
	config.Log.ElkEnable = viper.GetInt("log.elk.enable")
	config.Log.ElkIndex = viper.GetString("log.elk.index")
	config.Log.ElkURL = viper.GetString("log.elk.url")
	config.Log.LogstashEnable = viper.GetInt("log.logstash.enable")
	config.Log.LogstashURL = viper.GetString("log.logstash.url")
	config.Log.LogstashIndex = viper.GetString("log.logstash.index")
	config.Email.APIUrl = viper.GetString("email.api_url")
	config.Server.AppAPI = viper.GetString("server.app_api")
	config.Router.Port = viper.GetInt("router.port")
	EnvConfig = &config
}
