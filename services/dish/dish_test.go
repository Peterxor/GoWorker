package dish

import (
	"dishrank-go-worker/database"
	"dishrank-go-worker/structs"
	"dishrank-go-worker/utils"
	"fmt"
	"testing"
)

func init() {
	// 初始化 env
	var envService utils.EnvService
	envService.InitEnv()
	fmt.Println("參數初始化成功...")

	// 初始化 db
	database.InitDatabasePool()
	fmt.Println("資料庫始化成功...")
}
func TestMemberHasDishService_Start(t *testing.T) {
	defer database.Mysql.Close()

	var ingredientQueueParam structs.DishQueueParam
	ingredientQueueParam.Type = "SINGLE"
	// ingredientQueueParam.Type = "ALL"
	ingredientQueueParam.MemberId = "2c331ae2-d08d-411d-a709-4cbcd52ce44a"
	var memberHasDishService MemberHasDishService
	memberHasDishService.Start(ingredientQueueParam)
}
