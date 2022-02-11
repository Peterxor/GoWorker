package dish

import (
	"dishrank-go-worker/database"
	"dishrank-go-worker/enums"
	"dishrank-go-worker/models"
	"dishrank-go-worker/services"
	"dishrank-go-worker/services/log"
	"dishrank-go-worker/structs"
	"dishrank-go-worker/utils"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	gormbulk "github.com/t-tiger/gorm-bulk-insert/v2"
)

var (
	// dishHasOilEntities   []models.DishHasOil
	dishEntities              []models.Dish
	dishHasIngredientEntities []models.DishHasIngredient
	dishHasCookMethodEntities []models.DishHasCookMethod
	concurrentGoroutines      chan struct{}
	now                       time.Time
	processResult             map[string][]string
	location                  *time.Location
	mutex                         = &sync.Mutex{}
	retryCount                int = 0
	retryAttempTimes          int = 3
)

type MemberHasDishService struct {
	sync.Mutex
	dishQueueParam structs.DishQueueParam
	Errors         []structs.ErrorModel
}

// 處理資料的主要進入點
func (m *MemberHasDishService) Start(dishQueueParam structs.DishQueueParam) {

	if dishQueueParam.IsDie {
		panic(nil)
	}
	m.dishQueueParam = dishQueueParam

	// 初始化
	location, _ = time.LoadLocation("Asia/Taipei")
	now = time.Now().In(location)
	// fmt.Println("time now: ", time.Now())
	// fmt.Println("time + 8: ", now)
	processResult = make(map[string][]string)
	dishEntities = nil
	dishHasIngredientEntities = nil
	dishHasCookMethodEntities = nil
	// 針對特定用戶
	if dishQueueParam.Type == enums.ProcessSingle {

		fmt.Println("[dish] 處理方式： ", dishQueueParam.Type, "task_id", dishQueueParam.TaskID)

		if memberEntities, err := m.getMemberEntity(dishQueueParam.MemberId); err != nil {
			m.insertActivityLog(&memberEntities[0], false)
			return
		} else {
			if len(memberEntities) > 0 {
				fmt.Println("[dish] 開始處理用戶： ", memberEntities[0].ID)
				m.process(memberEntities[0], nil)
				// 查看是否有 Error log
				if len(m.Errors) == 0 {
					if insertActivityLogError := m.insertActivityLog(&memberEntities[0], true); insertActivityLogError == nil {
						m.JobDoneNotify()
					} else {
						fmt.Println(insertActivityLogError)
					}
				} else {
					if insertActivityLogError := m.insertActivityLog(&memberEntities[0], false); insertActivityLogError == nil {
						m.JobDoneNotify()
					}else {
						fmt.Println(insertActivityLogError)
					}
				}
			} else {
				fmt.Println("[dish] 查無此用戶：", dishQueueParam.MemberId)
				m.JobDoneNotify()
			}
		}
		fmt.Println("[dish] Done!!")
	}

	// 針對所有用戶
	if dishQueueParam.Type == enums.ProcessAll {
		if memberEntities, err := m.getMemberEntity(""); err != nil {
			m.insertActivityLog(nil, false)
			return
		} else {
			fmt.Println("[dish] 處理方式： ", dishQueueParam.Type, len(memberEntities), "task_id", dishQueueParam.TaskID)

			var wg sync.WaitGroup
			wg.Add(len(memberEntities))

			// 限制啟動 goroutine 的數量
			concurrentGoroutines = make(chan struct{}, utils.EnvConfig.ConcurrentAmount)

			// 針對某個特定用戶
			for _, memberEntity := range memberEntities {
				concurrentGoroutines <- struct{}{}
				// 針對某個用戶處理各自的 dish 的 transaction 資料
				go m.process(memberEntity, &wg)
			}
			wg.Wait()
			close(concurrentGoroutines)

			// 完成後，紀錄 log
			if len(m.Errors) == 0 {
				// 完成後，紀錄 log
				if insertActivityLogError := m.insertActivityLog(nil, true); insertActivityLogError == nil {
					m.JobDoneNotify()
				}
			} else {
				if insertActivityLogError := m.insertActivityLog(nil, false); insertActivityLogError == nil {
					m.JobDoneNotify()
				}
			}

			fmt.Println("Dish Done!!")
		}
	}
	// m.JobDoneNotify()
}

// 處理單個用戶的邏輯
func (m *MemberHasDishService) process(memberEntity models.Member, wg *sync.WaitGroup) {
	if wg != nil {

		fmt.Printf("開始處理用戶： %s, backend_id: %d\n", memberEntity.Nickname, memberEntity.BackendID)
		defer func() {
			wg.Done()
			fmt.Printf("完成： %s, backend_id: %d\n", memberEntity.Nickname, memberEntity.BackendID)
			<-concurrentGoroutines
		}()
	}

	var logService log.LogService
	logwg := logService.LoggerInit(memberEntity)
	logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("開始準備資料")

	// 假設計算過程有任何問題，這邊處理重新計算
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("發生未預期錯誤：", err)
			retryCount++
			if retryCount <= retryAttempTimes {
				fmt.Println(memberEntity.Nickname, "重新計算")
				m.process(memberEntity, nil)
			}
			logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "error_message": err, "attemp_times": retryCount}).Error("發生錯誤，重新計算")
		}
	}()

	memberIngredients, err := m.memberHasIngredientData(&memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢 會員喜好 ingredients 食材時發生錯誤", err.Error())
		return
	}



	// 這個會員的過敏原
	memberHasCookMethods, err := m.memberHasCookMethodIdList(&memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢過敏原時發生錯誤", err.Error())
		return
	}

	// 取得餐點總表
	if err := m.getDishEntity(&memberEntity); err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("取得餐點總表時發生錯誤", err.Error())
		return
	}

	// 準備要新增的資料
	var memberHasDishInsertEntities []models.MemberHasDish


	// 進行每一個餐點的資料檢查
	for _, dishEntity := range dishEntities {

		// 預設要做
		do := true
		points := 0
		dishType := enums.NeutralType

		// 餐點含有的食材先找出來
		dishHasIngredientIds, err := m.dishHasIngredientIdList(&memberEntity, dishEntity, dishEntities)
		if err != nil {
			logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("餐點含有的食材先找出來的時後錯誤", err.Error())
			return
		}

		// 檢查烹調方式是否命中
		//dishHasCookMethodIds, cookErr := m.dishHasCookMethodIdList(&memberEntity, dishEntity, dishEntities)
		//if cookErr != nil {
		//	logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("餐點含有的烹調方式找出來的時後錯誤", err.Error())
		//	return
		//}
		//
		//for _,id := range dishHasCookMethodIds {
		//	if _, ok := memberHasCookMethods[id]; ok {
		//		entity := m.getMemberHasDishEntity(memberEntity, dishEntity, enums.DeclineType, 0)
		//		memberHasDishInsertEntities = append(memberHasDishInsertEntities, entity)
		//		do = false
		//		break
		//	}
		//}
		if _,ok := memberHasCookMethods[dishEntity.CookMethodId]; ok {
			entity := m.getMemberHasDishEntity(memberEntity, dishEntity, enums.DeclineType, 0)
			memberHasDishInsertEntities = append(memberHasDishInsertEntities, entity)
			do = false
		}
		// 烹調方式命中，不往下算了
		if !do {
			continue
		}

		// 檢查菜餚食材
		for _, id := range dishHasIngredientIds {
			if val, ok := memberIngredients[id]; ok {
				switch val.Type {
				case enums.DeclineType:
					entity := m.getMemberHasDishEntity(memberEntity, dishEntity, enums.DeclineType, 0)
					memberHasDishInsertEntities = append(memberHasDishInsertEntities, entity)
					do = false
				case enums.SuggestType:
					points += 6
					dishType = enums.SuggestType
				default:
					points += 1
				}
			}
			// 如果 type 是 decline 則跳出循環
			if !do {
				break
			}
		}
		// 如果 type 不為 decline 則 插入新的結果
		if do {
			entity := m.getMemberHasDishEntity(memberEntity, dishEntity, dishType, points)
			memberHasDishInsertEntities = append(memberHasDishInsertEntities, entity)
		}
	}

	m.Lock()
	defer m.Unlock()

	logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("交易開始")

	tx := database.Mysql.Begin()
	defer func() {
		if r := recover(); r != nil {
			logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗: panic")
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		m.handleError(&memberEntity, err)
		return
	}

	// 先把 dishID 取出來
	var dishIDs []int64
	for _, memberHasDishInsertEntity := range memberHasDishInsertEntities {
		dishIDs = append(dishIDs, memberHasDishInsertEntity.DishID)
	}

	// 把 dish_has_restaurant 的 table 撈出來，下面可以帶入 restaurant_id
	var dishHasRestaurantEntities []models.DishHasRestaurant
	if err := tx.Joins("left join restaurants on dish_has_restaurants.restaurant_id = restaurants.id").
		Where("dish_has_restaurants.dish_id in (?) " +
			"and ( restaurants.organization_type = ? or (restaurants.organization_type = ? and restaurants.parent_id is not null))", dishIDs, "SINGLE", "CHAIN").
		Find(&dishHasRestaurantEntities).Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		m.handleError(&memberEntity, err)
		return
	}

	// 標示版號
	var version = time.Now().Unix()
	var insertRecords []interface{}
	for _, memberHasDishInsertEntity := range memberHasDishInsertEntities {

		// 因為 dishID 對應的 restaurant_id 是多對多，所以這是用 loop 的方式，有幾組對應就產生幾組 member_has_dish 的資料
		for _, dishHasRestaurantEntity := range dishHasRestaurantEntities {
			if dishHasRestaurantEntity.DishID == memberHasDishInsertEntity.DishID {
				memberHasDishInsertEntity.Version = version
				memberHasDishInsertEntity.RestaurantID = dishHasRestaurantEntity.RestaurantID
				insertRecords = append(insertRecords, memberHasDishInsertEntity)
			}
		}
	}

	// 找出這個用戶的推薦食材，這時後已經有 restaurant_id 可以用了
	recommendScoreMap := make(map[int64]int)
	for _, insertRecord := range insertRecords {
		memberHasDishInsertEntity := insertRecord.(models.MemberHasDish)
		if _, ok := recommendScoreMap[memberHasDishInsertEntity.RestaurantID]; !ok {
			recommendScoreMap[memberHasDishInsertEntity.RestaurantID] = 0
		}
		// 去組合該餐廳對應的推薦數
		if memberHasDishInsertEntity.Type == enums.SuggestType {
			recommendScoreMap[memberHasDishInsertEntity.RestaurantID] += 1
		}
	}

	for key, element := range recommendScoreMap {
		// 該餐廳有的全部食材
		restaurantHasTotalDish := 0
		if err := tx.Where("restaurant_id = ?", key).Model(models.DishHasRestaurant{}).Count(&restaurantHasTotalDish).Error; err != nil {
			logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
			m.handleError(&memberEntity, err)
			return
		}

		sortRecommendEntity := new(models.SortRecommend)
		sortRecommendEntity.RestaurantID = key
		sortRecommendEntity.MemberID = memberEntity.BackendID
		sortRecommendEntity.SuggestDishCount = element
		sortRecommendEntity.TotalDishCount = restaurantHasTotalDish
		sortRecommendEntity.SuggestRatio = float64(element) / float64(restaurantHasTotalDish)
		sortRecommendEntity.Recommend = math.Sqrt(float64(sortRecommendEntity.SuggestDishCount)) * float64(sortRecommendEntity.SuggestRatio)

		result := tx.
			Where("member_id = ? and restaurant_id = ?", sortRecommendEntity.MemberID, sortRecommendEntity.RestaurantID).
			Model(models.SortRecommend{}).
			Update(sortRecommendEntity)
		if result.Error != nil {
			logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
			m.handleError(&memberEntity, err)
			return
		}

		if result.RowsAffected == 0 {
			err = tx.Create(&sortRecommendEntity).Error
			if err != nil {
				logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
				m.handleError(&memberEntity, err)
				return
			}
		}
	}

	// 批次 insert
	if err := gormbulk.BulkInsert(tx, insertRecords, 3000); err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		tx.Rollback()
		m.handleError(&memberEntity, err)
		return
	}

	// 刪除 finished 的資料
	if err := tx.Where(models.MemberHasDish{Status: enums.FinishedStatus, MemberID: memberEntity.BackendID}).Delete(models.MemberHasDish{}).Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		tx.Rollback()
		m.handleError(&memberEntity, err)
		return
	}

	// 將資料更新成 finished
	if err := tx.Model(&models.MemberHasDish{}).Where(models.MemberHasDish{Status: enums.QueueStatus, MemberID: memberEntity.BackendID}).Update(models.MemberHasDish{Status: enums.FinishedStatus}).Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		tx.Rollback()
		m.handleError(&memberEntity, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		m.handleError(&memberEntity, err)
		return
	}

	// 如果數量不一致的話，發一個 ELK LOG
	if len(dishHasRestaurantEntities) != len(insertRecords) {
		logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "db_total_dish": len(dishEntities), "created_dis_total": len(insertRecords)}).Error("計算出來的筆數, 與資料庫筆數不一致")
	}

	logwg.WithFields(logrus.Fields{"task": "dish", "task_id": m.dishQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "db_total_dish": len(dishEntities), "created_dis_total": len(insertRecords)}).Info("交易完成")

	// 把處理好的用戶 id 加到結果參數中
	if _, ok := processResult["ok"]; !ok {
		var arr []string
		arr = append(arr, memberEntity.ID)
		processResult["ok"] = arr
	} else {
		processResult["ok"] = append(processResult["ok"], memberEntity.ID)
	}
}

// 餐點總表
func (m *MemberHasDishService) getDishEntity(memberEntity *models.Member) error {

	mutex.Lock()
	if len(dishEntities) == 0 {
		subquery2 := database.Mysql.Model(models.Restaurant{}).Select("id").Where(models.Restaurant{OrganizationType: "SINGLE"}).Or("organization_type = ? and parent_id is not null", "CHAIN").QueryExpr()
		var dishHasRestaurantEntity models.DishHasRestaurant
		subQuery1 := database.Mysql.Table(dishHasRestaurantEntity.TableName()).Where("restaurant_id in (?)", subquery2).Select("dish_id").QueryExpr()
		if errors := database.Mysql.Debug().Where(models.Dish{AuditType: enums.AuditType, IsAppVisible: 1, Analyzed: 1}).Where("id in (?)", subQuery1).Find(&dishEntities).GetErrors(); len(errors) != 0 {
			m.handleError(memberEntity, errors[0])
			return errors[0]
		}
	}
	mutex.Unlock()
	return nil
}

// 取得會員
func (m *MemberHasDishService) getMemberEntity(memberID string) ([]models.Member, error) {

	//subquery := database.Mysql.Select("member_id").Model(models.Quotation{}).Where("quotations.activation_code = members.activation_code and quotations.active = 1").QueryExpr()
	var memberEntities []models.Member
	dateline := time.Now().Add(time.Hour * (-7 * 24))
	if memberID == "" {
		if errors := database.Mysql.Debug().Where("members.expired_at > ?", dateline).Find(&memberEntities).GetErrors(); len(errors) != 0 {
			m.handleError(nil, errors[0])
			return nil, errors[0]
		}
	} else {
		if errors := database.Mysql.Debug().Where("expired_at > ? and id = ?", dateline, memberID).Find(&memberEntities).GetErrors(); len(errors) != 0 {
			var memberEntity models.Member
			memberEntity.ID = memberID
			memberEntity.Nickname = "查詢用戶資料表(members)有誤"
			m.handleError(&memberEntity, errors[0])
			return nil, errors[0]
		}
	}
	return memberEntities, nil
}

// 塞入執行紀錄的 log table
func (m *MemberHasDishService) insertActivityLog(memberEntity *models.Member, result bool) error {

	expireMemberCount := 0
	if err := database.Mysql.Table(memberEntity.TableName()).Where("expired_at < ?", now).Count(&expireMemberCount).Error; err != nil {
		return err
	}

	totalMemberCount := 0
	if err := database.Mysql.Table(memberEntity.TableName()).Count(&totalMemberCount).Error; err != nil {
		return err
	}

	fmt.Println("ok: ", len(processResult["ok"]))
	fmt.Println("fail: ", len(processResult["fail"]))
	fmt.Println("total: ", totalMemberCount)
	fmt.Println("expire: ", expireMemberCount)

	// 組合 json 訊息
	var activityLogJSONModel structs.ActivityLogJsonModel

	// 有用戶就放用戶資料
	if memberEntity != nil {
		activityLogJSONModel.MemberID = memberEntity.ID
		activityLogJSONModel.MemberName = memberEntity.Nickname
	}

	activityLogJSONModel.Type = m.dishQueueParam.Type
	activityLogJSONModel.Result = result

	if activityLogJSONModel.Result {
		activityLogJSONModel.Message = "ok"
	} else {
		if activityLogJSONModel.Type == enums.ProcessSingle {
			activityLogJSONModel.Message = m.Errors[0].ErrorMessage
		}

		if activityLogJSONModel.Type == enums.ProcessAll {
			var errorModel structs.ErrorModel
			errorModel.ErrorMessage = "查詢 ELK"
			activityLogJSONModel.Messages = append(activityLogJSONModel.Messages, errorModel)
		}
	}

	activityLogJSONModel.Statistic.TotalMember = totalMemberCount
	activityLogJSONModel.Statistic.ExpiredMember = expireMemberCount
	activityLogJSONModel.Statistic.OKMember = len(processResult["ok"])
	activityLogJSONModel.Statistic.FailMember = len(processResult["fail"])

	activityLogJSON, _ := json.Marshal(activityLogJSONModel)

	// 組合塞入到 db 的 model
	insertTime := time.Now().In(location)
	var activityLogEntity models.ActivityLog
	activityLogEntity.CreatedAt = &insertTime
	activityLogEntity.UpdatedAt = &insertTime
	activityLogEntity.LogName = "schedule.go.dish"
	activityLogEntity.Description = "會員菜餚計算"
	activityLogEntity.Properties = string(activityLogJSON)

	m.dishQueueParam.Result = string(activityLogJSON)

	if err := database.Mysql.Create(&activityLogEntity).Error; err != nil {
		return err
	}

	return nil
}

// 取得用戶菜餚
func (m *MemberHasDishService) getMemberHasDishEntity(memberEntity models.Member, dishEntity models.Dish, typepo string, points int) models.MemberHasDish {

	var MemberHasDishEntity models.MemberHasDish
	MemberHasDishEntity.DishID = dishEntity.ID
	MemberHasDishEntity.MemberID = memberEntity.BackendID
	MemberHasDishEntity.Type = typepo
	MemberHasDishEntity.Status = enums.QueueStatus
	MemberHasDishEntity.Points = points

	return MemberHasDishEntity
}

func (m *MemberHasDishService) memberHasIngredientData(memberEntity *models.Member) (map[int64]models.MemberHasIngredient, error) {
	result := make(map[int64]models.MemberHasIngredient)
	var memberHasIngredientManualEntities []models.MemberHasIngredient
	if errors := database.Mysql.Where(models.MemberHasIngredient{MemberID: memberEntity.BackendID}).Find(&memberHasIngredientManualEntities).GetErrors(); len(errors) != 0 {
		m.handleError(memberEntity, errors[0])
		return nil, errors[0]
	}
	for _, memberHasIngredientManualEntity := range memberHasIngredientManualEntities {
		result[memberHasIngredientManualEntity.IngredientID] = memberHasIngredientManualEntity
	}

	return result, nil
}

// 取得會員的烹調方式清單
func (m *MemberHasDishService) memberHasCookMethodIdList(memberEntity *models.Member) (map[int64]int, error) {

	result := make(map[int64]int)
	var memberHasCookMethodEntities []models.MemberHasCookMethod

	if errors := database.Mysql.Where(models.MemberHasCookMethod{MemberID: memberEntity.ID}).Find(&memberHasCookMethodEntities).GetErrors(); len(errors) != 0 {
		m.handleError(memberEntity, errors[0])
		return nil, errors[0]
	}

	for _, memberHasCookMethodEntity := range memberHasCookMethodEntities {
		result[memberHasCookMethodEntity.CookMethodID] = 1
	}
	return result, nil
}

// 餐點含有的食材
func (m *MemberHasDishService) dishHasIngredientIdList(memberEntity *models.Member, dishEntity models.Dish, dishEntities []models.Dish) ([]int64, error) {

	var ids []int64
	// 撈過就不撈了
	//if len(dishHasIngredientEntities) == 0 {
	//	var tempDishHasIngredientEntities []models.DishHasIngredient
	//	var resturantEntity models.Restaurant
	//	subQuery1 := database.Mysql.Table(resturantEntity.TableName()).Where(&models.Restaurant{OrganizationType: "SINGLE"}).Or("organization_type = ? and parent_id is not null", "CHAIN").Select("id").QueryExpr()
	//	//subQuery2 := database.Mysql.Table(dishEntity.TableName()).Where(models.Dish{AuditType: enums.AuditType}).Where("restaurant_id in (?)", subQuery1).Select("id").QueryExpr()
	//	var dishHasRestaurantEntity models.DishHasRestaurant
	//	subQuery2 := database.Mysql.Table(dishHasRestaurantEntity.TableName()).Where("restaurant_id in (?)", subQuery1).Select("dish_id").QueryExpr()
	//	if errors := database.Mysql.Debug().Where("dish_id in (?)", subQuery2).Find(&tempDishHasIngredientEntities).GetErrors(); len(errors) != 0 {
	//		m.handleError(memberEntity, errors[0])
	//		return nil, errors[0]
	//	}
	//	dishHasIngredientEntities = tempDishHasIngredientEntities
	//	//dishJson, _ := json.Marshal(dishHasIngredientEntities)
	//	//fmt.Printf("model: %s\n", string(dishJson))
	//}
	if len(dishHasIngredientEntities) == 0 {
		var tempDishHasIngredientEntities []models.DishHasIngredient
		var resturantEntitys []models.Restaurant
		//subQuery1 := database.Mysql.Table(resturantEntity.TableName()).Where(&models.Restaurant{OrganizationType: "SINGLE"}).Or("organization_type = ? and parent_id is not null", "CHAIN").Select("id").QueryExpr()
		//subQuery2 := database.Mysql.Table(dishEntity.TableName()).Where(models.Dish{AuditType: enums.AuditType}).Where("restaurant_id in (?)", subQuery1).Select("id").QueryExpr()
		if errors2 := database.Mysql.Debug().Where("organization_type = ? or (organization_type = ? and parent_id is not null)", "SINGLE", "CHAIN").Find(&resturantEntitys).GetErrors(); len(errors2) != 0 {
			m.handleError(memberEntity, errors2[0])
			return nil, errors2[0]
		}
		for _, resturant := range resturantEntitys {
			var dishHasRestaurantEntity models.DishHasRestaurant
			var tempDishIgredients []models.DishHasIngredient
			subQuery2 := database.Mysql.Table(dishHasRestaurantEntity.TableName()).Where("restaurant_id = ?", resturant.ID).Select("dish_id").QueryExpr()
			if errors := database.Mysql.Where("dish_id in (?)", subQuery2).Find(&tempDishIgredients).GetErrors(); len(errors) != 0 {
				m.handleError(memberEntity, errors[0])
				return nil, errors[0]
			}
			tempDishHasIngredientEntities = append(tempDishHasIngredientEntities, tempDishIgredients...)
		}
		dishHasIngredientEntities = tempDishHasIngredientEntities
		//dishJson, _ := json.Marshal(dishHasIngredientEntities)
		//fmt.Printf("model: %s\n", string(dishJson))
	}

	for _, dishHasIngredientEntity := range dishHasIngredientEntities {
		// 這是為了避免又重複的ingredients 暫時先註解
		//inIds := false
		//for _, id := range ids {
		//	if id == dishHasIngredientEntity.IngredientID {
		//		inIds = true
		//		break
		//	}
		//}
		if dishHasIngredientEntity.DishID == dishEntity.ID {
			ids = append(ids, dishHasIngredientEntity.IngredientID)
		}
	}
	return ids, nil
}

// 餐點含有的烹調方式
func (m *MemberHasDishService) dishHasCookMethodIdList(memberEntity *models.Member, dishEntity models.Dish, dishEntities []models.Dish) ([]int64, error) {

	var ids []int64
	// 撈過就不撈了
	if len(dishHasCookMethodEntities) == 0 {
		var tempDishHasCookMethodEntities []models.DishHasCookMethod
		var resturantEntity models.Restaurant
		subQuery1 := database.Mysql.Table(resturantEntity.TableName()).Where(&models.Restaurant{OrganizationType: "SINGLE"}).Or("organization_type = ? and parent_id is not null", "CHAIN").Select("id").QueryExpr()
		//subQuery2 := database.Mysql.Table(dishEntity.TableName()).Where(models.Dish{AuditType: enums.AuditType}).Where("restaurant_id in (?)", subQuery1).Select("id").QueryExpr()
		var dishHasRestaurantEntity models.DishHasRestaurant
		subQuery2 := database.Mysql.Table(dishHasRestaurantEntity.TableName()).Where("restaurant_id in (?)", subQuery1).Select("dish_id").QueryExpr()
		if errors := database.Mysql.Where("dish_id in (?)", subQuery2).Find(&tempDishHasCookMethodEntities).GetErrors(); len(errors) != 0 {
			m.handleError(memberEntity, errors[0])
			return nil, errors[0]
		}
		dishHasCookMethodEntities = tempDishHasCookMethodEntities
	}

	for _, dishHasCookMethodEntity := range dishHasCookMethodEntities {
		if dishHasCookMethodEntity.DishID == dishEntity.ID {
			ids = append(ids, dishHasCookMethodEntity.CookMethodID)
		}
	}
	return ids, nil
}

func (m *MemberHasDishService) JobDoneNotify() {

	endpoint := utils.EnvConfig.Server.AppAPI + "/api/v1/workerCallback/dish"
	fmt.Println("callback url", endpoint, "task_id", m.dishQueueParam.TaskID)
	_, err := services.HttpRequest(http.MethodPost, endpoint, nil, m.dishQueueParam)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// 餐點含有的食材 (這個先跳過，備用)
// func (m *MemberHasDishService) dishHasOilIdList(dishEntity models.Dish) []int64 {

// 	var ids []int64
// 	for _, dishHasOilEntity := range dishHasOilEntities {
// 		if dishHasOilEntity.DishID == dishEntity.ID {
// 			ids = append(ids, dishHasOilEntity.OilID)
// 		}
// 	}
// 	return ids
// }

func (m *MemberHasDishService) handleError(memberEntity *models.Member, err error) {
	errorModel := structs.ErrorModel{
		MemberID:     memberEntity.ID,
		ErrorMessage: err.Error(),
	}
	m.Errors = append(m.Errors, errorModel)

	// 把錯誤的用戶 id 加到結果參數中
	if _, ok := processResult["fail"]; !ok {
		var arr []string
		arr = append(arr, memberEntity.ID)
		processResult["fail"] = arr
	} else {
		processResult["fail"] = append(processResult["fail"], memberEntity.ID)
	}

	fmt.Println(errorModel)
}
