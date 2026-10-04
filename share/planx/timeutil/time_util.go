package timeutil

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/nghichtu91/platform/share/planx/rand_pool"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

/*
#Time Util
---------------------
##Day Time
##Week Time
*/

// 注意时区的影响
// TBD WeekTime支持其他时区

const (
	MinSec        = 60
	HourSec       = 60 * MinSec
	DaySec        = 24 * 3600
	WeekDayCount  = 7
	WeekSec       = WeekDayCount * 24 * 3600
	FourWeekSec   = 4 * WeekSec
	YearSec       = 365 * DaySec
	TimeSlotMin   = 30
	TimeSlotSec   = TimeSlotMin * MinSec
	RefreshOffset = 5  // 默认计算刷新的时间点为凌晨5点
	PM8HourNum    = 20 // 晚上8点刷新
	HourMin5AM    = "05:00"
)

const (
	Monday    = 1
	Tuesday   = 2
	Wednesday = 3
	Thursday  = 4
	Friday    = 5
	Saturday  = 6
	Sunday    = 7
)

const (
	TZ_CN   = "Asia/Shanghai"    // 国服时区
	TZ_HMT  = "Asia/Taipei"      // 港澳台时区
	TZ_AME  = "Etc/GMT+5"        // 美洲时区
	TZ_EMEA = "Etc/GMT-3"        // 欧洲时区
	TZ_JP   = "Asia/Tokyo"       // 日本时区
	TZ_VN   = "Asia/Ho_Chi_Minh" // 越南时区
)

// 如果后续添加新的时区, 请在此处添加对应的区域标识
// 需要同步更新 jws2/merger/core/logics.go:GetDuplicateNamePrefix 函数中, 关于地区的判断和改名的处理.
//	现在默认除了大陆是汉语, 其他地区都是英语.

const (
	ZoneCN  = "CN"  // 国服区域.
	ZoneHMT = "HMT" // 港澳台+欧美亚区域
	ZoneJP  = "JP"  // 日本区域
	ZoneVN  = "VN"  // 越南区域
	ZoneUnk = "Unk" // 未知区域
)

const (
	TimeStrFormat_YMDHMS = "2006-1-2 15:04:05"
	TimeStrFormat_HMSDMY = "15:04:05 2/1/2006"

	TimeStrFormat_YMDHM = "2006-1-2 15:04"
	TimeStrFormat_HMDMY = "15:04 2/1/2006"

	TimeStrFormat_YMD = "2006-1-2"
	TimeStrFormat_DMY = "2/1/2006"
)

const (
	VN_TimeRegex_YMDHMS = `\d{4}-\d{1,2}-\d{1,2} \d{1,2}:\d{2}:\d{2}`
	VN_TimeRegex_YMDHM  = `\d{4}-\d{1,2}-\d{1,2} \d{1,2}:\d{2}`
	VN_TimeRegex_YMD    = `\d{4}-\d{1,2}-\d{1,2}`
)

var (
	timeConvertFormatMap = map[string]string{}
	regex2TimeFormatMap  = map[string]string{}
	regexSorted          = []string{VN_TimeRegex_YMDHMS, VN_TimeRegex_YMDHM, VN_TimeRegex_YMD}
)

func regFormat(string1 string, string2 string) {
	timeConvertFormatMap[string2] = string1
	timeConvertFormatMap[string1] = string2
}

func regRegex(string1 string, string2 string) {
	regex2TimeFormatMap[string1] = string2
}

func init() {
	regRegex(VN_TimeRegex_YMDHMS, TimeStrFormat_YMDHMS)
	regRegex(VN_TimeRegex_YMDHM, TimeStrFormat_YMDHM)
	regRegex(VN_TimeRegex_YMD, TimeStrFormat_YMD)

	regFormat(TimeStrFormat_YMDHMS, TimeStrFormat_HMSDMY)
	regFormat(TimeStrFormat_YMDHM, TimeStrFormat_HMDMY)
	regFormat(TimeStrFormat_YMD, TimeStrFormat_DMY)
}

// GetZone 获取所属大区, QA服用于区分部署的服务器归属于哪个区域
func GetZone() string {
	switch time.Local.String() {
	case TZ_CN:
		return ZoneCN
	case TZ_HMT, TZ_AME, TZ_EMEA:
		return ZoneHMT
	case TZ_JP:
		return ZoneJP
	case TZ_VN:
		return ZoneVN
	default:
		return ZoneUnk
	}
}

// 处理时区产生的问题
var (
	TimeUnixBegin     time.Time
	TimeUnixBeginUnix int64
	FirstMonday       time.Time
	FirstMondayUnix   int64
	serverStartTime   map[uint]int64
)

func InitTimeConst() {
	var err error
	TimeUnixBegin, err = time.ParseInLocation("2006/1/2", "2000/1/1", time.Local)
	if err != nil {
		panic(fmt.Errorf("TimeUnixBegin Parse err %s", err.Error()))
	}
	// logs.Trace("TimeUnixBegin %v", TimeUnixBegin)
	TimeUnixBeginUnix = TimeUnixBegin.Unix()

	FirstMonday, err = time.ParseInLocation("2006/1/2", "2000/1/3", time.Local)
	if err != nil {
		panic(fmt.Errorf("FirstMonday Parse err %s", err.Error()))
	}
	// logs.Trace("FirstMonday %v", FirstMonday)
	FirstMondayUnix = FirstMonday.Unix()
}

func SetServerStartTimes(shard2ServerStartTime map[uint]int64) {
	if len(shard2ServerStartTime) <= 0 {
		// 这就不让起服了
		//		panic(errors.New("GetServerStartTime Cfg Err"))
		return
	}
	serverStartTime = make(map[uint]int64, len(shard2ServerStartTime))
	for shard, sst := range shard2ServerStartTime {
		serverStartTime[shard] = sst
		tilogs.L().Infof("set shard %d start time %d", shard, sst)
	}
}

// 获取开服时间
// 参数 sid 要用配置下的 sid，不要用玩家身上的 sid，否则合服后会有问题
// gamex下有个接口直接调用这个方法，可以直接调用那个接口 gamedata.GetServerStartTime()
func GetServerStartTime(sid uint) int64 {
	return serverStartTime[sid]
}

func SetServerStartTime(sid uint, t int64) {
	serverStartTime[sid] = t
}

func IsSameDayUnix(t1, t2 int64) bool {
	return DailyBeginUnix(t1) == DailyBeginUnix(t2)
}

/*
func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return (y1 == y2) && (m1 == m2) && (d1 == d2)
}

func IsNowDay(t time.Time) bool {
	return IsSameDay(t, time.Now())
}
*/

// 当日零点的时间
func DailyBeginUnix(u int64) int64 {
	return u - (u-TimeUnixBeginUnix)%DaySec
}

// Cur5AM 获取离当前时间最近的凌晨5点的时间戳
// 一般用于计算刷新周期
func Cur5AM(ts int64) int64 {
	return DailyBeginUnix(ts) + RefreshOffset*HourSec
}

// Pre5AM 距离当前时间最近的上一个凌晨5点
func Pre5AM(ts int64) int64 {
	return DailyBeginUnix(ts) + RefreshOffset*HourSec - DaySec
}

// Next5AM 获取离当前时间最近的下一个凌晨5点的时间戳
// 如果当前时间为凌晨5点或之后，返回24小时之后的凌晨5点时间戳
// 一般用于计算刷新周期
func Next5AM(ts int64) int64 {
	rTs := DailyBeginUnix(ts) + RefreshOffset*HourSec
	if ts >= rTs {
		rTs += DaySec
	}
	return rTs
}

// 获取下一个对应时间戳的具体时间
func NextTimeWithOffset(ts, offset int64) int64 {
	rTs := DailyBeginUnix(ts) + offset
	if ts >= rTs {
		rTs += DaySec
	}
	return rTs
}

// Next8PM 获取离当前时间最近的下一个晚上8点的时间戳
// 如果当前时间为晚上8点或之后，返回24小时之后的晚上8点时间戳
// 一般用于计算特殊运营活动刷新周期
func Next8PM(ts int64) int64 {
	rTs := DailyBeginUnix(ts) + PM8HourNum*HourSec
	if ts >= rTs {
		rTs += DaySec
	}
	return rTs
}

// NextWeekMonday5AM 获取离当前时间最近的下一个周一凌晨5点的时间戳
// 如果当前时间为本周一凌晨5点，返回下周一凌晨5点的时间戳
// 算出本周过去了多少秒，当前时间减去秒数，再加一周
func NextWeekMonday5AM(ts int64) int64 {
	return ts - (ts-FirstMondayUnix)%WeekSec + WeekSec
}

// 获取当天的以hms为起始时间点的秒数，hms如5:00
func TodayBeginUnix(hms string) int64 {
	t_now := time.Now()
	prefix := t_now.Format("20060102")
	expected, err := time.Parse("2006010215:04", prefix+hms)
	if err != nil {
		tilogs.L().Errorf("TodayBeginUnix time.Parse err %s", err.Error())
		return -1
	}
	ret := expected.Unix()
	if t_now.Unix() < ret {
		return ret - DaySec
	}
	return ret
}

// 获取下一个的以hms为起始时间点的秒数，hms如5:00
func NextDailyBeginUnix(hms string) int64 {
	_t := TodayBeginUnix(hms)
	if _t < 0 {
		return _t
	}
	if time.Now().Unix() >= _t {
		return _t + DaySec
	}
	return _t
}

// Week 以周一零点为开始
func IsSameWeek(t1, t2 time.Time) bool {
	return IsSameWeekUnix(t1.Unix(), t2.Unix())
}

func IsSameWeekUnix(u1, u2 int64) bool {
	w1 := (u1 - FirstMondayUnix) / WeekSec
	w2 := (u2 - FirstMondayUnix) / WeekSec
	return w1 == w2
}

func IsNowWeek(t time.Time) bool {
	return IsSameWeek(t, time.Now())
}

func IsNowWeekUnix(u int64) bool {
	return IsSameWeekUnix(u, time.Now().Unix())
}

func GetDayBefore(t_begin time.Time, t_end time.Time) int64 {
	// 注意时区的影响
	t_b := (t_begin.Unix() - TimeUnixBeginUnix) / DaySec
	t_e := (t_end.Unix() - TimeUnixBeginUnix) / DaySec

	return t_e - t_b
}

func GetDayBeforeUnix(t_begin, t_end int64) int64 {
	// 注意时区的影响
	t_b := (t_begin - TimeUnixBeginUnix) / DaySec
	t_e := (t_end - TimeUnixBeginUnix) / DaySec

	return t_e - t_b
}

func IsTimeBetween(begin, end, t time.Time) bool {
	if !begin.IsZero() && !t.After(begin) {
		return false
	}

	if !end.IsZero() && t.After(end) {
		return false
	}

	return true
}

func IsTimeBetweenUnix(begin, end, t int64) bool {
	return (begin <= 0 || t > begin) && (end <= 0 || t <= end)
}

/*
	 WeekTime
	   从每周一零点开始每个半点和整点的计数
	   如 0 - 周一 0:00
	      1 -      0:30
		  2 -      1:00
		  2 -      1:11
		  3 -      1:30
*/
func WeekTime(u int64, offset_tm string) int64 {
	return WeekTimeUnix(u, offset_tm)
}

func WeekTimeUnix(u int64, offset_tm string) int64 {
	ws := (u - FirstMondayUnix) % WeekSec
	t0 := ws / TimeSlotSec
	tb := DailyTimeFromString(offset_tm)
	if t0 >= tb {
		return t0 - tb
	} else {
		return t0 + (48 - tb)
	}
}

// 将周几和时间转换成一周内的时间段，效率一般，适合读配置时预先转换
func WeeklyTimeFromString(weekDay int, offset_tm string, tm string) int64 {
	t := fmt.Sprintf("2015/8/%v %v", 2+weekDay, tm)
	t_in_time, err := time.ParseInLocation("2006/1/2 15:04", t, time.Local)
	if err != nil {
		tilogs.L().Errorf("WeeklyTimeFromString Parse %s err %s", t, err.Error())
		return -1
	}
	return WeekTime(t_in_time.Unix(), offset_tm)
}

func WeeklyBeginUnix(offset_tm string, u int64) int64 {
	return FirstMondayUnix + ((u-FirstMondayUnix)/WeekSec)*WeekSec + DailyTimeFromString(offset_tm)*TimeSlotSec
}

func WeeklySlot2Unix(slot int64, offset_tm string, u int64) int64 {
	return WeeklyBeginUnix(offset_tm, u) + slot*TimeSlotSec
}

func IsNowMouth(t time.Time) bool {
	y1, m1, _ := time.Now().Date()
	y2, m2, _ := t.Date()
	return (y1 == y2) && (m1 == m2)
}

/*
   DailyTime
   从每天零点开始每个半点和整点的计数
   如 0 -      0:00
      1 -      0:30
	  2 -      1:00
	  2 -      1:11
	  3 -      1:30
*/

func DailyTime(t time.Time) int64 {
	return DailyTimeUnix(t.Unix())
}

func DailyTimeUnix(u int64) int64 {
	nb := DailyBeginUnix(u)
	return (u - nb) / TimeSlotSec
}

// DailyRealTimeUnix 当天零点开始的秒数
func DailyRealTimeUnix(u int64) int64 {
	nb := DailyBeginUnix(u)
	return u - nb
}

func DailyTime2UnixTime(now_time int64, daily_time int64) int64 {
	nb := DailyBeginUnix(now_time)
	return nb + (TimeSlotSec * daily_time)
}

// 从字符串转成DailyTime，效率一般，适合读配置时预先转换
// "00:00" -> 0
// "00:30" -> 1
// "01:00" -> 2
func DailyTimeFromString(s string) int64 {
	t := "2015/1/1 "
	if s == "24:00" {
		temp_time, err := time.ParseInLocation("2006/1/2 15:04", t+"00:00", time.Local)
		if err != nil {
			tilogs.L().Errorf("DailyTimeFromString Parse %s err %s", "00:00", err.Error())
			return -1
		}
		return DailyTime(temp_time) + 48
	}
	t_in_time, err := time.ParseInLocation("2006/1/2 15:04", t+s, time.Local)
	if err != nil {
		tilogs.L().Errorf("DailyTimeFromString Parse %s err %s", s, err.Error())
		return -1
	}
	return DailyTime(t_in_time)
}

// DailyRealTimeFromString 从字符串转成 DailyRealTime, 效率一般，适合读配置时预先转换
// "05:00" -> 18000
func DailyRealTimeFromString(s string) int64 {
	pre := "2015/1/1 "
	t, err := time.ParseInLocation("2006/1/2 15:04", pre+s, time.Local)
	if err != nil {
		tilogs.L().Errorf("DailyTimeFromString Parse %s err %s", s, err.Error())
		return -1
	}
	tUnix := t.Unix()
	return tUnix - DailyBeginUnix(tUnix)
}

func TimeFromString(s string) int64 {
	tInTime, err := time.ParseInLocation("2006/1/2 15:04", s, time.Local)
	if err != nil {
		tilogs.L().Errorf("DailyTimeFromString Parse %s err %s", s, err.Error())
		return -1
	}
	return tInTime.Unix()
}

func TimeFromString2(s string) int64 {
	tInTime, err := time.ParseInLocation("2006/1/2 15:04:00", s, time.Local)
	if err != nil {
		tilogs.L().Errorf("DailyTimeFromString Parse %s err %s", s, err.Error())
		return -1
	}
	return tInTime.Unix()
}

// 当前时间距离某个偏移时间的整天数
func DayOffsetFromString(s string, u int64) int64 {
	t := "2015/1/1 "
	t_in_time, err := time.ParseInLocation("2006/1/2 15:04", t+s, time.Local)
	if err != nil {
		tilogs.L().Errorf("DayOffsetFromString Parse %s err %s", s, err.Error())
		return -1
	}
	return (u - t_in_time.Unix()) / DaySec
}

/*
计算时先计算这一段时间中可以增加多少,
将结余的时间返回
*/
func AccountTime2Point(now_unix_sec, last_unix_sex, one_need int64) (int64, int64) {
	/*
	                 + s +
	   ----+---------+---+---+--------
	      last     this now  next
	   ----+---------+---+---+--------
	       +    add  +   + r +

	                 + one   +
	*/
	time_in_v := now_unix_sec - last_unix_sex
	s_v := time_in_v % one_need
	add_v := time_in_v - s_v

	return add_v / one_need, s_v
}

/*
纳秒从2015年8月1日开始, 目前用在Log中标记时间(数值比较小)
*/
func tiNowUnixNanoStartTime() int64 {
	return time.Date(2015, 8, 1, 0, 0, 0, 0, time.UTC).UnixNano() // 这个直接用UTC
}

var TiNowUnixNanoStart = tiNowUnixNanoStartTime()

func TiNowUnixNano() int64 {
	return time.Now().UnixNano() - TiNowUnixNanoStart
}

/*
转换weekday，time里周日为0，cfg一般为7
*/
func TimeWeekDayTranslate(weekday time.Weekday) int {
	ret := int(weekday)
	if ret == 0 {
		ret = 7
	}
	return ret
}

/*
转换weekday，time里周日为0，cfg一般为7
*/
func TimeWeekDayTranslateFromCfg(weekdayCfg int) int {
	ret := weekdayCfg
	if weekdayCfg == 7 {
		ret = 0
	}
	return ret
}

// 获取当前时间是周几 0 - 6
func GetWeek(t int64) int {
	tt := time.Unix(t, 0)
	return int(tt.In(time.Local).Weekday())
}

// 返回当前时间所在天的某个整点时间
func GetCurDayTimeAtHour(t int64, hour int) (ret int64) {
	tmpTime := time.Unix(t, 0).In(time.Local)
	ret = time.Date(tmpTime.Year(), tmpTime.Month(), tmpTime.Day(), hour,
		0, 0, 0, time.Local).Unix()
	return
}

func GetWeekTime(now_t int64, weekDay int, hourMin string) int64 {
	t := time.Unix(now_t, 0)
	t = t.In(time.Local)
	_t, err := time.ParseInLocation("2006-1-2 15:04",
		fmt.Sprintf("%d-%d-%d %s", t.Year(), t.Month(), t.Day(),
			hourMin), time.Local)
	if err != nil {
		tilogs.L().Errorf("GetNextWeekTime time.ParseInLocation err %v", err)
		return 0
	}
	t = _t
	if weekDay > int(time.Saturday) {
		weekDay = int(time.Sunday)
	}
	ts := t.Unix() + int64(weekDay-int(t.Weekday()))*int64(DaySec)
	return ts
}

func GetNextWeekTime(now_t int64, weekDay int, hourMin string) int64 {
	ts := GetWeekTime(now_t, weekDay, hourMin)
	if ts <= now_t {
		return ts + int64(WeekSec)
	}
	return ts
}

func Clock(t time.Time) (int, int, int) {
	return t.In(time.Local).Clock()
}

// 输入参数：Unix时间戳字符串，输出当地时间time结构
// 警告：如果返回错误，不要使用返回值。
func UnixStringToTimeByLoacl(unixTimeString string) (time.Time, error) {
	// 将输入的字符串转换为int64
	unixtimeint64, err := strconv.ParseInt(unixTimeString, 10, 64)
	if err != nil {
		tilogs.L().Errorf("time_util:UnixStringToTimeBylocal err :%v", err)
		return time.Time{}, err
	}
	tilogs.L().Debugf("当前时区为：%v", time.Local)
	// 将int64的时间戳转换成时间结构,并直接获取在当地时区下的时间结构
	unixtime := time.Unix(unixtimeint64, 0).In(time.Local)

	return unixtime, nil
}

// 用来判断是否国服，当前时区是否是东八区的上海时区
func IsAsiaShanghaiTZ() bool {
	return time.Local.String() == TZ_CN
}

func IsHMTTZ() bool {
	return time.Local.String() == TZ_HMT
}

func IsAMETZ() bool {
	return time.Local.String() == TZ_AME
}

func IsEMEATZ() bool {
	return time.Local.String() == TZ_EMEA
}

func IsVN() bool {
	return time.Local.String() == TZ_VN
}

// 用来判断是否日本版本
func IsJPTZ() bool {
	return time.Local.String() == TZ_JP
}

// 用来判断是否钛核国际服，包括港澳台，欧洲，美洲
func IsInternationalTZ() bool {
	return time.Local.String() == TZ_HMT || time.Local.String() == TZ_AME || time.Local.String() == TZ_EMEA || time.Local.String() == TZ_VN
}

// 用来判断是否是开启语音聊天的地区
func IsGameChatTZ() bool {
	return time.Local.String() == TZ_CN || time.Local.String() == TZ_VN
}

// 比较是否超时，支持改时间
func CheckTimeout(now, last int64, delt int64) bool {
	if now-last > delt {
		return true
	}
	// 防止往回改时间的情况发生
	if last-now > delt {
		return true
	}
	return false
}

// RandDuration 各个服务错峰使用
// 会随机增加时间到输入的时间段范围内
// 使用了全局的随机来源，可能对性能有一定影响，
// 只允许在模块Start等特定情况下使用，不应用于循环中
func RandDuration(d time.Duration) time.Duration {
	if int64(d) == 0 {
		return d
	}
	return time.Duration(int64(d) + rand_pool.Int63n(int64(d)))
}

func Timing(start time.Time, format string, args ...interface{}) {
	tilogs.L().Infof("[%v] "+format, append([]interface{}{time.Since(start)}, args...)...)
}

// ConvertTimeStrFormat
// 将时间字符串从一个格式转换为另一个格式
func ConvertTimeStrFormat(timeStr string) string {
	// 如果不是类似2006-这样的年月日开头的字符串，不需要转
	dateFormatRegex := `^\d{4}-`
	re := regexp.MustCompile(dateFormatRegex)
	// 使用正则表达式匹配
	if !re.MatchString(timeStr) {
		return timeStr
	}
	for fromFormat, toFormat := range timeConvertFormatMap {
		t, err := time.ParseInLocation(fromFormat, timeStr, time.Local)
		if err != nil {
			continue
		}
		return t.Format(toFormat)
	}
	return timeStr
}

// 带有其他文字的原始时间字符串转换为指定格式的时间字符串
func ConvertVNTime(original string) string {
	for _, reg := range regexSorted {
		// 使用正则表达式匹配时间
		regex := regexp.MustCompile(reg)
		if !regex.MatchString(original) {
			continue
		}
		// 定义时间的布局
		layout := regex2TimeFormatMap[reg]
		// 定义新的时间格式
		newLayout := timeConvertFormatMap[layout]

		// 替换函数
		replaceFunc := func(match string) string {
			parsedTime, err := time.Parse(layout, match)
			if err != nil {
				fmt.Println("Error parsing time:", err)
				return match
			}
			return parsedTime.Format(newLayout)
		}

		// 使用正则表达式替换所有匹配的时间字符串
		tilogs.L().Debugf("ConvertVNTime original: %v", original)
		result := regex.ReplaceAllStringFunc(original, replaceFunc)
		tilogs.L().Debugf("ConvertVNTime result: %v", result)

		return result
	}
	return original
}
