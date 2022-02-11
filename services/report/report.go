package report

import (
	"dishrank-go-worker/database"
	"dishrank-go-worker/enums"
	"dishrank-go-worker/models"
	"dishrank-go-worker/services"
	"dishrank-go-worker/services/log"
	"dishrank-go-worker/services/trackLog"
	"dishrank-go-worker/structs"
	"dishrank-go-worker/utils"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/sirupsen/logrus"
)

type ReportService struct {
	sync.Mutex
	reportQueueParam structs.ReportQueueParam
	Errors           []structs.ErrorModel
}

var (
	userEntities []models.User
	// dailyMainRecordEntities []models.DailyMainRecord
	concurrentGoroutines chan struct{}
	now                  time.Time
	processResult        map[string][]string
	location             *time.Location
	retryCount           int = 0
	retryAttempTimes     int = 3
)

// 起始 func
func (r *ReportService) Start(reportQueueParam structs.ReportQueueParam) {

	if reportQueueParam.IsDie {
		panic(nil)
	}

	r.reportQueueParam = reportQueueParam

	// 初始化
	location, _ = time.LoadLocation("Asia/Taipei")
	now = time.Now().In(location)
	processResult = make(map[string][]string)
	userEntities = nil
	retryCount = 0

	// 針對特定用戶
	if reportQueueParam.Type == enums.ProcessSingle {

		fmt.Println("[report] 處理方式： ", reportQueueParam.Type)

		if memberEntities, err := r.getMemberEntity(reportQueueParam.MemberId, reportQueueParam.StartDate); err != nil {
			r.insertActivityLog(&memberEntities[0], false)
			return
		} else {
			if len(memberEntities) > 0 {
				fmt.Println("[report] 開始處理用戶： ", memberEntities[0].ID, "task_id", reportQueueParam.TaskID)
				r.process(memberEntities[0], reportQueueParam, nil)
				// 查看是否有 Error log
				if len(r.Errors) == 0 {
					if insertActivityLogError := r.insertActivityLog(&memberEntities[0], true); insertActivityLogError == nil {
						r.JobDoneNotify()
					} else {
						fmt.Println(insertActivityLogError)
					}
				} else {
					if insertActivityLogError := r.insertActivityLog(&memberEntities[0], false); insertActivityLogError == nil {
						r.JobDoneNotify()
					} else {
						fmt.Println(insertActivityLogError)
					}
				}
			} else {
				fmt.Println("[report] 查無此用戶：", reportQueueParam.MemberId)
				r.JobDoneNotify()
			}
		}
		fmt.Println("[report] Done!!")
	}

	// 針對所有用戶
	if reportQueueParam.Type == enums.ProcessAll {

		if memberEntities, err := r.getMemberEntity("", reportQueueParam.StartDate); err != nil {
			r.insertActivityLog(nil, false)
			return
		} else {

			var logEntity models.Member
			logEntity.Nickname = "主程式"
			var logService log.LogService
			logwg := logService.LoggerInit(logEntity)
			logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": logEntity.Nickname, "total_member": len(memberEntities)}).Info("用戶資料準備完成")

			fmt.Println("[report] 處理方式： ", reportQueueParam.Type, len(memberEntities), "task_id", reportQueueParam.TaskID)

			var wg sync.WaitGroup
			wg.Add(len(memberEntities))

			// 限制啟動 goroutine 的數量
			concurrentGoroutines = make(chan struct{}, utils.EnvConfig.ConcurrentAmount)

			// 針對某個特定用戶
			for _, memberEntity := range memberEntities {
				concurrentGoroutines <- struct{}{}
				// 針對某個用戶處理各自的 ingredient 的 transaction 資料
				go r.process(memberEntity, reportQueueParam, &wg)
			}
			wg.Wait()
			close(concurrentGoroutines)

			if len(r.Errors) == 0 {
				// 完成後，紀錄 log
				if insertActivityLogError := r.insertActivityLog(nil, true); insertActivityLogError == nil {
					r.JobDoneNotify()
				}
			} else {
				if insertActivityLogError := r.insertActivityLog(nil, false); insertActivityLogError == nil {
					r.JobDoneNotify()
				}
			}
			logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": logEntity.Nickname, "total_member": len(memberEntities)}).Info("全部已完成")
			fmt.Println("[report] Done!!")
		}
	}

	// r.JobDoneNotify()
}

// 取得會員
// todo: v2.1
// todo: quotation_has_seats.member_id   = member.id 改用這個關聯
// todo: quotation_has_seats.expired_at  > '2021-05-08' 改用這個where條件
func (m *ReportService) getMemberEntity(memberID, startDate string) ([]models.Member, error) {

	var memberEntities []models.Member
	//time_layout := "2006-01-02"
	//t, err := time.Parse(time_layout, startDate)
	//if err != nil {
	//	return nil, err
	//}
	//dateline := t.Add(time.Hour * (-7 * 24))
	//dateline := time.Now().Add(time.Hour * (24 * 7))
	memberStatus := "deleted"
	dateline := time.Now()
	if memberID == "" {
		if sqlErrors := database.Mysql.Debug().
			Joins("join quotation_has_seats on members.id = quotation_has_seats.member_id").
			Joins("join quotations on quotation_has_seats.quotation_id = quotations.id").
			Where("members.expired_at > ? and members.status != ?", dateline, memberStatus).Find(&memberEntities).GetErrors(); len(sqlErrors) != 0 {
			m.handleError(nil, sqlErrors[0])
			return nil, sqlErrors[0]
		}
	} else {
		if sqlErrors := database.Mysql.Debug().
			Joins("join quotation_has_seats on members.id = quotation_has_seats.member_id").
			Joins("join quotations on quotation_has_seats.quotation_id = quotations.id").
			Where("members.expired_at > ?  and members.id = ? and members.status != ?", dateline, memberID, memberStatus).Find(&memberEntities).GetErrors(); len(sqlErrors) != 0 {
			var memberEntity models.Member
			memberEntity.ID = memberID
			memberEntity.Nickname = "查詢用戶資料表(members)有誤"
			m.handleError(&memberEntity, sqlErrors[0])
			return nil, sqlErrors[0]
		}
	}

	return memberEntities, nil
}

// 處理主邏輯流程
func (r *ReportService) process(memberEntity models.Member, reportQueueParam structs.ReportQueueParam, wg *sync.WaitGroup) {
	fmt.Println("開始處理用戶： ", memberEntity.Nickname)

	// waitgroup 的完成工作的 defer 處理
	if wg != nil {
		defer func() {
			wg.Done()
			fmt.Println("完成： ", memberEntity.Nickname)
			<-concurrentGoroutines
		}()
	}

	var reportModel structs.Report

	var logService log.LogService
	logwg := logService.LoggerInit(memberEntity)
	logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("開始準備資料")

	// 假設計算過程有任何問題，這邊處理重新計算
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("發生未預期錯誤：", err)
			retryCount++
			if retryCount <= retryAttempTimes {
				fmt.Println(memberEntity.Nickname, "重新計算")
				r.process(memberEntity, reportQueueParam, nil)
			}
			logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID, "error_message": err, "attemp_times": retryCount}).Error("發生錯誤，重新計算")
		}
	}()

	//reportModel.Mail = memberEntity.Email
	//reportModel.ActivationCode = memberEntity.ActivationCode

	// 計算卡路里跟水份
	logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("計算卡路里跟水份")
	var err error
	var calorieModel structs.Calorie
	var WaterModel structs.Water
	if calorieModel, WaterModel, err = r.getCalorieAndWaterModel(memberEntity); err != nil {
		logwg.WithFields(logrus.Fields{"name": memberEntity.Nickname, "task_id": r.reportQueueParam.TaskID, "member_id": memberEntity.BackendID, "error_message": err.Error()}).Error("計算卡路里跟水份時，錯誤")
		r.handleError(&memberEntity, err)
		return
	}
	jsonCalorie := structs.JsonCalorie{
		Average: services.Round(calorieModel.Average),
		Base:    calorieModel.Base,
		Icon:    calorieModel.Icon,
	}

	jsonWater := structs.JsonWater{
		Average: services.Round(WaterModel.Average),
		Base:    WaterModel.Base,
		Icon:    WaterModel.Icon,
	}

	reportModel.Calorie = jsonCalorie
	reportModel.Water = jsonWater
	achieveSum := 0.0
	var achieveNum float64
	reportModel.AchieveDetail.Calorie, achieveNum = calculateDiffValue(calorieModel.Average, float64(calorieModel.Base))
	achieveSum += achieveNum
	reportModel.AchieveDetail.Water, achieveNum = calculateDiffValue(WaterModel.Average, float64(WaterModel.Base))
	achieveSum += achieveNum

	// 取出 comment 跟 createdBy
	logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("取得用戶筆記")
	if memberNoteEntity, err := r.getMemberNote(memberEntity); err != nil {
		return
	} else {
		if memberNoteEntity != nil {
			reportModel.Comment = memberNoteEntity.ReportComment
			if userEntity, err := r.getUser(memberNoteEntity.ReportedBy, memberEntity); err != nil {
				return
			} else {
				if userEntity != nil {
					reportModel.CreatedBy = userEntity.Name
				}
			}
		}
	}

	// 計算七天的模型
	logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("計算七天模型")
	sevenDaysModel, err := r.getSevenDaysModel(memberEntity, reportQueueParam)
	if err != nil {
		r.handleError(&memberEntity, err)
		return
	}
	reportModel.Days = sevenDaysModel

	// 計算4種營養素
	logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("計算營養成份")
	if nutrientModel, err := r.getNutrients(memberEntity); err != nil {
		logwg.WithFields(logrus.Fields{"name": memberEntity.Nickname, "task_id": r.reportQueueParam.TaskID, "member_id": memberEntity.BackendID, "error_message": err.Error()}).Error("計算營養成份時，錯誤")
		r.handleError(&memberEntity, err)
		return
	} else {
		reportModel.Nutrients = nutrientsToJson(nutrientModel)

		reportModel.AchieveDetail.Protein, achieveNum = calculateDiffValue(nutrientModel.Protein.Total, nutrientModel.Protein.Suggest)
		achieveSum += achieveNum

		reportModel.AchieveDetail.Fat, achieveNum = calculateDiffValue(nutrientModel.Fat.Total, nutrientModel.Fat.Suggest)
		achieveSum += achieveNum

		reportModel.AchieveDetail.Fibre, achieveNum = calculateDiffValue(nutrientModel.Fibre.Total, nutrientModel.Fibre.Suggest)
		achieveSum += achieveNum

		reportModel.AchieveDetail.Carbohydrates, achieveNum = calculateDiffValue(nutrientModel.Carbohydrates.Total, nutrientModel.Carbohydrates.Suggest)
		achieveSum += achieveNum
	}
	// 獲得達成度
	if checkCanSum(reportModel.AchieveDetail) {
		reportModel.AchieveDetail.Sum = strconv.Itoa(services.Round(achieveSum / 6))
		reportModel.Achievement = getAchievement(reportModel.AchieveDetail.Sum, memberEntity.ID)
	} else {
		reportModel.AchieveDetail.Sum = "none"
		reportModel.Achievement = "none"
	}

	// 獲得上禮拜資料
	oldMemberReportEntity, oldErr := getOldMemberReport(memberEntity, reportQueueParam)
	if oldErr != nil && oldErr.Error() != "record not found" {
		logwg.WithFields(logrus.Fields{"name": memberEntity.Nickname, "task_id": r.reportQueueParam.TaskID, "member_id": memberEntity.BackendID, "error_message": oldErr.Error()}).Error("撈取舊member_report錯誤")
		r.handleError(&memberEntity, err)
		return
	}
	// 若有舊資料，檢查這禮拜與上禮拜的達成度，是不是有進步
	if oldMemberReportEntity != nil {
		reportModel.Achievement = checkAchievement(oldMemberReportEntity, memberEntity, reportModel)
	}

	// 插入member_report,若有則update
	logwg.WithFields(logrus.Fields{"task": "report", "task_id": r.reportQueueParam.TaskID, "name": memberEntity.Nickname, "member_id": memberEntity.BackendID}).Info("新增報表紀錄")
	if err = r.InsertMemberReport(memberEntity, reportQueueParam, reportModel); err != nil {
		logwg.WithFields(logrus.Fields{"name": memberEntity.Nickname, "task_id": r.reportQueueParam.TaskID, "member_id": memberEntity.BackendID, "error_message": err.Error()}).Error("新增報表紀錄的時後，錯誤")
		r.handleError(&memberEntity, err)
		return
	}
	// 檢查周報comment欄位
	logwg.WithFields(logrus.Fields{"name": memberEntity.Nickname, "task_id": r.reportQueueParam.TaskID, "member_id": memberEntity.BackendID}).Info("檢查週報 comment 欄位")
	if checkReportChangeErr := r.checkReportChange(memberEntity, reportQueueParam, reportModel, oldMemberReportEntity); err != nil {
		logwg.WithFields(logrus.Fields{"name": memberEntity.Nickname, "task_id": r.reportQueueParam.TaskID, "member_id": memberEntity.BackendID, "error_message": checkReportChangeErr.Error()}).Info("檢查週報 comment 欄位時有問題")
	}

	// 把處理好的用戶 id 加到結果參數中
	if _, ok := processResult["ok"]; !ok {
		var arr []string
		arr = append(arr, memberEntity.ID)
		processResult["ok"] = arr
	} else {
		processResult["ok"] = append(processResult["ok"], memberEntity.ID)
	}
}

func getAchievement(sum string, memberId string) string {
	if sum == "" {
		return "none"
	}
	i, err := strconv.Atoi(sum)
	if err != nil {
		trackLog.Error("memberId: "+memberId+" get achievement error: "+err.Error(), true)
		return "none"
	}
	if i <= 20 {
		return enums.VeryGood
	} else if i > 20 && i <= 40 {
		return enums.Good
	} else if i > 40 && i <= 60 {
		return enums.NotBad
	} else if i > 60 {
		return enums.NotGood
	}
	trackLog.Error("memberId: "+memberId+" get achievement error: get wrong number: i = "+strconv.Itoa(i), true)
	return "none"
}

// 檢查是否可以相加, 只要其中一個不是none就可以加
func checkCanSum(detail structs.AchieveDetail) bool {
	if detail.Calorie != "none" {
		return true
	}

	if detail.Water != "none" {
		return true
	}

	if detail.Protein != "none" {
		return true
	}

	if detail.Fat != "none" {
		return true
	}

	if detail.Fibre != "none" {
		return true
	}

	if detail.Carbohydrates != "none" {
		return true
	}
	return false
}

// 計算營養素達成度
func calculateDiffValue(average, base float64) (string, float64) {
	value := 0
	result := ""

	if base != 0 {
		value = services.Round((average/base - 1) * 100)
		result = strconv.Itoa(value)
	} else {
		result = "none"
	}
	return result, math.Abs(float64(value))
}

func getOldMemberReport(memberEntity models.Member, reportQueueParam structs.ReportQueueParam) (*models.MemberReport, error) {
	var memberReportEntity *models.MemberReport
	week, _ := strconv.Atoi(strings.Replace(reportQueueParam.Week, "W", "", 1))
	lastWeek := week - 1

	// 如果是當年第一週的話，就要抓去年的最後一週
	if lastWeek == 0 {
		year, _ := strconv.Atoi(reportQueueParam.Year)
		lastYear := year - 1

		var tmpEntity models.MemberReport
		if err := database.Mysql.Where("id = ?",
			database.Mysql.Table(memberReportEntity.TableName()).
				Where(models.MemberReport{Year: strconv.Itoa(lastYear), MemberID: memberEntity.ID}).
				Select("id").
				Having("MAX(CAST(SUBSTRING(week, 2, length(week)-1) AS UNSIGNED))").SubQuery()).
			First(&tmpEntity).Error; err != nil {
			return nil, err
		}
		memberReportEntity = &tmpEntity
	} else {

		var tmpEntity models.MemberReport
		if err := database.Mysql.Where(models.MemberReport{Year: reportQueueParam.Year, Week: "W" + strconv.Itoa(lastWeek), MemberID: memberEntity.ID}).First(&tmpEntity).Error; err != nil {
			return nil, err
		}
		memberReportEntity = &tmpEntity
	}
	return memberReportEntity, nil
}

// 檢查achievement 是否有進步
func checkAchievement(oldMemberReportEntity *models.MemberReport, memberEntity models.Member, reportModel structs.Report) string {
	var resultAchievement = reportModel.Achievement
	var oldData structs.Report
	if oldMemberReportEntity == nil {
		return resultAchievement
	}
	if err := json.Unmarshal([]byte(oldMemberReportEntity.Data), &oldData); err != nil {
		trackLog.Error("member_id: "+memberEntity.ID+" unmarshal data error: "+err.Error()+", data is"+oldMemberReportEntity.Data, true)
		return resultAchievement
	}
	// 如果上禮拜的達成度是有進步，則確定上禮拜真正的achievement
	if oldData.Achievement == enums.MoreBetter {
		oldData.Achievement = getAchievement(oldData.AchieveDetail.Sum, memberEntity.ID)
	}

	if reportModel.Achievement == oldData.Achievement && reportModel.Achievement != "none" {
		nowSum, _ := strconv.Atoi(reportModel.AchieveDetail.Sum)
		oldSum, _ := strconv.Atoi(oldData.AchieveDetail.Sum)
		//fmt.Println(reportModel.Achievement, nowSum, oldData.Achievement, oldSum)
		if nowSum < oldSum {
			resultAchievement = enums.MoreBetter
		}
	}
	return resultAchievement
}

// 檢查跟前一週的 comment 是否有差異
func (r *ReportService) checkReportChange(memberEntity models.Member, reportQueueParam structs.ReportQueueParam, reportModel structs.Report, memberReportEntity *models.MemberReport) error {

	// 預設一樣
	compareResult := enums.WeeklyCommentSame

	//var memberReportEntity *models.MemberReport
	//week, _ := strconv.Atoi(strings.Replace(reportQueueParam.Week, "W", "", 1))
	//lastWeek := week - 1
	//
	// 如果是當年第一週的話，就要抓去年的最後一週
	//if lastWeek == 0 {
	//	year, _ := strconv.Atoi(reportQueueParam.Year)
	//	lastYear := year - 1
	//
	//	var tmpEntity models.MemberReport
	//	if err := database.Mysql.Where("id = ?",
	//		database.Mysql.Table(memberReportEntity.TableName()).
	//			Where(models.MemberReport{Year: strconv.Itoa(lastYear), MemberID: memberEntity.ID}).
	//			Select("id").
	//			Having("MAX(CAST(SUBSTRING(week, 2, length(week)-1) AS UNSIGNED))").SubQuery()).
	//		First(&tmpEntity).Error; err != nil {
	//		return err
	//	}
	//	memberReportEntity = &tmpEntity
	//} else {
	//
	//	var tmpEntity models.MemberReport
	//	if err := database.Mysql.Where(models.MemberReport{Year: reportQueueParam.Year, Week: "W" + strconv.Itoa(lastWeek), MemberID: memberEntity.ID}).First(&tmpEntity).Error; err != nil {
	//		return err
	//	}
	//	memberReportEntity = &tmpEntity
	//}

	// 如果沒有找到紀錄的話
	if memberReportEntity == nil {
		compareResult = enums.WeeklyCommentChanged
	} else {
		var oldData structs.Report
		if err := json.Unmarshal([]byte(memberReportEntity.Data), &oldData); err != nil {
			return err
		}

		// 如果兩週的 comment 都是空值的話
		if (reportModel.Comment == "") && (oldData.Comment == "") {
			compareResult = enums.WeeklyCommentNA
		}

		if reportModel.Comment != oldData.Comment {
			compareResult = enums.WeeklyCommentChanged
		}
	}

	fmt.Println("兩週比對結果：", compareResult)

	if err := r.updateWeeklyTaskProgress(&memberEntity, reportQueueParam, compareResult); err != nil {
		// fmt.Println("updateWeeklyTaskProgress", err.Error())
		return err
	}

	return nil
}

func nutrientsToJson(model structs.Nutrient) structs.JsonNutrient {
	var jsonFat, jsonProtein, jsonFibre, jsonCarbohydrates structs.JsonNutrientItem
	var jsonNutrient structs.JsonNutrient
	jsonFat.Suggest = services.Round(model.Fat.Suggest)
	jsonFat.Total = services.Round(model.Fat.Total)

	jsonProtein.Suggest = services.Round(model.Protein.Suggest)
	jsonProtein.Total = services.Round(model.Protein.Total)

	jsonFibre.Suggest = services.Round(model.Fibre.Suggest)
	jsonFibre.Total = services.Round(model.Fibre.Total)

	jsonCarbohydrates.Suggest = services.Round(model.Carbohydrates.Suggest)
	jsonCarbohydrates.Total = services.Round(model.Carbohydrates.Total)

	jsonNutrient.Fat = jsonFat
	jsonNutrient.Protein = jsonProtein
	jsonNutrient.Fibre = jsonFibre
	jsonNutrient.Carbohydrates = jsonCarbohydrates
	return jsonNutrient

}

// 更新 weekly_task_Progress 的紀錄
func (r *ReportService) updateWeeklyTaskProgress(memberEntity *models.Member, reportQueueParam structs.ReportQueueParam, compareResult string) error {
	var weeklyTaskProgressEntity models.WeeklyTaskProgress
	weeklyTaskProgressEntity.CommentChange = compareResult
	if result := database.Mysql.Table(weeklyTaskProgressEntity.TableName()).Where(models.WeeklyTaskProgress{Week: reportQueueParam.Week, Year: reportQueueParam.Year, MemberID: memberEntity.ID}).Update(weeklyTaskProgressEntity); result != nil {
		//fmt.Println("WeeklyTaskProgress 更新筆數：", result.RowsAffected)
		if result.RowsAffected == 0 {
			return errors.New("WeeklyTaskProgress 沒有可更新的紀錄")
		}
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// 紀錄 job 成功/失敗
func (r *ReportService) insertActivityLog(memberEntity *models.Member, result bool) error {

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

	activityLogJSONModel.Type = r.reportQueueParam.Type
	activityLogJSONModel.Result = result

	if activityLogJSONModel.Result {
		activityLogJSONModel.Message = "ok"
	} else {
		if activityLogJSONModel.Type == enums.ProcessSingle {
			activityLogJSONModel.Message = r.Errors[0].ErrorMessage
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
	activityLogEntity.LogName = "schedule.go.report"
	activityLogEntity.Description = "會員報表計算"
	activityLogEntity.Properties = string(activityLogJSON)

	r.reportQueueParam.Result = string(activityLogJSON)

	if err := database.Mysql.Create(&activityLogEntity).Error; err != nil {
		return err
	}

	return nil
}

func (r *ReportService) JobDoneNotify() {
	endpoint := utils.EnvConfig.Server.AppAPI + "/api/v1/workerCallback/report"
	fmt.Println("callback url", endpoint, "task_id", r.reportQueueParam.TaskID)
	_, err := services.HttpRequest(http.MethodPost, endpoint, nil, r.reportQueueParam)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// 新增報表紀錄
func (r *ReportService) InsertMemberReport(memberEntity models.Member, reportQueueParam structs.ReportQueueParam, reportModel structs.Report) error {

	var err error

	out, err := json.Marshal(reportModel)
	if err != nil {
		return err
	}

	start, _ := time.Parse("2006-01-02", reportQueueParam.StartDate)
	end, _ := time.Parse("2006-01-02", reportQueueParam.EndDate)

	var memberReportEntity models.MemberReport
	if err = database.Mysql.Where(models.MemberReport{Week: reportQueueParam.Week, Year: reportQueueParam.Year, MemberID: memberEntity.ID}).First(&memberReportEntity).Error; gorm.IsRecordNotFoundError(err) {

		// 如果找不到資料的話，就新增
		memberReportEntity = models.MemberReport{}
		memberReportEntity.MemberID = memberEntity.ID
		memberReportEntity.Week = reportQueueParam.Week
		memberReportEntity.Year = reportQueueParam.Year
		memberReportEntity.StartDate = start
		memberReportEntity.EndDate = end
		memberReportEntity.Data = string(out)
		if err = database.Mysql.Create(&memberReportEntity).Error; err != nil {
			return err
		}
	} else {
		if err != nil {
			return err
		} else {
			insertTime := time.Now().In(location)
			memberReportEntity = models.MemberReport{}
			memberReportEntity.StartDate = start
			memberReportEntity.EndDate = end
			memberReportEntity.Data = string(out)
			memberReportEntity.UpdatedAt = &insertTime
			if err = database.Mysql.Table(memberReportEntity.TableName()).Where(models.MemberReport{Week: reportQueueParam.Week, Year: reportQueueParam.Year, MemberID: memberEntity.ID}).Update(memberReportEntity).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// 取得七日的資料模型
func (r *ReportService) getSevenDaysModel(memberEntity models.Member, reportQueueParam structs.ReportQueueParam) ([]structs.Day, error) {

	// 抓用戶的每日營養資料, 不足七筆要補足
	dailyMainRecords, err := r.getMemberDailyMainRecords(memberEntity)
	if err != nil {
		return nil, err
	}
	var dateStringArray []structs.Day
	// weekStart := *dailyMainRecords[0].DiaryDate
	weekStart, _ := time.Parse("2006-01-02", reportQueueParam.StartDate)
	for i := 0; i < 7; i++ {

		// 有命中日期的話，就帶入原本的資料
		var dayModel *structs.Day
		for _, dailyMainRecord := range dailyMainRecords {
			if dailyMainRecord.DiaryDate.Format("2006-01-02") == weekStart.Format("2006-01-02") {
				dayModel = new(structs.Day)
				dayModel.Day = weekStart.Format("2006-01-02")
				dayModel.Calorie = dailyMainRecord.KcalTotal
				dayModel.Water = dailyMainRecord.MoistureTotal
				break
			}
		}

		// 假如上面的 loop 都沒命中，就帶 0
		if dayModel == nil {
			dayModel = new(structs.Day)
			dayModel.Day = weekStart.Format("2006-01-02")
			dayModel.Calorie = 0
			dayModel.Water = 0
		}

		// 存起來
		dateStringArray = append(dateStringArray, *dayModel)

		// 加一天
		weekStart = weekStart.Add(24 * time.Hour)
	}

	return dateStringArray, nil
}

// 取得營養成份的資料
func (r *ReportService) getNutrients(memberEntity models.Member) (structs.Nutrient, error) {

	// 回傳模型
	var nutrientModel structs.Nutrient

	//// 用戶的攝取紀錄的筆數
	//memberDailyMainRecordEntities, err := r.getMemberDailyMainRecords(memberEntity)
	//if err != nil {
	//	return nutrientModel, err
	//}
	//memberDailyMainRecordCount := len(memberDailyMainRecordEntities)
	var recordNumber int
	subQuery := database.Mysql.Model(&models.DailyMealDish{}).Group("daily_meal_dish.main_id").Select("daily_meal_dish.main_id").QueryExpr()
	database.Mysql.Model(&models.DailyMainRecord{}).
		Joins("inner join (?) as sub_query on daily_main_records.id = sub_query.main_id", subQuery).
		Where("daily_main_records.member_id = ? and daily_main_records.diary_date >= ? and daily_main_records.diary_date <= ?", memberEntity.ID, r.reportQueueParam.StartDate, r.reportQueueParam.EndDate).
		Count(&recordNumber)
	fmt.Println("record number: ", recordNumber)

	// 取得用戶的健康資訊
	var dietingPlanModel structs.DietingPlan
	var trueDietingPlanModel structs.TrueDietingPlan
	if memberHealthData, err := r.getMemberHealthDatas(memberEntity); err != nil {
		r.handleError(&memberEntity, err)
		return nutrientModel, err
	} else {
		if memberHealthData != nil {
			if err := json.Unmarshal([]byte(memberHealthData.DietingPlans), &trueDietingPlanModel); err != nil {
				// return nutrientModel, err

				// 如果無法 parse 就給 0
				trackLog.Error("member_id: "+memberEntity.ID+", json unmarshal true dietingPlanModel error in getNutrients(): "+err.Error(), true)
				dietingPlanModel.Calories = 0
				dietingPlanModel.Carbohydrate = 0
				dietingPlanModel.Dietaryfiber = 0
				dietingPlanModel.Fats = 0
				dietingPlanModel.Protein = 0
				dietingPlanModel.Water = 0
			} else {
				trueDietingPlanModelJson, _ := json.Marshal(trueDietingPlanModel)
				trackLog.Info(string(trueDietingPlanModelJson), true)
				dietingPlanModel.Calories = trueDietingPlanModel.Calories + trueDietingPlanModel.FixedKcal
				dietingPlanModel.Carbohydrate = trueDietingPlanModel.Carbohydrate + trueDietingPlanModel.TotalCarbohydrate
				dietingPlanModel.Dietaryfiber = trueDietingPlanModel.Dietaryfiber + trueDietingPlanModel.DietaryfiberDash
				dietingPlanModel.Fats = trueDietingPlanModel.Fats + trueDietingPlanModel.CrudeFat
				dietingPlanModel.Protein = trueDietingPlanModel.Protein + trueDietingPlanModel.CrudeProtein
				dietingPlanModel.Water = trueDietingPlanModel.Water
			}
		}
	}

	// 計算 protein total
	var dailyMealRecordEntities []models.DailyMealRecord
	if err := database.Mysql.Model(&models.DailyMealRecord{}).Joins("left join daily_main_records on daily_meal_records.main_id = daily_main_records.id").
		Where("daily_main_records.member_id = ? and daily_main_records.diary_date >= ? and daily_main_records.diary_date <= ?", memberEntity.ID, r.reportQueueParam.StartDate, r.reportQueueParam.EndDate).
		Scan(&dailyMealRecordEntities).Error; err != nil {
		return nutrientModel, err
	}
	var totalProtein float64 = 0
	//var dailyMealRecordCount = len(dailyMealRecordEntities)
	for _, dailyMealRecordEntity := range dailyMealRecordEntities {
		//fmt.Printf("totalProtein: %f, CrudeProtein: %f\n", totalProtein, dailyMealRecordEntity.CrudeProtein)
		totalProtein = totalProtein + dailyMealRecordEntity.CrudeProtein
	}
	fmt.Println("total protein, ", totalProtein)
	var nums = 1
	if recordNumber != 0 {
		nums = recordNumber
	}
	nutrientModel.Protein.Total = decimal(totalProtein / float64(nums))
	nutrientModel.Protein.Suggest = decimal(dietingPlanModel.Protein * 1)

	// 計算 fat total
	var totalFat float64 = 0
	for _, dailyMealRecordEntity := range dailyMealRecordEntities {
		totalFat = totalFat + dailyMealRecordEntity.CrudeFat
	}
	nutrientModel.Fat.Total = decimal(totalFat / float64(nums))
	nutrientModel.Fat.Suggest = decimal(dietingPlanModel.Fats * 1)


	// 計算 carbohydrates total
	var totalCarbohydrate float64 = 0
	for _, dailyMealRecordEntity := range dailyMealRecordEntities {
		totalCarbohydrate = totalCarbohydrate + dailyMealRecordEntity.TotalCarbohydrate
	}
	nutrientModel.Carbohydrates.Total = decimal(totalCarbohydrate / float64(nums))
	nutrientModel.Carbohydrates.Suggest = decimal(dietingPlanModel.Carbohydrate * 1)

	// 計算 fibre total
	var totalDietaryFiber float64 = 0
	for _, dailyMealRecordEntity := range dailyMealRecordEntities {
		totalDietaryFiber = totalDietaryFiber + dailyMealRecordEntity.DietaryFiber
	}
	nutrientModel.Fibre.Total = decimal(totalDietaryFiber / float64(nums))
	nutrientModel.Fibre.Suggest = decimal(dietingPlanModel.Dietaryfiber * 1)


	return nutrientModel, nil
}

// 取得卡路里、水份的資料模型
func (r *ReportService) getCalorieAndWaterModel(memberEntity models.Member) (structs.Calorie, structs.Water, error) {

	// 計算平均卡路里、水份
	var calorieModel structs.Calorie
	var waterModel structs.Water

	memberDailyMainRecordEntities, err := r.getMemberDailyMainRecords(memberEntity)
	if err != nil {
		return calorieModel, waterModel, err
	}

	// 預設平均是 0
	calorieModel.Average = 0
	waterModel.Average = 0

	// 有資料的話就計算平均
	if len(memberDailyMainRecordEntities) != 0 {

		// 有紀錄的天數
		totalRecordDays := len(memberDailyMainRecordEntities)

		// 計算總卡路里
		var totalKcal float64 = 0
		for _, memberDailyMainRecordEntity := range memberDailyMainRecordEntities {
			totalKcal = totalKcal + float64(memberDailyMainRecordEntity.KcalTotal)
		}

		// 計算總水份
		var totalMoisture float64 = 0
		for _, memberDailyMainRecordEntity := range memberDailyMainRecordEntities {
			totalMoisture = totalMoisture + float64(memberDailyMainRecordEntity.MoistureTotal)
		}
		calorieModel.Average = decimal(totalKcal / float64(totalRecordDays))
		waterModel.Average = decimal(totalMoisture / float64(totalRecordDays))
	}

	// 取得 water base 的數值, 取得卡路里的 base
	if memberHealthData, err := r.getMemberHealthDatas(memberEntity); err != nil {
		return calorieModel, waterModel, err
	} else {
		if memberHealthData == nil {
			// 沒健康資料也要算下去
			return calorieModel, waterModel, nil
		} else {
			var dietingPlanModel structs.TrueDietingPlan
			if len(memberHealthData.DietingPlans) != 0 {
				if err := json.Unmarshal([]byte(memberHealthData.DietingPlans), &dietingPlanModel); err != nil {
					// return calorieModel, waterModel, err

					// 如果無法 parse 就全給 0
					trackLog.Error("member_id: "+memberEntity.ID+", json unmarshal true dietingPlanModel error in getCalorieAndWaterModel(): "+err.Error(), true)
					dietingPlanModel.Calories = 0
					dietingPlanModel.Carbohydrate = 0
					dietingPlanModel.Dietaryfiber = 0
					dietingPlanModel.Fats = 0
					dietingPlanModel.Protein = 0
					dietingPlanModel.Water = 0
				} else {
					if dietingPlanModel.Calories+dietingPlanModel.FixedKcal == 0 {
						calorieModel.Base = memberEntity.SuggestKcal
					} else {
						calorieModel.Base = services.Round(dietingPlanModel.Calories + dietingPlanModel.FixedKcal)
					}
					if dietingPlanModel.Water == 0 {
						weight, _ := strconv.Atoi(memberEntity.Weight)
						waterModel.Base = weight * 30
					} else {
						waterModel.Base = services.Round(dietingPlanModel.Water)
					}
				}
			}
		}
	}

	// 取得卡路里的 base
	// calorieModel.Base = memberEntity.SuggestKcal

	var diffCalorie, diffWater int
	if calorieModel.Base > 0 {
		diffCalorie = services.Round(calorieModel.Average / float64(calorieModel.Base) * 100)
	}
	if waterModel.Base > 0 {
		diffWater = services.Round(waterModel.Average / float64(waterModel.Base) * 100)
	}

	//if (float64(calorieModel.Base) - calorieModel.Average) == 0 {
	//	diffCalorie = 0
	//	diffWater = 0
	//} else {
	//	// 熱量差
	//	diffCalorie = decimal((float64(calorieModel.Base) - calorieModel.Average) / calorieModel.Average)
	//	diffWater = decimal((float64(waterModel.Base) - waterModel.Average) / waterModel.Average)
	//}

	// 用熱量差取得對應的 icon
	calorieModel.Icon = r.getIcon(diffCalorie)
	//if calorieModel.Icon = r.getIcon(diffCalorie); calorieModel.Icon == 0 {
	//	calorieModel.Icon = 3 //設為中間值
	//	// return calorieModel, waterModel, errors.New("calorie 沒有找到 icon 區間" + fmt.Sprintf("%f", diffCalorie))
	//}

	// 用熱量差取得對應的 icon
	waterModel.Icon = r.getIcon(diffWater)
	//if waterModel.Icon = r.getIcon(diffWater); waterModel.Icon == 0 {
	//	waterModel.Icon = 3 //設為中間值
	//	// return calorieModel, waterModel, errors.New("water 沒有找到 icon 區間" + fmt.Sprintf("%f", diffWater))
	//}

	return calorieModel, waterModel, nil
}

// 取小數後兩位
func decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}

// 取得熱量 icon 對應的數字判斷
//func (r *ReportService) getCalorieIcon(diffNumber float64) int {
//	if diffNumber < -0.4 {
//		return 1
//	}
//	if diffNumber >= -0.4 && diffNumber < -0.25 {
//		return 2
//	}
//	if diffNumber >= -0.25 && diffNumber < 0.25 {
//		return 3
//	}
//	if diffNumber >= 0.25 && diffNumber < 0.4 {
//		return 4
//	}
//	if diffNumber >= 0.4 {
//		return 5
//	}
//	return 0
//}

// 取得水分 icon 對應的數字判斷
//func (r *ReportService) getWaterIcon(diffNumber float64) int {
//	if diffNumber < -0.8 {
//		return 1
//	}
//	if diffNumber >= -0.8 && diffNumber < -0.4 {
//		return 2
//	}
//	if diffNumber >= -0.4 && diffNumber < 0.4 {
//		return 3
//	}
//	if diffNumber >= 0.4 && diffNumber < 0.8 {
//		return 4
//	}
//	if diffNumber >= 0.8 {
//		return 5
//	}
//	return 0
//}

// 新版取 icon 的判斷，calorie water 都一樣
func (r *ReportService) getIcon(diffNumber int) int {
	if diffNumber <= 0 {
		return 0
	} else if diffNumber >= 1 && diffNumber < 70 {
		return 2
	} else if diffNumber >= 70 && diffNumber < 120 {
		return 3
	} else {
		return 5
	}
}

// 取得用戶的卡路里紀錄
func (r *ReportService) getMemberKcalRecords(memberEntity models.Member) ([]models.DailyMainRecord, error) {

	var dailyMainRecordEntities []models.DailyMainRecord
	if errors := database.Mysql.Order("diary_date asc").Where("diary_date between ? and ? and member_id = ?", r.reportQueueParam.StartDate, r.reportQueueParam.EndDate, memberEntity.ID).Find(&dailyMainRecordEntities).GetErrors(); len(errors) != 0 {
		for _, err := range errors {
			r.handleError(&memberEntity, err)
		}
		return nil, errors[0]
	}

	var entities []models.DailyMainRecord
	for _, dailyMainRecordEntity := range dailyMainRecordEntities {
		entities = append(entities, dailyMainRecordEntity)
	}
	return entities, nil
}

// 取得用戶的健康資料
func (r *ReportService) getMemberHealthDatas(memberEntity models.Member) (*models.MemberHealthData, error) {

	var memberHealthDataEntities []models.MemberHealthData
	if errors := database.Mysql.Where(models.MemberHealthData{MemberID: memberEntity.ID}).Find(&memberHealthDataEntities).GetErrors(); len(errors) != 0 {
		for _, err := range errors {
			r.handleError(&memberEntity, err)
		}
		return nil, errors[0]
	}
	if len(memberHealthDataEntities) == 0 {
		return nil, nil
	}
	return &memberHealthDataEntities[0], nil
}

// 取得後台使用者
func (r *ReportService) getUser(userID int, memberEntity models.Member) (*models.User, error) {

	if len(userEntities) == 0 {
		if errors := database.Mysql.Find(&userEntities).GetErrors(); len(errors) != 0 {
			for _, err := range errors {
				r.handleError(&memberEntity, err)
			}
			return nil, errors[0]
		}
	}
	var entity *models.User
	for _, userEntity := range userEntities {
		if userEntity.ID == userID {
			entity = &userEntity
			break
		}
	}
	return entity, nil
}

// 取得每日的營養紀錄資料
func (r *ReportService) getMemberDailyMainRecords(memberEntity models.Member) ([]models.DailyMainRecord, error) {

	var dailyMainRecordEntities []models.DailyMainRecord
	if errors := database.Mysql.Order("diary_date asc").Where("diary_date between ? and ? and member_id = ?", r.reportQueueParam.StartDate, r.reportQueueParam.EndDate, memberEntity.ID).Find(&dailyMainRecordEntities).GetErrors(); len(errors) != 0 {
		for _, err := range errors {
			r.handleError(&memberEntity, err)
		}
		return nil, errors[0]
	}

	var entities []models.DailyMainRecord
	for _, dailyMainRecordEntity := range dailyMainRecordEntities {
		entities = append(entities, dailyMainRecordEntity)
	}
	return entities, nil
}

// 取得用戶筆記
func (r *ReportService) getMemberNote(memberEntity models.Member) (*models.MemberNote, error) {

	var memberNoteEntities []models.MemberNote
	if errors := database.Mysql.Where(models.MemberNote{MemberID: memberEntity.ID}).Find(&memberNoteEntities).GetErrors(); len(errors) != 0 {
		for _, err := range errors {
			r.handleError(&memberEntity, err)
		}
		return nil, errors[0]
	}
	if len(memberNoteEntities) == 0 {
		return nil, nil
	}
	return &memberNoteEntities[0], nil
}

// 錯誤處理
func (r *ReportService) handleError(memberEntity *models.Member, err error) {
	errorModel := structs.ErrorModel{
		MemberID:     memberEntity.ID,
		ErrorMessage: err.Error(),
	}
	r.Errors = append(r.Errors, errorModel)

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
