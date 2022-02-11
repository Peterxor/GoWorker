package main

import (
	"bytes"
	"dishrank-go-worker/database"
	"dishrank-go-worker/models"
	"dishrank-go-worker/router"
	"dishrank-go-worker/services"
	"dishrank-go-worker/services/dish"
	"dishrank-go-worker/services/ingredient"
	"dishrank-go-worker/services/rabbitmq"
	"dishrank-go-worker/services/report"
	"dishrank-go-worker/services/trackLog"
	"dishrank-go-worker/structs"
	"dishrank-go-worker/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"log"

	logLib "dishrank-go-worker/services/log"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	jobProcessChan chan int
	signalChan     chan int
	activeConn     int = 0
)

func main() {

	// 初始化 env
	var envService utils.EnvService
	envService.InitEnv()
	fmt.Println("參數初始化成功...")

	database.InitDatabasePool()
	insertActivityLog("schedule.go.job.init", "dishrank-worker 初始化")
	database.Mysql.Close()
	trackLog.LogTrackInit()

	defer func() {

		// 發送 ELK
		var memberEntity models.Member
		memberEntity.ID = "main"
		memberEntity.Nickname = "主程式"
		var logService logLib.LogService
		logwr := logService.LoggerInit(memberEntity)
		logwr.WithFields(logrus.Fields{"task": "main", "name": memberEntity.Nickname, "member_id": memberEntity.ID}).Error("worker shutdown")
		// 發送 email
		crashEmailAlert()

		fmt.Println("worker shutdown")
	}()

	// 判斷工作還有工作未完成
	// jobProcessChan = make(chan int, 4)

	route := router.Router()

	var wg sync.WaitGroup
	wg.Add(1)
	go route.Run(fmt.Sprintf(":%d", utils.EnvConfig.Router.Port))
	// go healthCheckQueue()

	wg.Add(1)
	go DishrankQueue()

	wg.Wait()
}

func DishrankQueue() {
	conn := rabbitmq.NewConnection("dishrank", []string{"dish", "ingredient", "member-report"})

	if err := conn.Connect(); err != nil {
		panic(err)
	}
	if err := conn.BindQueue(); err != nil {
		panic(err)
	}
	deliveries, err := conn.Consume()
	if err != nil {
		panic(err)
	}

	for q, d := range deliveries {
		go conn.HandleConsumedDeliveries(q, d, DishrankHandler)
	}
	log.Printf(" [ dishrank ] [ dish ingredient member-report ] Waiting for messages. To exit press CTRL+C")
}

func DishrankHandler(c rabbitmq.Connection, q string, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {

		// jobProcessChan <- 1
		trackLog.Info(fmt.Sprintf("Queue[%s] 接受資料: %s\n", q, string(d.Body)), true)

		database.InitDatabasePool()

		activeConn++

		if q == "dish" {
			var dishQueueParam structs.DishQueueParam
			if err := json.Unmarshal(d.Body, &dishQueueParam); err != nil {
				fmt.Println(err.Error())
			}
			// 檢查queue是否正確
			if q != dishQueueParam.QueueType {
				notifyMismatchQueueApi(dishQueueParam.TaskID, q, dishQueueParam.QueueType)
			} else {
				var memberHasDishService dish.MemberHasDishService
				_ = insertActivityLog("schedule.go.job.received", "("+strconv.Itoa(int(dishQueueParam.TaskID))+"), "+"queue name: "+q+", start...")
				if memberHasDishService.Start(dishQueueParam); len(memberHasDishService.Errors) != 0 {
				}
			}

			activeConn--
		}

		if q == "ingredient" {
			var ingredientQueueParam structs.IngredientQueueParam
			if err := json.Unmarshal(d.Body, &ingredientQueueParam); err != nil {
				fmt.Println(err.Error())
			}
			// 檢查queue是否正確
			if q != ingredientQueueParam.QueueType {
				notifyMismatchQueueApi(ingredientQueueParam.TaskID, q, ingredientQueueParam.QueueType)
			} else {
				var memberHasIngrediantService ingredient.MemberHasIngrediantService
				_ = insertActivityLog("schedule.go.job.received", "("+strconv.Itoa(int(ingredientQueueParam.TaskID))+"), "+"queue name: "+q+", start...")
				if memberHasIngrediantService.Start(ingredientQueueParam); len(memberHasIngrediantService.Errors) != 0 {
				}
			}
			activeConn--
		}

		if q == "member-report" {
			var reportQueueParam structs.ReportQueueParam
			if err := json.Unmarshal(d.Body, &reportQueueParam); err != nil {
				fmt.Println(err.Error())
			} else {
				fmt.Println("取得參數： ", reportQueueParam)
			}

			// 檢查queue是否正確
			if q != reportQueueParam.QueueType {
				notifyMismatchQueueApi(reportQueueParam.TaskID, q, reportQueueParam.QueueType)
			} else {
				var reportService report.ReportService
				_ = insertActivityLog("schedule.go.job.received", "("+strconv.Itoa(int(reportQueueParam.TaskID))+"), "+"queue name: "+q+", start...")
				if reportService.Start(reportQueueParam); len(reportService.Errors) != 0 {
				}
			}
			activeConn--
		}

		if activeConn == 0 {
			database.Mysql.Close()
			fmt.Println("資料庫關閉連線")
		}
		// <-jobProcessChan
	}
}

func healthCheckQueue() {
	conn, err := amqp.Dial(utils.EnvConfig.RabbitMQ.Domain)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"health-check", // name
		false,          // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			// n, err := strconv.Atoi(string(d.Body))
			// failOnError(err, "Failed to convert body to integer")

			log.Println(" [health check] be called")
			jobProcessChan <- 1
			response := 1

			err = ch.Publish(
				"",        // exchange
				d.ReplyTo, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType:   "application/json",
					CorrelationId: d.CorrelationId,
					Body:          []byte(strconv.Itoa(response)),
				})
			failOnError(err, "Failed to publish a message")

			d.Ack(false)
			<-jobProcessChan
		}
	}()

	log.Printf(" [ health check ] Awaiting RPC requests")
	<-forever
}

func crashEmailAlert() {
	// URL & Body
	api := utils.EnvConfig.Email.APIUrl
	body, err := json.Marshal("body")
	if err != nil {
		failOnError(err, "Failed to open a channel")

	}

	// POST Request
	resp, err := http.Post(api, "application/json", bytes.NewBuffer(body))
	if err != nil {
		failOnError(err, "Failed to open a channel")

	}
	defer resp.Body.Close()
	// body, err = ioutil.ReadAll(resp.Body)

	// err = json.Unmarshal(body, &exchangeTokenResponse)
	// if err != nil {
	// 	failOnError(err, "Failed to open a channel")
	// }

	return

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// 塞入執行紀錄的 log table
func insertActivityLog(jobname string, data interface{}) error {

	activityLogJSON, _ := json.Marshal(data)

	// 組合塞入到 db 的 model
	location, _ := time.LoadLocation("Asia/Taipei")
	insertTime := time.Now().In(location)
	var activityLogEntity models.ActivityLog
	activityLogEntity.CreatedAt = &insertTime
	activityLogEntity.UpdatedAt = &insertTime
	activityLogEntity.LogName = jobname
	activityLogEntity.Description = "golang-worker log"
	activityLogEntity.Properties = string(activityLogJSON)

	if err := database.Mysql.Create(&activityLogEntity).Error; err != nil {
		return err
	}

	return nil
}

func notifyMismatchQueueApi (taskId uint, queue, queueType string) {
	endpoint := utils.EnvConfig.Server.AppAPI + "/api/v1/workerCallback/mismatchQueue"
	//endpoint := "http://localhost:8000/api/test"
	fmt.Println("callback url", endpoint, "task_id", taskId, "queue", queue)
	body := structs.MismatchQueueResponse{
		TaskId: taskId,
		Queue: queue,
	}
	trackLog.Info(fmt.Sprintf("[MismatchQueue]queue發生錯誤, task_id: %d, mismatch queue: %s, queue_type: %s, callback url: %s", taskId, queue, queueType, endpoint), true)
	_, err := services.HttpRequest(http.MethodPost, endpoint, nil, body)
	if err != nil {
		trackLog.Error(err.Error(), true)
	}
}
