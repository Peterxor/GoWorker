package check

import (
	"dishrank-go-worker/services/rabbitmq"
	"dishrank-go-worker/services/trackLog"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"
	"time"
)

type AliveResponse struct {
	Success  bool   `json:"success"`
	Messsage string `json:"message"`
	Info     CheckInfo `json:"info"`
}

type CheckInfo struct {
	Queues []string `json:"queue"`
	RoutineNum int `json:"routine_num"`
}

func CheckAlive(c *gin.Context) {
	rabbitConn := rabbitmq.GetConnection("dishrank")
	resMsg := "main thread alive"
	checkInfo := CheckInfo{}
	//檢查mq實體是否在連線池
	if rabbitConn != nil {
		// 檢查mq連線
		if rabbitConn.Conn == nil {
			resMsg = "Api detect Connection lost, Reconnecting.."
			trackLog.Error(resMsg, false)
			if err := rabbitConn.Reconnect(); err != nil {
				resMsg = fmt.Sprintf("reconnect rabbit fail: %s", err.Error())
				trackLog.Error(resMsg, false)
			}
		}
		//檢查mq channel
		if rabbitConn.Channel != nil {
			for _, q := range rabbitConn.Queues {
				//檢查每一個queue
				queue, queueErr := rabbitConn.Channel.QueueInspect(q)
				if queueErr != nil {
					resMsg = fmt.Sprintf("Queue[%s] error: %s\n", q, queueErr.Error())
					trackLog.Error(resMsg, false)
				} else {
					// queue的狀態
					queueJson, _ := json.Marshal(queue)
					checkInfo.Queues = append(checkInfo.Queues, string(queueJson))
					trackLog.Info(fmt.Sprintf("Queue[%s]: %s\n", q, queueJson), false)
				}
			}
		} else {
			resMsg = "Channel get fail"
			trackLog.Error(resMsg, false)
		}
		// 花1秒檢查是否重連線
		select {
		case err := <-rabbitConn.ApiErr:
			trackLog.Error(fmt.Sprintf("api error: %s\n", err.Error()), false)
			if err := rabbitConn.Reconnect(); err != nil {
				resMsg = fmt.Sprintf("reconnect rabbit fail: %s\n", err.Error())
				trackLog.Error(resMsg, false)
			}
		case <-time.After(time.Second * 1):
		}
	} else {
		resMsg = "Get connection pool fail"
		trackLog.Error(resMsg, false)
	}

	// 檢查gorutine數目
	trackLog.Info(fmt.Sprintf("goroutine number: %d\n", runtime.NumGoroutine()), false)
	checkInfo.RoutineNum = runtime.NumGoroutine()

	c.JSON(http.StatusOK, AliveResponse{true, resMsg, checkInfo})
	return
}
