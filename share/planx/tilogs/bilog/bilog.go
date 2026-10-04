package bilog

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"
)

/*
专为游戏业务记录bi日志，发行方需要的日志功能实现
	1、目前只提供根据天和小时两种log文件分割方式，时间可配置时区
	2、log文件名根据模板样式生成，可满足发行对log文件名的不同需求；
		如：同样是按小时分割，可能的需求有logiclog_2018-10-19_4.log和logiclog_2018-10-19-4.log两种
	3、若已有log文件续写，并因为权限等原因，不能往已有的log文件中写，则在文件名后加数字的方法，区别log文件；
		用此方法以防丢失log（一般不会发生，可以先忽视）
	4、为了应对长期没有写入，buff中的数据不能flush并且rotate的问题，加了定时rotate，1秒检测一次
	5、为了方便运维tail log，保持输出的log文件名不变，则在另写一份log，大小限制由tailFileSize控制，并且只多保留一份
		如：logiclog.log和logiclog.log.1，运维只需要tail logiclog.log即可
*/

type RotateType int

const (
	DayRotate RotateType = iota
	HourRotate
)

type BiLog struct {
	fileTemplate string // 文件名模板，如：/opt/supervisor/log/logiclog_%y-%m-%d_%h.log
	rotateType   RotateType
	local        *time.Location // 时区

	fileTS string
	file   *os.File

	fileForTail     *os.File
	fileNameForTail string

	writeChan chan []byte

	// mu   sync.Mutex
	quit chan struct{}
	wait util.WaitGroupWrapper
	once sync.Once
}

const (
	YearTag  = "%Y"
	MonthTag = "%M"
	DayTag   = "%D"
	HourTag  = "%H"

	buffSize = 4096
)

var time_offset_fortest time.Duration

//var tailFileSize = 1024 * 512 // 512k
var tailFileSize = 1024 * 1024 * 200 // 200M

/*
	创建一个用于写bilog的实例
	fileTemplate：log文件模板，如：/opt/supervisor/log/logiclog_%y-%m-%d_%h.log
				%y：代表年
				%m：代表月
				%d：代表日
				%h：代表小时
	localTime：时区，一般是"Asia/Shanghai"; 若为空字符串，则为utc时区
	rotateType：log分割方式；目前支持两种，按天和按小时
	fileForTail：方便运维tail log，单独写出的一份log的文件名，文件名中不用包含%y%m%d%h; 若每此需求则填空字符串即可
*/
func CreateBiLog(fileTemplate,
	localTime string,
	rotateType RotateType,
	fileForTail string) *BiLog {
	if !strings.Contains(fileTemplate, YearTag) &&
		!strings.Contains(fileTemplate, strings.ToLower(YearTag)) &&
		!strings.Contains(fileTemplate, MonthTag) &&
		!strings.Contains(fileTemplate, strings.ToLower(MonthTag)) &&
		!strings.Contains(fileTemplate, DayTag) &&
		!strings.Contains(fileTemplate, strings.ToLower(DayTag)) {
		tilogs.L().Errorf("CreateBiLog fileTemplate err, %s", fileTemplate)
		return nil
	}
	if rotateType == HourRotate &&
		!strings.Contains(fileTemplate, HourTag) &&
		!strings.Contains(fileTemplate, strings.ToLower(HourTag)) {
		tilogs.L().Errorf("CreateBiLog fileTemplate err, rotateType %d, fileTemple %s",
			rotateType, fileTemplate)
		return nil
	}
	if rotateType != HourRotate && rotateType != DayRotate {
		tilogs.L().Errorf("CreateBiLog rotateType err  %s", rotateType)
		return nil
	}
	if err := mkdirIfNotExist(fileTemplate); err != nil {
		tilogs.L().Errorf("CreateBiLog mkdirIfNotExist err  %s", rotateType)
		return nil
	}

	local := time.UTC
	if localTime != "" {
		l, err := time.LoadLocation(localTime)
		if err != nil {
			tilogs.L().Errorf("CreateBiLog LoadLocation err %s, %s", err.Error(), localTime)
			return nil
		}
		local = l
	}
	log := &BiLog{
		fileTemplate:    fileTemplate,
		rotateType:      rotateType,
		local:           local,
		quit:            make(chan struct{}, 1),
		writeChan:       make(chan []byte, 8192),
		fileNameForTail: fileForTail,
	}
	log.timeRotate()
	return log
}

func (l *BiLog) Write(p []byte) (n int, err error) {
	select {
	case l.writeChan <- p:
		return len(p), nil
	default:
		return 0, fmt.Errorf("bilog write chan full")
	}
}

func (l *BiLog) write(p []byte) (n int, err error) {
	// l.mu.Lock()
	// defer l.mu.Unlock()

	if l.file == nil {
		if err = l.openExistingOrNew(); err != nil {
			return 0, err
		}
	}
	// file for tail
	if l.fileNameForTail != "" && l.fileForTail == nil {
		if err := l.openFileForTail(); err != nil {
			return 0, err
		}
	}
	if l.getFileTS() != l.fileTS {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = l.file.Write(p)
	if err != nil {
		return n, err
	}

	// file for tail
	if l.fileNameForTail != "" {
		n, err = l.fileForTail.Write(p)
	}
	return n, err
}

func (l *BiLog) openExistingOrNew() error {
	lastFileName, nextFileName, fileTS, err := l.getFileName()
	if err != nil {
		return fmt.Errorf("getFileName err: %s", err)
	}
	if lastFileName == "" {
		return l.openNew(nextFileName, fileTS)
	}

	// open exist
	f, err := os.OpenFile(lastFileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// if we fail to open the old log file for some reason, just ignore
		// it and open a new log file.
		tilogs.L().Errorf("fail to open the old log, so open new, err %s\n", err.Error())
		return l.openNew(nextFileName, fileTS)
	}
	l.file = f
	l.fileTS = fileTS

	return nil
}

// openNew opens a new log file for writing, moving any old log file out of the
// way.  This methods assumes the file has already been closed.
func (l *BiLog) openNew(fileName, fileTS string) error {
	err := os.MkdirAll(filepath.Dir(l.fileTemplate), 0744)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}

	mode := os.FileMode(0644)
	// we use truncate here because this should only get called when we've moved
	// the file ourselves. if someone else creates the file in the meantime,
	// just wipe out the contents.
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}

	l.file = f
	l.fileTS = fileTS
	return nil
}

func (l *BiLog) openFileForTail() error {
	mode := os.FileMode(0644)
	if util.FileExists(l.fileNameForTail) {
		_f, err := os.OpenFile(l.fileNameForTail, os.O_APPEND|os.O_WRONLY, mode)
		if err != nil {
			return fmt.Errorf("BiLog openExistingOrNew O_APPEND open %s err %s",
				l.fileNameForTail, err.Error())
		}
		l.fileForTail = _f
		tilogs.L().Debugf("BiLog openFileForTail OpenFile success")
	} else {
		_f, err := os.OpenFile(l.fileNameForTail, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return fmt.Errorf("BiLog openExistingOrNew O_CREATE open %s err %s",
				l.fileNameForTail, err.Error())
		}
		l.fileForTail = _f
		tilogs.L().Debugf("BiLog openFileForTail CreateFile success")
	}
	return nil
}

func (l *BiLog) timeRotate() {
	l.wait.Wrap(func() {
		tickForTail := 0
		timerChan := timeutil.Timer10MS.After(time.Second)
		for {
			select {
			case <-timerChan:
				tickForTail++
				// l.mu.Lock()
				if l.file != nil && l.getFileTS() != l.fileTS {
					if l.getFileTS() != l.fileTS {
						l.rotate()
					}
				}
				if tickForTail >= 5 && l.fileNameForTail != "" && l.fileForTail != nil {
					tickForTail = 0
					n, err := util.FileSize(l.fileNameForTail)
					if err == nil {
						if n > int64(tailFileSize) {
							l.fileForTail.Close()
							l.rotateFileForTail()
							l.openFileForTail()
						}
					}
				}
				// l.mu.Unlock()
				timerChan = timeutil.Timer10MS.After(time.Second)
			case p := <-l.writeChan:
				_, err := l.write(p)
				if err != nil {
					tilogs.L().Errorf("BiLog write failed, err %s", err.Error())
				}
			case <-l.quit:
				err := l.close()
				if err != nil {
					tilogs.L().Errorf("BiLog close failed, err %s", err.Error())
				}
				return
			}
		}
	})
}

func (l *BiLog) rotate() error {
	if err := l.file.Close(); err != nil {
		return err
	}
	l.file = nil

	_, nextFileName, fileTS, err := l.getFileName()
	if err != nil {
		return err
	}
	if err := l.openNew(nextFileName, fileTS); err != nil {
		return err
	}
	return nil
}

func (l *BiLog) rotateFileForTail() {
	ft1 := fmt.Sprintf("%s.1", l.fileNameForTail)
	if util.FileExists(ft1) {
		if err := util.DelFile(ft1); err != nil {
			tilogs.L().Errorf("BiLog rotateFileForTail DelFile %s err %s\n", ft1, err.Error())
		}
	}
	if util.FileExists(l.fileNameForTail) {
		if err := util.FileRename(l.fileNameForTail, ft1); err != nil {
			tilogs.L().Errorf("BiLog rotateFileForTail FileExists %s err %s\n", ft1, err.Error())
		}
	}
}

func (l *BiLog) getFileName() (lastFileName, nextFileName, fileTS string, err error) {
	fn := l.getCurFileName()
	dir := filepath.Dir(l.fileTemplate)
	if !util.FileExists(dir) {
		return "", fn, l.getFileTS(), nil
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", "", "", fmt.Errorf("can't read log file directory: %s", err)
	}
	mn := -1
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fName := filepath.Join(dir, f.Name())
		if strings.Contains(fName, filepath.Clean(fn)) {
			ss := strings.Split(fName, ".")
			n, err := strconv.Atoi(ss[len(ss)-1])
			if err != nil {
				if mn < 0 {
					mn = 0
					lastFileName = fName
					continue
				}
			}
			if n > mn {
				mn = n
				lastFileName = fName
			}
		}
	}
	if mn >= 0 {
		return lastFileName,
			fmt.Sprintf("%s.%d", fn, mn+1), l.getFileTS(), nil
	}
	return "", fn, l.getFileTS(), nil
}

func (l *BiLog) getCurFileName() string {
	t := time.Now().In(l.local).Add(time_offset_fortest)
	fn := l.fileTemplate
	fn = strings.Replace(fn, YearTag, fmt.Sprintf("%02d", t.Year()), -1)
	fn = strings.Replace(fn, strings.ToLower(YearTag), fmt.Sprintf("%02d", t.Year()), -1)
	fn = strings.Replace(fn, MonthTag, fmt.Sprintf("%02d", t.Month()), -1)
	fn = strings.Replace(fn, strings.ToLower(MonthTag), fmt.Sprintf("%02d", t.Month()), -1)
	fn = strings.Replace(fn, DayTag, fmt.Sprintf("%02d", t.Day()), -1)
	fn = strings.Replace(fn, strings.ToLower(DayTag), fmt.Sprintf("%02d", t.Day()), -1)
	fn = strings.Replace(fn, HourTag, fmt.Sprintf("%02d", t.Hour()), -1)
	fn = strings.Replace(fn, strings.ToLower(HourTag), fmt.Sprintf("%02d", t.Hour()), -1)
	return fn
}

func (l *BiLog) getFileTS() string {
	t := time.Now().In(l.local).Add(time_offset_fortest)
	s := fmt.Sprintf("%d%d%d", t.Year(), t.Month(), t.Day())
	if l.rotateType == HourRotate {
		s = fmt.Sprintf("%s%d", s, t.Hour())
	}
	return s
}

func (l *BiLog) Close() error {
	// l.mu.Lock()
	// defer l.mu.Unlock()
	l.once.Do(func() {
		close(l.quit)
	})
	l.wait.Wait()
	return nil
}

// close closes the file if it is open.
func (l *BiLog) close() (err error) {
	if l.file != nil {
		err = l.file.Close()
		l.file = nil
	}
	if l.fileForTail != nil {
		l.fileForTail.Close()
		l.fileForTail = nil
	}
	return err
}

func mkdirIfNotExist(fileTemplate string) error {
	// 目录不存在，则创建目录
	i := strings.LastIndex(fileTemplate, "/")
	if i > 0 {
		path := fileTemplate[:i]
		_, errStat := os.Stat(path)
		if errStat != nil {
			if os.IsNotExist(errStat) {
				if errMk := os.MkdirAll(path, os.ModePerm); errMk != nil {
					tilogs.L().Errorf("CreateBiLog MkdirAll path %s err %s", path, errMk.Error())
					return errMk
				}
			} else {
				tilogs.L().Errorf("CreateBiLog Mkdir os.Stat path %s err %s", path, errStat.Error())
				return errStat
			}
		}
	}
	return nil
}
