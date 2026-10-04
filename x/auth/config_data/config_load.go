package config_data

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	RewardFileNamePay          = "PayAward.csv"
	RewardFileNameLoginAndRank = "LoginAndRank.csv"
)

/*
	所有返利的配置文件。
	起服时写，之后并发读。
*/
var loadData *RewardLoadData

// 加载配置文件。
func LoadConfigData(dataPath string) error {
	loadData = &RewardLoadData{}

	dataAbsPath := filepath.Join(GetWorkPath(), dataPath)
	rd, errRead := ioutil.ReadDir(dataAbsPath)
	if errRead != nil {
		tilogs.L().Errorf("LoadConfigData ReadDir failed, dataAbsPath: %v", dataAbsPath)
		return errRead
	}

	for _, fi := range rd {
		switch fi.Name() {
		case RewardFileNamePay:
			if errLoad := loadPayAward(dataAbsPath, RewardFileNamePay); errLoad != nil {
				return errLoad
			}
		case RewardFileNameLoginAndRank:
			if errLoad := loadLoginAndRankAward(dataAbsPath, RewardFileNameLoginAndRank); errLoad != nil {
				return errLoad
			}
		}
	}

	return nil
}

// loadPayAward 加载充值返利的配置文件。
func loadPayAward(dataAbsPath, fileName string) error {
	file, err := os.Open(filepath.Join(dataAbsPath, fileName))
	if err != nil {
		tilogs.L().Errorf("loadPayAward os.Open failed, err: %v, dataAbsPath: %v, fileName: %v", err, dataAbsPath, fileName)
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	if reader == nil {
		errReader := fmt.Errorf("loadPayAward csv.NewReader failed, dataAbsPath: %v, fileName: %v", dataAbsPath, fileName)
		tilogs.L().Errorf(errReader.Error())
		return errReader
	}

	records, errAll := reader.ReadAll()
	if errAll != nil {
		tilogs.L().Errorf("loadPayAward reader.ReadAll failed, errAll: %v, dataAbsPath: %v, fileName: %v", errAll, dataAbsPath, fileName)
		return errAll
	}

	recordNum := len(records)
	loadData.PayRewardMap = make(map[string]*PayRewardConfig, recordNum)
	for i := 1; i < recordNum; i++ {
		tempRecord := records[i]

		// 玩家账号ID。
		tempUserID := tempRecord[0]
		if tempUserID == "" {
			errID := fmt.Errorf("loadPayAward tempUserID is blank, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				dataAbsPath, fileName, i, tempRecord)
			tilogs.L().Errorf(errID.Error())
			return errID
		}
		if _, ok := loadData.PayRewardMap[tempUserID]; ok {
			// 存在重复的UserID。
			errRepeat := fmt.Errorf("loadPayAward repeat userID, userID: %v, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				tempUserID, dataAbsPath, fileName, i, tempRecord)
			tilogs.L().Errorf(errRepeat.Error())
			return errRepeat
		}

		// 玩家累充金额。
		tempMoneyScore, errA2I := strconv.Atoi(tempRecord[1])
		if errA2I != nil {
			tilogs.L().Errorf("loadPayAward A2I failed, errA2I: %v, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				errA2I, dataAbsPath, fileName, i, tempRecord)
			return errA2I
		}

		loadData.PayRewardMap[tempUserID] = &PayRewardConfig{
			UserID:     tempUserID,
			MoneyScore: int32(tempMoneyScore),
		}
	}

	return nil
}

// GetPayRewardConfig 获取指定账号的充值返利信息。
func GetPayRewardConfig(userID string) *PayRewardConfig {
	cfg, ok := loadData.PayRewardMap[userID]
	if !ok {
		return nil
	}
	return cfg
}

// loadLoginAndRankAward 加载登陆和排名的配置文件。
func loadLoginAndRankAward(dataAbsPath, fileName string) error {
	file, err := os.Open(filepath.Join(dataAbsPath, fileName))
	if err != nil {
		tilogs.L().Errorf("loadLoginAndRankAward os.Open failed, err: %v, dataAbsPath: %v, fileName: %v", err, dataAbsPath, fileName)
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	if reader == nil {
		errReader := fmt.Errorf("loadLoginAndRankAward csv.NewReader failed, dataAbsPath: %v, fileName: %v", dataAbsPath, fileName)
		tilogs.L().Errorf(errReader.Error())
		return errReader
	}

	records, errAll := reader.ReadAll()
	if errAll != nil {
		tilogs.L().Errorf("loadLoginAndRankAward reader.ReadAll failed, errAll: %v, dataAbsPath: %v, fileName: %v", errAll, dataAbsPath, fileName)
		return errAll
	}

	recordNum := len(records)
	loadData.LoginAndRankMap = make(map[string]*LoginAndRankConfig, recordNum)
	for i := 1; i < recordNum; i++ {
		tempRecord := records[i]

		// 玩家账号ID。
		tempUserID := tempRecord[0]
		if tempUserID == "" {
			errID := fmt.Errorf("loadLoginAndRankAward tempUserID is blank, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				dataAbsPath, fileName, i, tempRecord)
			tilogs.L().Errorf(errID.Error())
			return errID
		}
		if _, ok := loadData.LoginAndRankMap[tempUserID]; ok {
			// 存在重复的UserID。
			errRepeat := fmt.Errorf("loadLoginAndRankAward repeat userID, userID: %v, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				tempUserID, dataAbsPath, fileName, i, tempRecord)
			tilogs.L().Errorf(errRepeat.Error())
			return errRepeat
		}

		// 玩家登陆天数。
		tempLoginDay, errA2ILD := strconv.Atoi(tempRecord[1])
		if errA2ILD != nil {
			tilogs.L().Errorf("loadLoginAndRankAward A2I failed, errA2ILD: %v, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				errA2ILD, dataAbsPath, fileName, i, tempRecord)
			return errA2ILD
		}

		// 玩家公平1v1的段位。
		tempFair1v1Dan, errA2I1v1Dan := strconv.Atoi(tempRecord[2])
		if errA2I1v1Dan != nil {
			tilogs.L().Errorf("loadLoginAndRankAward A2I failed, errA2I1v1Dan: %v, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				errA2I1v1Dan, dataAbsPath, fileName, i, tempRecord)
			return errA2I1v1Dan
		}

		// 玩家公平3v3的段位。
		tempFair3v3Dan, errA2I3v3Dan := strconv.Atoi(tempRecord[3])
		if errA2I3v3Dan != nil {
			tilogs.L().Errorf("loadLoginAndRankAward A2I failed, errA2I3v3Dan: %v, dataAbsPath: %v, fileName: %v, rowNum(from 0): %v, rowContent: %v",
				errA2I3v3Dan, dataAbsPath, fileName, i, tempRecord)
			return errA2I3v3Dan
		}

		loadData.LoginAndRankMap[tempUserID] = &LoginAndRankConfig{
			UserID:     tempUserID,
			LoginDay:   int32(tempLoginDay),
			Fair1v1Dan: int32(tempFair1v1Dan),
			Fair3v3Dan: int32(tempFair3v3Dan),
		}
	}

	return nil
}

// GetLoginAndRankRewardConfig 获取指定账号的登陆和排名信息。
func GetLoginAndRankRewardConfig(userID string) *LoginAndRankConfig {
	cfg, ok := loadData.LoginAndRankMap[userID]
	if !ok {
		return nil
	}
	return cfg
}

func GetWorkPath() string {
	workPath, _ := os.Getwd()
	workPath, _ = filepath.Abs(workPath)
	AppPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	tilogs.L().Debugf("data path work=%s, app=%s", workPath, AppPath)
	appConfigPath := AppPath
	if workPath != AppPath {
		if err := os.Chdir(AppPath); err != nil {
			appConfigPath = workPath
		}
	}
	return appConfigPath
}
