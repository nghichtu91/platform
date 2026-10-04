package file_hash_cache

import (
	"context"
	"sync"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	_ = iota
	CCmdSave
	CCmdUpdateFiles
	CCmdGetChangedInfo
)

// CCache 支持并发的缓存，C代表使用channel
type CCache struct {
	*Cache

	recChan chan *CCmd

	quit chan struct{}

	once sync.Once
}

type CCmd struct {
	cmdTyp int

	filenames []string

	retChan chan *CRet
}

type CRet struct {
	changed bool
	err     error
	*ChangeInfo
}

func (cc *CCmd) Reply(cr *CRet) {
	if cc.retChan != nil {
		ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
		defer cancel()

		select {
		case cc.retChan <- cr:
		case <-ctx.Done():
			tilogs.L().Errorf("cmd %v send return timeout", cc)
		}
	}
}

func NewCCache(path string) *CCache {
	return &CCache{
		Cache:   NewCache(path),
		recChan: make(chan *CCmd, 1024),
		quit:    make(chan struct{}, 1),
	}
}

func (cc *CCache) Start() error {
	if err := cc.Cache.Start(); err != nil {
		return err
	}

	go cc.loop()

	return nil
}

func (cc *CCache) loop() {
	for {
		select {
		case <-cc.quit:
			cc.Cache.Save()
			return
		case cmd := <-cc.recChan:
			func(cmd *CCmd) {
				ret := &CRet{}

				switch cmd.cmdTyp {
				case CCmdSave:
					ret.err = cc.Cache.Save()
				case CCmdUpdateFiles:
					ret.changed, ret.err = cc.Cache.UpdateFiles(cmd.filenames...)
				case CCmdGetChangedInfo:
					ret.ChangeInfo = cc.Cache.GetChangedInfo()
				default:
					tilogs.L().Errorf("CCache loop receive unknown cmd %v", cmd)
					return
				}

				cmd.Reply(ret)
			}(cmd)
		}
	}
}

func (cc *CCache) Close() error {
	cc.once.Do(func() {
		close(cc.quit)
	})

	return nil
}

func (cc *CCache) Save() error {
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()

	ret := make(chan *CRet, 1)

	select {
	case cc.recChan <- &CCmd{
		cmdTyp:  CCmdSave,
		retChan: ret,
	}:
	case <-ctx.Done():
		return ErrCCacheCmdSendTimeout
	}

	select {
	case r := <-ret:
		return r.err
	case <-ctx.Done():
		return ErrCCacheCmdRetTimeout
	}
}

func (cc *CCache) UpdateFiles(filenames ...string) (changed bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()

	ret := make(chan *CRet, 1)

	select {
	case cc.recChan <- &CCmd{
		cmdTyp:    CCmdUpdateFiles,
		filenames: filenames,
		retChan:   ret,
	}:
	case <-ctx.Done():
		return false, ErrCCacheCmdSendTimeout
	}

	select {
	case r := <-ret:
		return r.changed, r.err
	case <-ctx.Done():
		return false, ErrCCacheCmdRetTimeout
	}
}

func (cc *CCache) GetChangedInfo() *ChangeInfo {
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()

	ret := make(chan *CRet, 1)

	select {
	case cc.recChan <- &CCmd{
		cmdTyp:  CCmdGetChangedInfo,
		retChan: ret,
	}:
	case <-ctx.Done():
		tilogs.L().Errorf("CCache cmd GetChangedInfo failed, send cmd time out")
		return nil
	}

	select {
	case r := <-ret:
		return r.ChangeInfo
	case <-ctx.Done():
		tilogs.L().Errorf("CCache cmd GetChangedInfo failed, receive ret time out")
		return nil
	}
}
