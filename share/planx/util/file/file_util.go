package file

import (
	"bufio"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func CopyDirOrFile(from, to string) error {
	file, err := os.Stat(from)
	if err != nil {
		return err
	}

	if file.IsDir() {
		//文件夹拷贝给文件夹
		if list, err := ioutil.ReadDir(from); err == nil {
			for _, item := range list {
				if err = CopyDirOrFile(filepath.Join(from, item.Name()), filepath.Join(to, item.Name())); err != nil {
					return err
				}
			}
		}
	} else {
		//拷贝文件
		//先创建to文件夹
		p := filepath.Dir(to)
		if _, err = os.Stat(p); err != nil {
			if err = os.MkdirAll(p, fs.ModePerm); err != nil {
				return err
			}
		}
		fileData, err := os.Open(from)
		if err != nil {
			return err
		}
		defer fileData.Close()
		bufReader := bufio.NewReader(fileData)
		out, err := os.Create(to)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, bufReader)
	}
	return err
}

func CreateSoftLink(from, to string) error {
	// 如果to存在，则先删除to
	if exist, isDir := PathExistAndIsDir(to); exist {
		var err error = nil
		if isDir {
			err = os.RemoveAll(to)
		} else {
			err = os.Remove(to)
		}
		if err != nil {
			return err
		}
	}
	// 如果to父级别目录不存在，则先创建to的父级目录
	lastSepIndex := strings.LastIndex(to, "/")
	father := to[:lastSepIndex]
	if exist, _ := PathExistAndIsDir(to[:lastSepIndex]); !exist {
		err := os.MkdirAll(father, 0777)
		if err != nil {
			return err
		} else {
			tilogs.L().Infof("Successfully created directories %v", father)
		}
	}
	_, err := exec.Command("ln", "-s", from, to).Output()
	if err != nil {
		tilogs.L().Errorf("CreateSoftLink error: %v", err)
		return err
	}
	return nil
}

func PathExistAndIsDir(path string) (bool, bool) {
	s, err := os.Stat(path)
	if err == nil {
		return true, s.IsDir()
	}
	//isnotexist来判断，是不是不存在的错误
	if os.IsNotExist(err) {
		return false, false
	}
	return false, false
}
