package ip2loc

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ip2location/ip2location-go"
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/signalhandler"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	db  *ip2location.DB
	C2C map[string]string // [国家代码]大洲名称，譬如 ["CN"] "Asia"，由于某些国家存在于两个大洲，初始化时总会取到最后的值
)

func init() {
	C2C = make(map[string]string, 256) // 国家和地区总数量
}

func InitReloadIp(fIPDB string) error {
	cfg := NewIPConfig(fIPDB, true, func(lcfgname string, cmd config.LoadCmd) error {
		if cmd == config.Load || cmd == config.Reload {
			if db != nil {
				db.Close()
				db = nil
			}
			_db, err := ip2location.OpenDB(lcfgname)
			if err != nil {
				tilogs.L().Errorf("config path %s load ip config fail %v", lcfgname, err)
				return err
			}
			db = _db
			tilogs.L().Infof("load ip2loc %s", lcfgname)
		} else {
			tilogs.L().Errorf("unload ip db config not supported")
		}
		return nil
	})
	if cfg == nil {
		return fmt.Errorf("load ip db config fail")
	}
	return nil
}
func NewIPConfig(cfgname string, reload bool, loadf func(string, config.LoadCmd) error) *config.Config {
	var cfg config.Config
	cfg.AppConfigPath = NewIPConfigPath(cfgname)
	file, inerr := os.Open(cfg.AppConfigPath)
	log.Println("NewConfig: ", cfg.AppConfigPath)
	if inerr != nil {
		log.Printf("NewConfig %s err %s \n", cfgname, inerr.Error())
		return nil
	}
	defer file.Close()
	md5h := md5.New()
	io.Copy(md5h, file)
	cfg.Md5 = hex.EncodeToString(md5h.Sum(nil))
	if loadf != nil {
		cfg.LoadFunc = loadf
		if err := loadf(cfg.AppConfigPath, config.Load); err != nil {
			log.Printf("NewConfig %s loadf err %s \n", cfgname, err.Error())
			return nil
		}
		if reload {
			signalhandler.SignalReloadFunc(cfg.AppConfigPath, func() {
				cfg.Reload()
			})
		}
	}
	return &cfg
}
func NewIPConfigPath(cfgname string) string {
	workPath, _ := os.Getwd()
	workPath, _ = filepath.Abs(workPath)
	//initialize default configurations
	AppPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	appConfigPath1 := filepath.Join(AppPath, "ipdata", cfgname)

	appConfigPath := appConfigPath1
	if util.FileExists(appConfigPath1) {
		os.Chdir(AppPath)
	} else {
		appConfigPath1 = filepath.Join(workPath, "ipdata", cfgname)
		appConfigPath = appConfigPath1
		if util.FileExists(appConfigPath1) {
			appConfigPath = appConfigPath1
		}
	}
	return appConfigPath
}

func InitReloadCountry(fCountry2Continent string) error {
	cfg := NewIPConfig(fCountry2Continent, true, func(lcfgname string, cmd config.LoadCmd) error {
		if cmd == config.Load || cmd == config.Reload {
			// 初始化国家/大洲列表
			if !util.FileExists(lcfgname) {
				return fmt.Errorf("local country to continent file not exist")
			}

			if err := loadCountry2ContinentInfo(lcfgname); err != nil {
				tilogs.L().Errorf("load country to continent file fail, %v", err)
				return err
			}
			tilogs.L().Infof("load country config %s", lcfgname)
		} else {
			tilogs.L().Errorf("country config not supporte unload")
		}
		return nil
	})
	if cfg == nil {
		return fmt.Errorf("load country config failed")
	}
	return nil
}

func Close() {
	if db != nil {
		db.Close()
	}
}

func loadCountry2ContinentInfo(fCountry2Continent string) error {
	f, err := os.OpenFile(fCountry2Continent, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	C2CTmp := make(map[string]string)
	n := 0
	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	for {
		n++
		ss, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// 跳过首行
		if n == 1 {
			continue
		}
		C2CTmp[ss[1]] = ss[3] // [国家代码]大洲名称，譬如 ["CN"] "Asia"
	}
	C2C = C2CTmp
	return nil
}

// IP2Location 使用离线库获取IP地址的大洲和国家名称
func IP2Location(ip string) (continent, country string, err error) {
	result, err := db.Get_all(ip)
	if err != nil {
		tilogs.L().Errorf("IP2Location Get_all err %s, ip %s", err.Error(), ip)
		return "", "", err
	}
	country = result.Country_long
	continent = C2C[result.Country_short] // empty is acceptable

	// 判断地区是否存在
	if _, ok := C2C[result.Country_short]; !ok {
		tilogs.L().Infof("IP2Location no continentData for ip %s", ip)
	}

	// 向sentry发一个错误作为统计
	if country == "" || country == "-" {
		tilogs.L().Infof("IP2Location no data for ip %s", ip)
		//err := metrics.SimpleSend("ticore.ip2loc.parseFail", "1")
		//if err != nil {
		//	logs.Warn("metrics fail %v", err)
		//}
		return "", "", err
	}

	return
}
