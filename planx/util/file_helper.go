package util

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func FileRename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func FileSize(name string) (int64, error) {
	fileInfo, err := os.Stat(name)
	if err != nil {
		log.Println("FileSize os.Stat err", err.Error())
		return 0, err
	}
	return fileInfo.Size(), nil //获取size
}

// 删除文件或空目录
func DelFile(name string) error {
	return os.Remove(name)
}

func DelAllFile(path string) error {
	return os.RemoveAll(path)
}

// 删除指定后缀文件
func RemoveFileWithSuffix(dirPth, suffix string) error {
	suffix = strings.ToUpper(suffix)
	return filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			os.Remove(filename)
		}
		return nil
	})
}

// WalkWithAllFiles 遍历目录，获取所有的文件名列表
// useAbs 是否替换路径为绝对路径
// onlyName 文件名是否包括路径
// skipDir 是否跳过目录
func WalkWithAllFiles(dir string, useAbs, onlyName, skipDir bool) ([]string, error) {
	var (
		root string
		err  error
	)
	fn := make([]string, 0, 1024)

	// 绝对路径
	if useAbs {
		root, err = filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
	} else {
		root = dir
	}

	err = filepath.Walk(root,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// 跳过目录
			if skipDir && info.IsDir() {
				return nil
			}

			// 文件名是否包括路径
			var name string
			if onlyName {
				name = info.Name()
			} else {
				name = path
			}

			fn = append(fn, name)
			return nil
		})

	return fn, err
}

// 过滤elb的地址
func NeedFilterElbAddr(remoteAddr string, filterElbAddrs []string) bool {
	for _, ea := range filterElbAddrs {
		if ea != "" && strings.HasPrefix(remoteAddr, ea) {
			return true
		}
	}
	return false
}

// 复制文件到指定目录
func Copy(sourceFilePath string, dstFilePath string) error {
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return err
	}
	dstFile, err := os.Create(dstFilePath)
	defer func() {
		if dstFile != nil {
			dstFile.Close()
		}
	}()
	if err != nil {
		return err
	}

	// 拷贝源文件至目标文件。
	_, err = io.Copy(dstFile, sourceFile)
	defer func() {
		if sourceFile != nil {
			sourceFile.Close()
		}
	}()
	if err != nil {
		return err
	}
	dstFile.Sync()

	return nil
}

// 将含有指定后缀的文件复制到某目录下
func CopyBySuffix(sourceDir string, dstDir string, suffix string) error {
	if dstDir[len(dstDir)-1:] != "/" {
		dstDir += "/"
	}
	var err error
	filepath.Walk(sourceDir, func(path string, fi os.FileInfo, _err error) error {
		if !fi.IsDir() && strings.HasSuffix(fi.Name(), suffix) {
			err = Copy(path, dstDir+fi.Name())
		}
		return nil
	})
	return err
}
