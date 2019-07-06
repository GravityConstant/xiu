package util

import (
	"bytes"
	"encoding/gob"
	// "errors"
	"time"

	"github.com/go-redis/redis"
)

var (
	addr     string
	password string

	client *redis.Client
)

func InitCache() {
	defer func() {
		recover()
	}()

	addr = PbxConfigInstance.Get("redis::addr")
	password = PbxConfigInstance.Get("redis::password")
	Debug("util/cache.go", "get init params", addr, password)

	client = NewRedisClient(10)
}

func NewRedisClient(poolSize int) *redis.Client {
	client = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     poolSize,
		PoolTimeout:  30 * time.Second,
	})
	// 一flushDB，所有数据都被清空了啊！！！
	if err := client.FlushDB().Err(); err != nil {
		Error("util/cache.go", "connect to redis", err)
		panic(err)
	}
	return client
}

func CheckRedis() (b bool) {
	defer func() {
		if r := recover(); r != nil {
			b = false
		}
	}()
	// 检查redis状态
	if client == nil {
		client = NewRedisClient(10)
	}
	if pong, err := client.Ping().Result(); err == nil {
		if pong == "PONG" {
			b = true
		}
	}
	return b
}

// SetCache
func SetCache(key string, value interface{}, timeout int) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// 检查redis状态
	if client == nil {
		client = NewRedisClient(10)
	}
	// 数据转成二进制
	data, err := Encode(value)
	if err != nil {
		Error("util/cache.go", "Encode失败，key="+key, err)
		return err
	}

	// colorlog.Info("encode data: %v", data)
	timeouts := time.Duration(timeout) * time.Second
	err = client.Set(key, data, timeouts).Err()
	if err != nil {
		Error("util/cache.go", "SetCache失败，key="+key, err)
		return err
	}

	return err
}

func GetCache(key string, to interface{}) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// 检查redis状态
	if client == nil {
		client = NewRedisClient(10)
	}

	data, err := client.Get(key).Bytes()
	if err != nil {
		Error("util/cache.go", "GetCache失败，key="+key, err.Error())
		return err
	}

	err = Decode(data, to)
	if err != nil {
		Error("util/cache.go", "Decode失败，key="+key, err)
	}

	return err
}

// DelCache
func DelCache(key string) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// 检查redis状态
	if client == nil {
		client = NewRedisClient(10)
	}

	_, err = client.Del(key).Result()
	if err != nil {
		Error("util/cache.go", "Cache删除失败，key="+key, err)
		return err
	} else {
		return nil
	}
}

// Encode
// 用gob进行数据编码
func Encode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode
// 用gob进行数据解码
func Decode(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}
