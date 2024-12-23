package gcron

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Option struct {
	F func(o *Options)
}

type Options struct {
	RedisUrl   string
	Name       string
	Signals    []os.Signal
	LockExpire time.Duration
}

func (o *Options) Apply(opts []Option) {
	for _, op := range opts {
		op.F(o)
	}
}

func WithSignal(sig ...os.Signal) Option {
	return Option{F: func(o *Options) {
		o.Signals = sig
	}}
}

func WithName(name string) Option {
	return Option{func(o *Options) {
		o.Name = name
	}}
}

func WithRedisUrl(redisUrl string) Option {
	return Option{F: func(o *Options) {
		o.RedisUrl = redisUrl
	}}
}

func WithLockExpireSeconds(d time.Duration) Option {
	return Option{F: func(o *Options) {
		o.LockExpire = d
	}}
}

type RedisOptions struct {
	Scheme   string
	Addr     string
	Host     string
	Port     int
	Password string
	DB       int
}

func ParsedRedisURL(uri string) (*RedisOptions, error) {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse Redis URL: %v", err))
	}
	scheme := parsedURL.Scheme
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	password, _ := parsedURL.User.Password()   // 获取密码
	db, er := strconv.Atoi(parsedURL.Path[1:]) // 去掉前导斜杠并转换为整数
	if er != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse DB index: %v", err))
	}

	_port := 6379
	if port != "" {
		portInt, _er := strconv.Atoi(port)

		if _er != nil {
			return nil, errors.New(fmt.Sprintf("Failed to parse Port index: %v", _er))
		}
		_port = portInt
	}

	addr := fmt.Sprintf("%v:%v", host, port)

	return &RedisOptions{
		Scheme:   scheme,
		Addr:     addr,
		Host:     host,
		Port:     _port,
		Password: password,
		DB:       db,
	}, nil
}

func NewRedisClient(uri string, poolSize int) *redis.Client {
	opt, er := ParsedRedisURL(uri)
	if er != nil {
		log.Fatalf("%v%v%v", "\033[31m", er.Error(), "\033[0m")
	}

	rcli := redis.NewClient(&redis.Options{
		Addr:     opt.Addr,
		Password: opt.Password,
		DB:       opt.DB,
		PoolSize: poolSize,
		//IdleTimeout:        -1,
		//IdleCheckFrequency: -1,
	})

	return rcli
}
