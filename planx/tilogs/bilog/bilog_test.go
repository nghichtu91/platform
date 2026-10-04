package bilog

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

const path = "./log"

func TestNewFile(t *testing.T) {
	defer os.RemoveAll(path)
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()

	bilog := CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	b := []byte("boo!")
	n, err := bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	existsWithLen(bilog.getCurFileName(), n, t)
	fileCount(path, 2, t)
}

func TestExistFile(t *testing.T) {
	defer os.RemoveAll(path)
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()

	bilog := CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	b := []byte("boo!")
	n, err := bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)

	bilog = CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	n, err = bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	existsWithLen(bilog.getCurFileName(), n*2, t)
	fileCount(path, 2, t)
}

func TestTimeRotate(t *testing.T) {
	defer os.RemoveAll(path)
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()

	bilog := CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	b := []byte("boo!")
	n, err := bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	existsWithLen(bilog.getCurFileName(), n, t)
	fileCount(path, 2, t)

	bilog = CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	time_offset_fortest = time.Hour * 24
	defer func() { time_offset_fortest = 0 }()
	n, err = bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	existsWithLen(bilog.getCurFileName(), n, t)
	fileCount(path, 3, t)
}

func TestLogSuffix(t *testing.T) {
	defer os.RemoveAll(path)
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()

	bilog := CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	b := []byte("boo!")
	n, err := bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	existsWithLen(bilog.getCurFileName(), n, t)
	fileCount(path, 2, t)

	fn := bilog.getCurFileName()
	err = os.Rename(fn, fmt.Sprintf("%s.1", fn))
	isNil(err, t)

	bilog = CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"", DayRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	n, err = bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	existsWithLen(fmt.Sprintf("%s.1", fn), n*2, t)
	fileCount(path, 2, t)
}

func TestWriteRotate(t *testing.T) {
	defer os.RemoveAll(path)
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()

	bilog := CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"Asia/Shanghai", HourRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	b := []byte("boo!")
	n, err := bilog.Write(b)
	isNil(err, t)
	equals(len(b), n, t)
	n, err = bilog.Write(b)
	isNil(err, t)
	equals(len(b), n, t)

	time_offset_fortest = time.Hour
	defer func() { time_offset_fortest = 0 }()
	n, err = bilog.Write(b)
	isNil(err, t)
	equals(len(b), n, t)
	existsWithLen(bilog.getCurFileName(), n, t)
	fileCount(path, 3, t)

	time_offset_fortest = time.Hour * 2
	time.Sleep(time.Second)
	fileCount(path, 4, t)

	n, err = bilog.Write(b)
	isNil(err, t)
	bilog.close()
	equals(len(b), n, t)
	fileCount(path, 4, t)
}

func TestTailRotate(t *testing.T) {
	//defer os.RemoveAll(path)
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()

	tailFileSize = 1
	bilog := CreateBiLog(filepath.Join(path, "logiclog_%y-%m-%d_%h.log"),
		"Asia/Shanghai", HourRotate, filepath.Join(path, "logiclog.log"))
	notNil(bilog, t)
	bilog.Write([]byte("test1"))
	time.Sleep(6 * time.Second)
	bilog.Write([]byte("test2"))
	time.Sleep(6 * time.Second)
	bilog.Write([]byte("test3"))
	bilog.close()
}

// fileCount checks that the number of files in the directory is exp.
func fileCount(dir string, exp int, t testing.TB) {
	files, err := ioutil.ReadDir(dir)
	isNilUp(err, t, 1)
	// Make sure no other files were created.
	equalsUp(exp, len(files), t, 1)
}

// existsWithLen checks that the given file exists and has the correct length.
func existsWithLen(path string, length int, t testing.TB) {
	info, err := os.Stat(path)
	isNilUp(err, t, 1)
	equalsUp(int64(length), info.Size(), t, 1)
}
