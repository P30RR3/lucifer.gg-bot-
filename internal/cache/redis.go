package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"krypton.dev/internal/config"
)

// Client envuelve un *redis.Client con helpers de Krypton.
type Client struct {
	rdb  *redis.Client
	keys Keys
	log  *slog.Logger
	cfg  config.RedisConfig
}

// New construye un cliente Redis a partir de la configuración y verifica
// conectividad con un PING acotado por timeout.
func New(ctx context.Context, cfg config.RedisConfig, log *slog.Logger) (*Client, error) {
	if log == nil {
		log = slog.Default()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		TLSConfig:    &tls.Config{}, // <--- LÍNEA MÁGICA PARA UPSTASH
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("cache: ping redis en %s: %w", cfg.Addr, err)
	}

	log.Info("redis conectado",
		"addr", cfg.Addr,
		"db", cfg.DB,
		"pool_size", cfg.PoolSize,
	)

	return &Client{
		rdb:  rdb,
		keys: Keys{},
		log:  log.With("component", "cache"),
		cfg:  cfg,
	}, nil
}

func (c *Client) Raw() *redis.Client             { return c.rdb }
func (c *Client) Keys() Keys                     { return c.keys }
func (c *Client) Config() config.RedisConfig     { return c.cfg }

func (c *Client) Health(ctx context.Context) error {
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.rdb.Ping(probeCtx).Err()
}

func (c *Client) Close() error {
	c.log.Info("redis cerrando")
	return c.rdb.Close()
}

func (c *Client) GetString(ctx context.Context, key string) (string, bool, error) {
	v, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("cache: get %q: %w", key, err)
	}
	return v, true, nil
}

func (c *Client) SetString(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.cfg.DefaultTTL
	}
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

func (c *Client) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = c.cfg.DefaultTTL
	}
	ok, err := c.rdb.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("cache: setnx %q: %w", key, err)
	}
	return ok, nil
}

func (c *Client) GetOrSet(ctx context.Context, key string, ttl time.Duration, compute func(context.Context) (string, error)) (string, error) {
	if v, ok, err := c.GetString(ctx, key); err == nil && ok {
		return v, nil
	}
	v, err := compute(ctx)
	if err != nil {
		return "", err
	}
	if err := c.SetString(ctx, key, v, ttl); err != nil {
		c.log.Warn("get-or-set: fallo al guardar", "key", key, "err", err)
	}
	return v, nil
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache: del %v: %w", keys, err)
	}
	return nil
}

func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	n, err := c.rdb.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: exists: %w", err)
	}
	return n, nil
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: ttl %q: %w", key, err)
	}
	return d, nil
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("cache: expire %q: %w", key, err)
	}
	return nil
}

func (c *Client) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("cache: incr %q: %w", key, err)
	}
	return incr.Val(), nil
}

func (c *Client) IncrBy(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	pipe := c.rdb.TxPipeline()
	incr := pipe.IncrBy(ctx, key, delta)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("cache: incrby %q: %w", key, err)
	}
	return incr.Val(), nil
}

func (c *Client) ZAddScore(ctx context.Context, key, member string, score float64) error {
	if err := c.rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err(); err != nil {
		return fmt.Errorf("cache: zadd %q: %w", key, err)
	}
	return nil
}

func (c *Client) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	n, err := c.rdb.ZRemRangeByScore(ctx, key, strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64)).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: zremrangebyscore %q: %w", key, err)
	}
	return n, nil
}

func (c *Client) ZRangeTop(ctx context.Context, key string, n int64) ([]redis.Z, error) {
	if n <= 0 {
		n = 10
	}
	res, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, n-1).Result()
	if err != nil {
		return nil, fmt.Errorf("cache: zrevrange %q: %w", key, err)
	}
	return res, nil
}

func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	n, err := c.rdb.ZCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: zcard %q: %w", key, err)
	}
	return n, nil
}

func (c *Client) HSet(ctx context.Context, key string, fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	if err := c.rdb.HSet(ctx, key, args...).Err(); err != nil {
		return fmt.Errorf("cache: hset %q: %w", key, err)
	}
	return nil
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	m, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("cache: hgetall %q: %w", key, err)
	}
	return m, nil
}

func (c *Client) HIncrBy(ctx context.Context, key, field string, delta int64) (int64, error) {
	v, err := c.rdb.HIncrBy(ctx, key, field, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: hincrby %q/%q: %w", key, field, err)
	}
	return v, nil
}

func (c *Client) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("cache: marshal %q: %w", key, err)
	}
	return c.SetString(ctx, key, string(raw), ttl)
}

func (c *Client) GetJSON(ctx context.Context, key string, v any) (bool, error) {
	raw, ok, err := c.GetString(ctx, key)
	if err != nil || !ok {
		return ok, err
	}
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return false, fmt.Errorf("cache: unmarshal %q: %w", key, err)
	}
	return true, nil
}

func (c *Client) Lock(ctx context.Context, key, token string, ttl time.Duration) (release func(context.Context) error, err error) {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	ok, err := c.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("cache: lock %q: %w", key, err)
	}
	if !ok {
		return nil, ErrLockBusy
	}
	release = func(rctx context.Context) error {
		const script = `
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			else
				return 0
			end`
		if err := c.rdb.Eval(rctx, script, []string{key}, token).Err(); err != nil {
			return fmt.Errorf("cache: release lock %q: %w", key, err)
		}
		return nil
	}
	return release, nil
}

var ErrLockBusy = errors.New("cache: lock ocupado")