package log

import (
	"dishrank-go-worker/models"
	"dishrank-go-worker/utils"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	logrustash "github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-extras/elogrus.v7"
)

type LogService struct{}

func (l *LogService) LoggerInit(memberEntity models.Member) *logrus.Logger {
	now := time.Now()
	logFilePath := ""
	if dir, err := os.Getwd(); err == nil {
		logFilePath = dir + "/logs/" + now.Format("2006-01-02") + "/"
	}
	if err := os.MkdirAll(logFilePath, 0777); err != nil {
		fmt.Println(err.Error())
	}
	logFileName := memberEntity.ID + ".log"
	//日志文件
	fileName := path.Join(logFilePath, logFileName)
	if _, err := os.Stat(fileName); err != nil {
		if _, err := os.Create(fileName); err != nil {
			fmt.Println(err.Error())
		}
	}
	//写入文件
	src, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
	}

	//实例化
	logger := logrus.New()

	//设置输出
	logger.Out = src

	//设置日志级别
	logger.SetLevel(logrus.DebugLevel)

	//设置日志格式
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	if utils.EnvConfig.Log.ElkEnable == 1 {
		client, err := elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{utils.EnvConfig.Log.ElkURL},
		})
		if err != nil {
			logger.Debug(err.Error())
		}
		hook, err := elogrus.NewAsyncElasticHook(client, "dishrank-golang-worker", logrus.DebugLevel, utils.EnvConfig.Log.ElkIndex)
		if err != nil {
			logger.Debug(err.Error())
		} else {
			logger.Hooks.Add(hook)
		}
	}

	if utils.EnvConfig.Log.LogstashEnable == 1 {

		conn, err := net.Dial("udp", utils.EnvConfig.Log.LogstashURL)
		if err != nil {
			logger.Debug(err)
		} else {
			hook := logrustash.New(conn, logrustash.DefaultFormatter(logrus.Fields{"type": "dishrank-golang-worker"}))
			logger.Hooks.Add(hook)
		}
	}

	return logger
}
