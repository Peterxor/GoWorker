package database

import (
	"dishrank-go-worker/utils"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	Mysql *gorm.DB
)

func InitDatabasePool() {
	createPool()
}

func createPool() {

	// 取得config參數
	client := utils.EnvConfig.Database.Client
	host := utils.EnvConfig.Database.Host
	port := utils.EnvConfig.Database.Port
	user := utils.EnvConfig.Database.User
	password := utils.EnvConfig.Database.Password
	dbname := utils.EnvConfig.Database.Db
	parameter := utils.EnvConfig.Database.Params

	var err error
	Mysql, err = gorm.Open(client, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, password, host, port, dbname, parameter))
	if err != nil {
		panic(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	Mysql.DB().SetMaxIdleConns(int(utils.EnvConfig.Database.MaxIdle))

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	Mysql.DB().SetMaxOpenConns(int(utils.EnvConfig.Database.MaxOpenConn))

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	lifeTime, _ := time.ParseDuration(utils.EnvConfig.Database.MaxLifeTime)
	Mysql.DB().SetConnMaxLifetime(lifeTime)

	if utils.EnvConfig.Database.LogEnable == 1 {
		Mysql.LogMode(true)
	} else {
		Mysql.LogMode(false)
	}

}
