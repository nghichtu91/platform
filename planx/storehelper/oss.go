package storehelper

import (
	"github.com/nghichtu91/platform/share/planx/cloud_db/ossdb"
)

type StoreOSS struct {
	s *ossdb.OSS
}

func NewStoreOSS(endPoint, accessKey, secretKey, bucket, cloudDbRoot, format, seq string) IStore {
	newStore := ossdb.NewStoreOSS(endPoint, accessKey, secretKey, bucket, cloudDbRoot, format, seq)
	return &StoreOSS{
		s: newStore,
	}
}

func (s *StoreOSS) Open() error {
	return s.s.Open()
}

func (s *StoreOSS) Close() error {
	// TODO
	return nil
}

func (s *StoreOSS) Put(key string, val []byte, rh ReadHandler) error {
	return s.s.Put(key, val)
}

func (s *StoreOSS) Get(key string) ([]byte, error) {
	return s.s.Get(key)
}

func (s *StoreOSS) Del(key string) error {
	return s.s.Del(key)
}

func (s *StoreOSS) StoreKey(key string) string {
	// TODO 新DB接口未实现
	return ""
}

func (s *StoreOSS) RedisKey(key_in_store string) (string, bool) {
	// TODO 新DB接口未实现
	return "", false
}

func (s *StoreOSS) Clone() (IStore, error) {
	cs := &StoreOSS{}
	_cs, err := s.s.Clone()
	if err != nil {
		return nil, err
	}
	cs.s = _cs
	return cs, nil
}
