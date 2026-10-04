package redis_helper

import (
	"errors"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type redisKeysHander func(keys []string, values []string) error

type redisScanner struct {
	conn     redis.Conn
	last_idx string
	key      string

	hander redisKeysHander
}

func newScanner(conn redis.Conn, key string, hander redisKeysHander) *redisScanner {
	return &redisScanner{
		conn:     conn,
		last_idx: "",
		hander:   hander,
		key:      key,
	}
}

func (r *redisScanner) Start() {
	r.last_idx = "0"
}

func (r *redisScanner) Next() error {
	res, err := redis.Values(r.conn.Do("HSCAN", r.key, r.last_idx, "count", 100))
	if err != nil {
		return err
	}

	if len(res) < 2 {
		return errors.New("Scan Res Len less 2")
	}

	ins, ok := res[0].([]byte)
	if !ok {
		return errors.New("res[0] No []byte")
	}

	//logs.Trace("res[0] : %v", string(ins))

	keys, ok := res[1].([]interface{})
	if !ok {
		return errors.New("res[1] No []interface{}")
	}

	r.last_idx = string(ins)

	key_strs := make([]string, 0, len(keys))
	values_strs := make([]string, 0, len(keys))
	if len(keys)%2 != 0 {
		return errors.New("keys Len not 2")
	}

	for i := 0; i < len(keys); i++ {
		key := keys[i]
		k, ok := key.([]byte)
		if !ok {
			tilogs.L().Errorf("key Data %v Error", key)
			continue
		}
		//logs.Trace("res Key %d : %v", i, string(k))
		if i%2 == 0 {
			key_strs = append(key_strs, string(k))
		} else {
			values_strs = append(values_strs, string(k))
		}
	}

	return r.hander(key_strs, values_strs)
}

func (r *redisScanner) IsScanOver() bool {
	return r.last_idx == "0"
}

func HScan(conn redis.Conn, key string, f redisKeysHander) error {
	s := newScanner(conn, key, f)
	s.Start()
	for {
		err := s.Next()
		if err != nil {
			return err
		}
		if s.IsScanOver() {
			return nil
		}
	}
}

func (r *redisScanner) ScanNext() ([]string, error) {
	res, err := redis.Values(r.conn.Do("SCAN", r.last_idx, "match", r.key, "count", 500))
	if err != nil {
		return nil, err
	}

	if len(res) < 2 {
		return nil, errors.New("Scan Res Len less 2")
	}

	ins, ok := res[0].([]byte)
	if !ok {
		return nil, errors.New("res[0] No []byte")
	}
	r.last_idx = string(ins)

	keys, ok := res[1].([]interface{})
	if !ok {
		return nil, errors.New("res[1] No []interface{}")
	}
	key_strs := make([]string, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		k, ok := key.([]byte)
		if !ok {
			tilogs.L().Errorf("key Data %v Error", key)
			continue
		}
		key_strs = append(key_strs, string(k))
	}

	return key_strs, nil
}

func Scan(conn redis.Conn, key string, f redisKeysHander) error {
	s := newScanner(conn, key, f)
	s.Start()
	ret := make([]string, 0)
	for {
		keys, err := s.ScanNext()
		if err != nil {
			return err
		}
		ret = append(ret, keys...)
		if s.IsScanOver() {
			s.hander(ret, nil)
			return nil
		}
	}
}
