package ingredient

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
	"net/http"
	"sync"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/sirupsen/logrus"
	gormbulk "github.com/t-tiger/gorm-bulk-insert/v2"
)

var (
	memberHasIngredientManualDeclineEntities []models.MemberHasIngredient
	memberHasIngredientManualSuggestEntities []models.MemberHasIngredient
	ingredientEntities                       []models.Ingredient
	memberHealthDataMap                      map[string]models.MemberHealthData
	concurrentGoroutines                     chan struct{}
	ingredientHasAllergenEntities            []models.IngredientHasAllergen
	ingredientHasDeclineIngredientEntities   []models.IngredientHasDeclineIngredient
	ingredientRelievesSymptomEntities        []models.IngredientRelievesSymptom
	processResult                            map[string][]string
	now                                      time.Time
	location                                 *time.Location
	mutex                                        = &sync.Mutex{}
	retryCount                               int = 0
	retryAttempTimes                         int = 3
)

type MemberHasIngrediantService struct {
	sync.Mutex
	ingredientQueueParam structs.IngredientQueueParam
	Errors               []structs.ErrorModel
}

// 處理資料的主要進入點
func (m *MemberHasIngrediantService) Start(ingredientQueueParam structs.IngredientQueueParam) {

	if ingredientQueueParam.IsDie {
		panic(nil)
	}
	m.ingredientQueueParam = ingredientQueueParam

	// 初始化
	location, _ = time.LoadLocation("Asia/Taipei")
	now = time.Now().In(location)
	processResult = make(map[string][]string)
	memberHasIngredientManualDeclineEntities = nil
	memberHasIngredientManualSuggestEntities = nil
	ingredientEntities = nil
	ingredientHasAllergenEntities = nil
	ingredientHasDeclineIngredientEntities = nil
	ingredientRelievesSymptomEntities = nil
	memberHealthDataMap = make(map[string]models.MemberHealthData)

	// 針對特定用戶
	if ingredientQueueParam.Type == enums.ProcessSingle {

		fmt.Println("[ingredient] 處理方式： ", ingredientQueueParam.Type, "task_id", ingredientQueueParam.TaskID)

		if memberEntities, err := m.getMemberEntity(ingredientQueueParam.MemberId); err != nil {
			m.insertActivityLog(&memberEntities[0], false)
			return
		} else {
			if len(memberEntities) > 0 {
				fmt.Println("[ingredient] 開始處理用戶： ", memberEntities[0].ID)
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
					} else {
						fmt.Println(insertActivityLogError)
					}
				}
			} else {
				fmt.Println("[ingredient] 查無此用戶：", ingredientQueueParam.MemberId)
				m.JobDoneNotify()
			}
		}
		fmt.Println("[ingredient] Done!!")
	}

	// 針對所有用戶
	if ingredientQueueParam.Type == enums.ProcessAll {

		if memberEntities, err := m.getMemberEntity(""); err != nil {
			m.insertActivityLog(nil, false)
			return
		} else {
			fmt.Println("[ingredient] 處理方式： ", ingredientQueueParam.Type, len(memberEntities), "task_id", ingredientQueueParam.TaskID)

			var wg sync.WaitGroup
			wg.Add(len(memberEntities))

			// 限制啟動 goroutine 的數量
			concurrentGoroutines = make(chan struct{}, utils.EnvConfig.ConcurrentAmount)

			// 針對某個特定用戶
			for _, memberEntity := range memberEntities {
				concurrentGoroutines <- struct{}{}
				// 針對某個用戶處理各自的 ingredient 的 transaction 資料
				go m.process(memberEntity, &wg)
			}
			wg.Wait()
			close(concurrentGoroutines)

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
			fmt.Println("[ingredient] Done!!")
		}
	}

	// m.JobDoneNotify()
}

// 處理單個用戶的邏輯
func (m *MemberHasIngrediantService) process(memberEntity models.Member, wg *sync.WaitGroup) {

	var logService log.LogService
	logwg := logService.LoggerInit(memberEntity)
	logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("開始準備資料")

	if wg != nil {

		defer func() {
			wg.Done()
			logwg = nil
			fmt.Println("完成： ", memberEntity.Nickname)
			<-concurrentGoroutines
		}()
	}

	// 假設計算過程有任何問題，這邊處理重新計算
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("發生未預期錯誤：", err)
			retryCount++
			if retryCount <= retryAttempTimes {
				fmt.Println(memberEntity.Nickname, "重新計算")
				m.process(memberEntity, nil)
			}
			logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "error_message": err, "attemp_times": retryCount}).Error("發生錯誤，重新計算")
		}
	}()

	// 先找出 decline 的食材
	memberDeclines, err := m.memberHasDeclineData(memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢 decline 食材時發生錯誤", err.Error())
		return
	}

	// 先找出 suggest 的食材
	memberSuggests, err := m.memberHasSuggestData(memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢 suggest 食材時發生錯誤", err.Error())
		return
	}

	// 這個會員的過敏原
	memberAllergen, err := m.memberHasAllergenData(memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢會員的過敏原時發生錯誤", err.Error())
		return
	}

	// 這個會員不吃的東西
	memberDiet, err := m.memberHasDietData(memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢會員不吃的東西時發生錯誤", err.Error())
		return
	}

	// 這個會員的症狀
	memberPhysiology, err := m.memberHasPhysiologyData(memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢會員的症狀時發生錯誤", err.Error())
		return
	}

	// 這個會員營養師推薦的食材分類
	memberIngredients, err := m.memberHasIngredientsData(memberEntity)
	if err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("查詢會員營養師推薦的食材時發生錯誤", err.Error())
		return
	}

	// 準備要新增的資料
	var memberHasIngredientInsertEntities []models.MemberHasIngredient

	// 取得食材
	if err := m.getIngredientEntity(memberEntity); err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("取得食材總表時發生錯誤", err.Error())
		return
	}

	// 進行資料檢查
	for _, ingredientEntity := range ingredientEntities {
		// 預設要做
		do := true

		// 如果食材在 decline 的清單中，就不用在檢查了
		for _, decline := range memberDeclines {
			if decline.IngredientID == ingredientEntity.ID {
				do = false
				memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, decline)
				// 判斷到就可以結束了
				break
			}
		}

		if !do {
			continue
		}

		// 判斷是否含有會員的過敏原
		ingredientHasAllergenIds, err := m.ingredientHasAllergenIdList(memberEntity, ingredientEntity)
		if err != nil {
			logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("取得會員的過敏原時發生錯誤", err.Error())
			return
		}
		for _, allergenId := range memberAllergen.Data {
			for _, ingredientHasAllergenId := range ingredientHasAllergenIds {
				if ingredientHasAllergenId == allergenId {
					do = false
					entity := m.getMemberHasIngredientEntity(memberEntity, ingredientEntity, enums.DeclineType)
					memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, entity)
					// 判斷到就可以結束了
					break
				}
			}
			if !do {
				break
			}
		}

		if !do {
			continue
		}

		// 葷素排除
		if memberDiet.Type == "素食" && ingredientEntity.VegInedible == 1 {
			do = false
			entity := m.getMemberHasIngredientEntity(memberEntity, ingredientEntity, enums.DeclineType)
			memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, entity)
		}

		// 直接進行下一次的檢查
		if !do {
			continue
		}

		// 判斷是否含有不吃的食材
		ingredientHasDeclineIngredientIds, err := m.ingredientHasDeclineIngredientIdList(memberEntity, ingredientEntity)
		if err != nil {
			logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("取得含有不吃的食材時發生錯誤", err.Error())
			return
		}
		for _, memberDietId := range memberDiet.Data {
			for _, ingredientHasDeclineIngredientId := range ingredientHasDeclineIngredientIds {
				if ingredientHasDeclineIngredientId == memberDietId {
					do = false
					entity := m.getMemberHasIngredientEntity(memberEntity, ingredientEntity, enums.DeclineType)
					memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, entity)

					// 判斷到就可以結束了
					break
				}
			}
			if !do {
				break
			}
		}

		// 直接進行下一次的檢查
		if !do {
			continue
		}

		// 判斷是否有適合會員的食材
		ingredientRelievesSymptomIds, err := m.ingredientRelievesSymptomIdList(memberEntity, ingredientEntity)
		if err != nil {
			logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("取得是否有適合會員的食材時發生錯誤", err.Error())
			return
		}
		for _, memberPhysiologyId := range memberPhysiology.Data {
			for _, ingredientRelievesSymptomId := range ingredientRelievesSymptomIds {
				if ingredientRelievesSymptomId == memberPhysiologyId {
					do = false
					entity := m.getMemberHasIngredientEntity(memberEntity, ingredientEntity, enums.SuggestType)
					memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, entity)

					// 判斷到就可以結束了
					break
				}
			}
			if !do {
				break
			}
		}

		// 直接進行下一次的檢查
		if !do {
			continue
		}

		// 判斷可以推薦給會員的食材
		for _, memberIngredientID := range memberIngredients.Data {
			if int64(ingredientEntity.CategoryForSuggestion) == memberIngredientID {
				do = false
				entity := m.getMemberHasIngredientEntity(memberEntity, ingredientEntity, enums.SuggestType)
				memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, entity)

				// 判斷到就可以結束了
				break
			}
		}

		// 直接進行下一次的檢查
		if !do {
			continue
		}

		// 如果食材在 suggest 的清單中，就不用在檢查了
		for _, suggest := range memberSuggests {
			if suggest.IngredientID == ingredientEntity.ID {
				do = false
				memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, suggest)

				// 判斷到就可以結束了
				break
			}
		}

		// 直接進行下一次的檢查
		if !do {
			continue
		}

		// 都沒有檢查到，所以就建一筆 neutral
		entity := m.getMemberHasIngredientEntity(memberEntity, ingredientEntity, enums.NeutralType)
		memberHasIngredientInsertEntities = append(memberHasIngredientInsertEntities, entity)
	}

	// fmt.Println(memberEntity.Nickname, "total: ", len(memberHasIngredientInsertEntities), len(ingredientEntities))
	m.Lock()
	defer m.Unlock()

	logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("交易開始")
	tx := database.Mysql.Begin()
	defer func() {
		if r := recover(); r != nil {
			logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗: panic")
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		m.handleError(&memberEntity, err)
		return
	}

	// 標示版號
	var version = time.Now().Unix()
	var insertRecords []interface{}
	for _, memberHasIngredientInsertEntity := range memberHasIngredientInsertEntities {
		memberHasIngredientInsertEntity.Version = version
		insertRecords = append(insertRecords, memberHasIngredientInsertEntity)
	}

	// 批次 insert，一次 3000 筆
	if err := gormbulk.BulkInsert(tx, insertRecords, 3000); err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		tx.Rollback()
		m.handleError(&memberEntity, err)
		return
	}

	// 刪除 finished 的資料
	if err := tx.Where(models.MemberHasIngredient{Status: enums.FinishedStatus, MemberID: memberEntity.BackendID}).Delete(models.MemberHasIngredient{}).Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		tx.Rollback()
		m.handleError(&memberEntity, err)
		return
	}

	// logwg.WithFields(logrus.Fields{"task": "ingredient", "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("將資料更新成 finished")
	// 將資料更新成 finished
	if err := tx.Model(&models.MemberHasIngredient{}).Where(models.MemberHasIngredient{Status: enums.QueueStatus, MemberID: memberEntity.BackendID}).Update(models.MemberHasIngredient{Status: enums.FinishedStatus}).Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		tx.Rollback()
		m.handleError(&memberEntity, err)
		return
	}

	// logwg.WithFields(logrus.Fields{"task": "ingredient", "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("將資料更新成 finished")
	if err := tx.Commit().Error; err != nil {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Error("交易失敗", err.Error())
		m.handleError(&memberEntity, err)
		return
	}

	// 如果數量不一致的話，發一個 ELK LOG
	if len(ingredientEntities) != len(insertRecords) {
		logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "db_total_ingredient": len(ingredientEntities), "created_ingredient_total": len(insertRecords)}).Error("計算出來的筆數, 與資料庫筆數不一致")
	}

	logwg.WithFields(logrus.Fields{"task": "ingredient", "task_id": m.ingredientQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "db_total_ingredient": len(ingredientEntities), "created_ingredient_total": len(insertRecords)}).Info("交易完成")

	// 把處理好的用戶 id 加到結果參數中
	if _, ok := processResult["ok"]; !ok {
		var arr []string
		arr = append(arr, memberEntity.ID)
		processResult["ok"] = arr
	} else {
		processResult["ok"] = append(processResult["ok"], memberEntity.ID)
	}
}

func (m *MemberHasIngrediantService) getIngredientEntity(memberEntity models.Member) error {
	mutex.Lock()
	if len(ingredientEntities) == 0 {
		if errors := database.Mysql.Find(&ingredientEntities).GetErrors(); len(errors) != 0 {
			m.handleError(&memberEntity, errors[0])
			return errors[0]
		}
	}
	mutex.Unlock()
	return nil
}

// 取得會員
func (m *MemberHasIngrediantService) getMemberEntity(memberID string) ([]models.Member, error) {
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

// log 紀錄
func (m *MemberHasIngrediantService) insertActivityLog(memberEntity *models.Member, result bool) error {

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

	activityLogJSONModel.Type = m.ingredientQueueParam.Type
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
	activityLogEntity.LogName = "schedule.go.ingredient"
	activityLogEntity.Description = "會員食材計算"
	activityLogEntity.Properties = string(activityLogJSON)

	m.ingredientQueueParam.Result = string(activityLogJSON)

	if err := database.Mysql.Create(&activityLogEntity).Error; err != nil {
		return err
	}

	return nil
}

func (m *MemberHasIngrediantService) JobDoneNotify() {

	endpoint := utils.EnvConfig.Server.AppAPI + "/api/v1/workerCallback/ingredient"
	fmt.Println("callback url", endpoint, "task_id", m.ingredientQueueParam.TaskID)
	_, err := services.HttpRequest(http.MethodPost, endpoint, nil, m.ingredientQueueParam)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// 組合 member_has_ingredient 的 entity 資料
func (m *MemberHasIngrediantService) getMemberHasIngredientEntity(memberEntity models.Member, ingredientEntity models.Ingredient, typepo string) models.MemberHasIngredient {

	var memberHasIngredientEntity models.MemberHasIngredient
	memberHasIngredientEntity.IngredientID = ingredientEntity.ID
	memberHasIngredientEntity.MemberID = memberEntity.BackendID
	memberHasIngredientEntity.OperationType = enums.SystemOperate
	memberHasIngredientEntity.Status = enums.QueueStatus
	memberHasIngredientEntity.Type = typepo
	return memberHasIngredientEntity
}

// 確認用戶是否已經有被手動改成 decline 的資料
func (m *MemberHasIngrediantService) memberHasDeclineData(memberEntity models.Member) ([]models.MemberHasIngredient, error) {

	// 撈過就不撈了
	if len(memberHasIngredientManualDeclineEntities) == 0 {
		if errors := database.Mysql.Where(models.MemberHasIngredient{OperationType: enums.ManualOperate, Type: enums.DeclineType, MemberID: memberEntity.BackendID}).Find(&memberHasIngredientManualDeclineEntities).GetErrors(); len(errors) != 0 {
			m.handleError(&memberEntity, errors[0])
			return nil, errors[0]
		}
	}

	var declines []models.MemberHasIngredient
	for _, memberHasIngredientManualEntity := range memberHasIngredientManualDeclineEntities {
		if memberHasIngredientManualEntity.MemberID == memberEntity.BackendID {
			declines = append(declines, memberHasIngredientManualEntity)
		}
	}

	return declines, nil
}

// 確認用戶是否已經有被手動改成 suggest 的資料
func (m *MemberHasIngrediantService) memberHasSuggestData(memberEntity models.Member) ([]models.MemberHasIngredient, error) {

	// 撈過就不撈了
	if len(memberHasIngredientManualSuggestEntities) == 0 {
		if errors := database.Mysql.Where(models.MemberHasIngredient{OperationType: enums.ManualOperate, Type: enums.SuggestType, MemberID: memberEntity.BackendID}).Find(&memberHasIngredientManualSuggestEntities).GetErrors(); len(errors) != 0 {
			m.handleError(&memberEntity, errors[0])
			return nil, errors[0]
		}
	}

	var suggests []models.MemberHasIngredient
	for _, memberHasIngredientManualSuggestEntity := range memberHasIngredientManualSuggestEntities {
		if memberHasIngredientManualSuggestEntity.MemberID == memberEntity.BackendID {
			suggests = append(suggests, memberHasIngredientManualSuggestEntity)
		}
	}

	return suggests, nil
}

// 找出會員的過敏原
func (m *MemberHasIngrediantService) memberHasAllergenData(memberEntity models.Member) (structs.JsonModel, error) {

	// 看看 map 裡面有沒有暫存了
	var memberHealthDataEntity models.MemberHealthData
	var jsonModel structs.JsonModel
	if val, ok := memberHealthDataMap[memberEntity.ID]; !ok {

		// 沒有的話就撈
		if errors := database.Mysql.Where(models.MemberHealthData{MemberID: memberEntity.ID}).First(&memberHealthDataEntity).GetErrors(); len(errors) != 0 {
			if gorm.IsRecordNotFoundError(errors[0]) {
				return jsonModel, nil
			} else {
				m.handleError(&memberEntity, errors[0])
				return jsonModel, errors[0]
			}
		}
		memberHealthDataMap[memberEntity.ID] = memberHealthDataEntity
	} else {
		memberHealthDataEntity = val
	}

	// fmt.Println(memberHealthDataEntity.Allergens)
	if memberHealthDataEntity.Allergens != "" {
		if err := json.Unmarshal([]byte(memberHealthDataEntity.Allergens), &jsonModel); err != nil {
			m.handleError(&memberEntity, err)
			return jsonModel, err
		}
	}

	return jsonModel, nil
}

// 找出食材含有的過敏原
func (m *MemberHasIngrediantService) ingredientHasAllergenIdList(memberEntity models.Member, ingredientEntity models.Ingredient) ([]int64, error) {

	if len(ingredientHasAllergenEntities) == 0 {
		if errors := database.Mysql.Find(&ingredientHasAllergenEntities).GetErrors(); len(errors) != 0 {
			m.handleError(&memberEntity, errors[0])
			return nil, errors[0]
		}
	}

	var allergenIds []int64
	for _, ingredientHasAllergenEntity := range ingredientHasAllergenEntities {
		if ingredientHasAllergenEntity.IngredientID == ingredientEntity.ID {
			allergenIds = append(allergenIds, ingredientHasAllergenEntity.AllergenID)
		}
	}
	return allergenIds, nil
}

// 找出會員不吃的食材
func (m *MemberHasIngrediantService) memberHasDietData(memberEntity models.Member) (structs.JsonModel, error) {

	// 看看 map 裡面有沒有暫存了
	var memberHealthDataEntity models.MemberHealthData
	var jsonModel structs.JsonModel
	if val, ok := memberHealthDataMap[memberEntity.ID]; !ok {

		// 沒有的話就撈
		if errors := database.Mysql.Where(models.MemberHealthData{MemberID: memberEntity.ID}).First(&memberHealthDataEntity).GetErrors(); len(errors) != 0 {
			if gorm.IsRecordNotFoundError(errors[0]) {
				return jsonModel, nil
			} else {
				m.handleError(&memberEntity, errors[0])
				return jsonModel, errors[0]
			}
		}
		memberHealthDataMap[memberEntity.ID] = memberHealthDataEntity
	} else {
		memberHealthDataEntity = val
	}

	if memberHealthDataEntity.Diet != "" {
		if err := json.Unmarshal([]byte(memberHealthDataEntity.Diet), &jsonModel); err != nil {
			m.handleError(&memberEntity, err)
			return jsonModel, err
		}
	}
	return jsonModel, nil
}

// 食物含有不吃的成份
func (m *MemberHasIngrediantService) ingredientHasDeclineIngredientIdList(memberEntity models.Member, ingredientEntity models.Ingredient) ([]int64, error) {

	if len(ingredientHasDeclineIngredientEntities) == 0 {
		if errors := database.Mysql.Find(&ingredientHasDeclineIngredientEntities).GetErrors(); len(errors) != 0 {
			m.handleError(&memberEntity, errors[0])
			return nil, errors[0]
		}
	}

	var ids []int64
	for _, ingredientHasDeclineIngredientEntity := range ingredientHasDeclineIngredientEntities {
		if ingredientHasDeclineIngredientEntity.IngredientID == ingredientEntity.ID {
			ids = append(ids, ingredientHasDeclineIngredientEntity.DeclineIngredientID)
		}
	}
	return ids, nil
}

// 找出會員的自覺症狀
func (m *MemberHasIngrediantService) memberHasPhysiologyData(memberEntity models.Member) (structs.JsonModel, error) {

	// 看看 map 裡面有沒有暫存了
	var memberHealthDataEntity models.MemberHealthData
	var jsonModel structs.JsonModel
	if val, ok := memberHealthDataMap[memberEntity.ID]; !ok {

		// 沒有的話就撈
		if errors := database.Mysql.Where(models.MemberHealthData{MemberID: memberEntity.ID}).First(&memberHealthDataEntity).GetErrors(); len(errors) != 0 {
			if gorm.IsRecordNotFoundError(errors[0]) {
				return jsonModel, nil
			} else {
				m.handleError(&memberEntity, errors[0])
				return jsonModel, errors[0]
			}
		}
		memberHealthDataMap[memberEntity.ID] = memberHealthDataEntity
	} else {
		memberHealthDataEntity = val
	}

	if memberHealthDataEntity.Physiology != "" {
		if err := json.Unmarshal([]byte(memberHealthDataEntity.Physiology), &jsonModel); err != nil {
			m.handleError(&memberEntity, err)
			return jsonModel, err
		}
	}

	return jsonModel, nil
}

// 找出會員營養師推薦的食材分類
func (m *MemberHasIngrediantService) memberHasIngredientsData(memberEntity models.Member) (structs.JsonModel, error) {

	// 看看 map 裡面有沒有暫存了
	var memberHealthDataEntity models.MemberHealthData
	var jsonModel structs.JsonModel
	if val, ok := memberHealthDataMap[memberEntity.ID]; !ok {

		// 沒有的話就撈
		if errors := database.Mysql.Where(models.MemberHealthData{MemberID: memberEntity.ID}).First(&memberHealthDataEntity).GetErrors(); len(errors) != 0 {
			if gorm.IsRecordNotFoundError(errors[0]) {
				return jsonModel, nil
			} else {
				m.handleError(&memberEntity, errors[0])
				return jsonModel, errors[0]
			}
		}
		memberHealthDataMap[memberEntity.ID] = memberHealthDataEntity
	} else {
		memberHealthDataEntity = val
	}

	if memberHealthDataEntity.Ingredients != "" {
		if err := json.Unmarshal([]byte(memberHealthDataEntity.Ingredients), &jsonModel); err != nil {
			m.handleError(&memberEntity, err)
			return jsonModel, err
		}
	}

	return jsonModel, nil
}

// 食材可能會有的症狀
func (m *MemberHasIngrediantService) ingredientRelievesSymptomIdList(memberEntity models.Member, ingredientEntity models.Ingredient) ([]int64, error) {

	if len(ingredientRelievesSymptomEntities) == 0 {
		if errors := database.Mysql.Find(&ingredientRelievesSymptomEntities).GetErrors(); len(errors) != 0 {
			m.handleError(&memberEntity, errors[0])
			return nil, errors[0]
		}
	}

	var ids []int64
	for _, ingredientRelievesSymptomEntity := range ingredientRelievesSymptomEntities {
		if ingredientRelievesSymptomEntity.IngredientID == ingredientEntity.ID {
			ids = append(ids, ingredientRelievesSymptomEntity.SymptomID)
		}
	}
	return ids, nil
}

// Todo:需要增加錯誤的處理機制
func (m *MemberHasIngrediantService) handleError(memberEntity *models.Member, err error) {
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
