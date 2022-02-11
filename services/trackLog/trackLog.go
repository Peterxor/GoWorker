package trackLog

import (
	"dishrank-go-worker/models"
	"dishrank-go-worker/services/log"
	"fmt"
	"github.com/sirupsen/logrus"
)

var logTracker *logrus.Entry

func LogTrackInit()  {
	var memberEntity models.Member
	memberEntity.ID = "tracker"
	memberEntity.Nickname = "log追蹤"
	var trackerService log.LogService
	temp := trackerService.LoggerInit(memberEntity)
	logTracker = temp.WithFields(logrus.Fields{"task": "track", "name": memberEntity.Nickname, "member_id": memberEntity.ID})
}


func Info(message string, needWriteLog bool) {
	if needWriteLog {
		logTracker.Info(message)
	}
	fmt.Println(message)
}

func Error(message string, needWriteLog bool) {
	if needWriteLog {
		logTracker.Error(message)
	}
	fmt.Println(message)
}
