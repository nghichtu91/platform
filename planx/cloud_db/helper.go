package cloud_db

import (
	"errors"
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	MaxListLength = 1000 // 最大的单次扫描长度
	EmptyListIdx  = ""   // 空代表迭代结束
)

var (
	ErrNilDB = errors.New("cloud db is nil")
)

type ObjectFilter func(string) bool

//ListAllObjectWithBucket 获取指定bucket所有的文件
//filters: 如果不为空, 则需要对应的文件满足所有的filter才是有效的
func ListAllObjectWithBucket(db CloudDb, bucket, cloudDbRoot, prefix string, filters ...ObjectFilter) (all []string, err error) {
	if db == nil {
		return nil, ErrNilDB
	}
	lastIdx := EmptyListIdx

	// 注意:
	/*
		1. 华为云/阿里云返回的 lastIdx 会在最后一次有效迭代后返回空, 且返回值等同于文件名
			- objects,  lastIndex
			- [0,  100], "100"
			- [101,200], "200"
			- [201,260], ""
		2. 谷歌云返回的lastIdx会在最后一次有效迭代后返回空, 且返回值不等同于文件名
			- objects,    lastIndex
			- [0,  100], "XXXCCCAAA"
			- [101,200], "GGGADADSD"
			- [201,260], ""
	*/

	for {
		var objects []string
		oldIdx := lastIdx
		objects, lastIdx, err = db.ListObjectWithBucket(bucket, lastIdx, MaxListLength, cloudDbRoot, prefix)
		if err != nil {
			tilogs.L().Errorf(
				"get all object error:%v. Bucket:%v,DBRoot:%v,Prefix:%v,LastIdx:%v",
				err,
				bucket,
				cloudDbRoot,
				prefix,
				oldIdx,
			)
			return
		}
		if len(filters) > 0 {
			// 只选取满足所有条件的文件
			var valid = objects[:0]
		next:
			for _, obj := range objects {
				for _, filter := range filters {
					// 有一个不满足的就看下一个文件
					if !filter(obj) {
						continue next
					}
				}
				valid = append(valid, obj)
			}
			objects = valid
		}

		all = append(all, objects...)
		if lastIdx == EmptyListIdx {
			break
		}
	}
	return
}

// MoveObject 在同等bucket下移动文件, 拥有相同的cloudDbRoot
// 涉及到跨bucket移动的情况, 请使用 GetWithBucket + PutWithBucket
func MoveObject(db CloudDb, bucket, cloudDbRoot, src, dst string) error {
	if db == nil {
		return ErrNilDB
	}
	// 如果是同一个文件, 则不需要移动
	if src == dst {
		return nil
	}
	// 先尝试复制
	cpErr := db.CopyWithBucket(bucket, cloudDbRoot, src, dst)
	if cpErr != nil {
		return fmt.Errorf(
			"copy object %s[%s/%s] to %s[%s/%s] err %v",
			bucket, cloudDbRoot, src,
			bucket, cloudDbRoot, dst,
			cpErr,
		)
	}
	// 如果复制成功, 则删除原文件
	delErr := db.DelWithBucket(bucket, cloudDbRoot, src)
	if delErr != nil {
		return fmt.Errorf(
			"delete object %s[%s/%s] err %v",
			bucket, cloudDbRoot, src,
			delErr,
		)
	}
	return nil
}
