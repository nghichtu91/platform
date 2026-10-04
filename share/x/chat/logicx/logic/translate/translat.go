package translate

import (
	"context"
	"fmt"
	"github.com/nghichtu91/platform/share/x/chat/web/logic/chatSystem"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/translate"
	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/redispool"
	chat_common "github.com/nghichtu91/platform/share/planx/servers/chat"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
	c "github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"golang.org/x/text/language"
)

const (
	cacheTTL                = timeutil.WeekSec * 12
	lruPolicy               = "volatile-lru"
	defTimeOutMS            = 3000
	translationCntKey       = "TaiYouXiTranslationCnt"
	translateTimeOutKey     = "TranslateTimeOut"  // 翻译超时ms
	translateSwitchKey      = "TranslateOff"      // 翻译是否关闭
	translateCacheSwitchKey = "TranslateCacheOff" // 翻译缓存是否关闭
)

var (
	cache     map[string]redispool.IPool
	cacheLock sync.RWMutex
)

func InitCache(addr, language, db, pwd, lru []string) error {
	l := len(addr)
	ok := len(addr) == len(language) && len(language) == len(db) && len(db) == len(pwd) && len(pwd) == len(lru)
	if !ok {
		return fmt.Errorf("translate cfg fmt err")
	}

	cache = make(map[string]redispool.IPool, l)
	for i := 0; i < l; i++ {
		pool := chat_common.NewRedisPool(fmt.Sprintf("chattranslate-%s",
			language[i]), addr[i], pwd[i], db[i], addr[i], &config.EtcdConf, redispool.DefaultMaxCapacity)
		cacheLock.Lock()
		cache[language[i]] = pool
		cacheLock.Unlock()

		err := func() error {
			var err error
			conn := pool.Get()
			if conn.IsNil() {
				return fmt.Errorf("translate conn nil %s", language[i])
			}
			defer conn.Close()

			_, err = conn.Do(metrics.GetDBStatPrefix("chat", "maxmemory-policy", "CONFIG"),
				"CONFIG", "SET", "maxmemory-policy", lruPolicy)
			if err != nil {
				return err
			}
			_, err = conn.Do(metrics.GetDBStatPrefix("chat", "maxmemory", "CONFIG"),
				"CONFIG", "SET", "maxmemory", lru[i])
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	log.L().Infof("init translate cache %v", language)
	return nil
}

func getTargetCache(targetLanguage string) (conn redispool.RedisPoolConn) {
	cacheLock.RLock()
	defer cacheLock.RUnlock()
	if pool, ok := cache[targetLanguage]; ok {
		return pool.Get()
	}
	return
}

func Translate(targetLanguage, text string) (cnt int64, result string, err error) {
	// switch off
	if !translateOn() {
		return cnt, result, err
	}

	// dont translate
	if text == translationCntKey {
		return cnt, result, fmt.Errorf("oops, text is %s", translationCntKey)
	}

	// get target cache
	conn := getTargetCache(targetLanguage)
	if conn.IsNil() {
		return cnt, result, fmt.Errorf("target cache nil, %s", targetLanguage)
	}
	defer conn.Close()

	// try get from catch
	if translateCacheOn() {
		result = translationFromCache(conn, text)
		if result != "" {
			log.L().Debugf("translate from cache %s %s", text, result)
			return cnt, result, nil
		}
	}

	// try api
	result, err = translateText(targetLanguage, text)
	if err != nil {
		return cnt, result, err
	}
	cnt, err = addTranslationCnt(conn)
	if err != nil {
		log.L().Warnf("translate count add fail %v", err)
	}
	if result == "" {
		return cnt, result, fmt.Errorf("translate result empty, %s %s", targetLanguage, text)
	}

	// result put in catch
	if translateCacheOn() {
		err = translationToCache(conn, text, result)
		if err != nil {
			log.L().Warnf("translate result put in cache fail %v", err)
		}
	}
	log.L().Debugf("translate from api %s %s", text, result)
	return cnt, result, nil
}

func translationFromCache(conn redispool.RedisPoolConn, text string) string {
	data, err := redis.String(conn.Do(metrics.GetDBStatPrefix("chat", "translationCache", "GET"), "GET", text))
	if err != nil {
		log.L().Warnf("translate from cache %v", err)
	}
	return data
}

func translationToCache(conn redispool.RedisPoolConn, text, result string) error {
	_, err := conn.Do(metrics.GetDBStatPrefix("chat", "translationCache", "SETEX"), "SETEX", text, cacheTTL, result)
	if err != nil {
		return err
	}
	return nil
}

func addTranslationCnt(conn redispool.RedisPoolConn) (int64, error) {
	return redis.Int64(conn.Do(metrics.GetDBStatPrefix("chat", "translationCache", "INCR"), "INCR", translationCntKey))
}

func translateOn() bool {
	v := c.GetChatConfigV(chatSystem.Common, translateSwitchKey)
	if v == 0 { //默认开
		return true
	}
	return false
}

func translateCacheOn() bool {
	v := c.GetChatConfigV(chatSystem.Common, translateCacheSwitchKey)
	if v == 0 { //默认开
		return true
	}
	return false
}

func translateTimeOut() int {
	v := c.GetChatConfigV(chatSystem.Common, translateTimeOutKey)
	if v <= 0 {
		return defTimeOutMS
	}
	return v
}

// https://cloud.google.com/translate/docs/languages
func tag(t string) string {
	if t == "zh-CN" || t == "zh-TW" {
		return t
	}
	ss := strings.Split(t, "-")
	if len(ss) == 0 {
		return t
	}
	return ss[0]
}

// translateText Google translate API
func translateText(targetLanguage, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(translateTimeOut()))
	defer cancel()

	lang, err := language.Parse(tag(targetLanguage))
	if err != nil {
		return "", fmt.Errorf("language.Parse: %v", err)
	}
	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer func(client *translate.Client) {
		_ = client.Close()
	}(client)

	resp, err := client.Translate(ctx, []string{text}, lang, nil)
	if err != nil {
		return "", fmt.Errorf("translate: %v", err)
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("translate returned empty response to text: %s", text)
	}
	return resp[0].Text, nil
}
