package ratelimit

import (
	"context"
	"database/sql"
	"eth2-exporter/db"
	"eth2-exporter/metrics"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type TimeWindow string

const (
	SecondTimeWindow = "second"
	HourTimeWindow   = "hour"
	MonthTimeWindow  = "month"
)

const (
	HeaderRateLimitLimit       = "X-RateLimit-Limit"        // the rate limit ceiling that is applicable for the current request
	HeaderRateLimitRemaining   = "X-RateLimit-Remaining"    // the number of requests left for the current rate-limit window
	HeaderRateLimitReset       = "X-RateLimit-Reset"        // the number of seconds until the quota resets
	HeaderRetryAfter           = "Retry-After"              // the number of seconds until the quota resets, same as HeaderRateLimitReset, RFC 7231, 7.1.3
	HeaderRateLimitLimitSecond = "X-RateLimit-Limit-Second" // the rate limit ceiling that is applicable for the current user
	HeaderRateLimitLimitHour   = "X-RateLimit-Limit-Hour"   // the rate limit ceiling that is applicable for the current user
	HeaderRateLimitLimitMonth  = "X-RateLimit-Limit-Month"  // the rate limit ceiling that is applicable for the current user

	DefaultRateLimitSecond = 2   // RateLimit per second if no ratelimits are set in database
	DefaultRateLimitHour   = 500 // RateLimit per second if no ratelimits are set in database
	DefaultRateLimitMonth  = 0   // RateLimit per second if no ratelimits are set in database

	FallbackRateLimitSecond = 20 // RateLimit per second for when redis is offline
	FallbackRateLimitBurst  = 20 // RateLimit burst for when redis is offline

	defaultBucket = "default"

	statsTruncateDuration = time.Hour * 1 // ratelimit-stats are truncated to this duration
)

var NoKeyRateLimit = &RateLimit{
	Second: DefaultRateLimitSecond,
	Hour:   DefaultRateLimitHour,
	Month:  DefaultRateLimitMonth,
}

var FreeRatelimit = NoKeyRateLimit

var redisClient *redis.Client
var redisIsHealthy atomic.Bool

var lastRateLimitUpdateKeys = time.Unix(0, 0)       // guarded by lastRateLimitUpdateMu
var lastRateLimitUpdateRateLimits = time.Unix(0, 0) // guarded by lastRateLimitUpdateMu
var lastRateLimitUpdateMu = &sync.Mutex{}

var fallbackRateLimiter = NewFallbackRateLimiter() // if redis is offline, use this rate limiter

var initializedWg = &sync.WaitGroup{} // wait for everything to be initialized before serving requests

var rateLimitsMu = &sync.RWMutex{}
var rateLimits = map[string]*RateLimit{}        // guarded by rateLimitsMu
var rateLimitsByUserId = map[int64]*RateLimit{} // guarded by rateLimitsMu
var userIdByApiKey = map[string]int64{}         // guarded by rateLimitsMu

var weightsMu = &sync.RWMutex{}
var weights = map[string]int64{}  // guarded by weightsMu
var buckets = map[string]string{} // guarded by weightsMu

var pathPrefix = "" // only requests with this prefix will be ratelimited

var logger = logrus.StandardLogger().WithField("module", "ratelimit")

type dbEntry struct {
	Date   time.Time
	ApiKey string
	Path   string
	Count  int64
}

type RateLimit struct {
	Second int64
	Hour   int64
	Month  int64
}

type RateLimitResult struct {
	Time          time.Time
	Weight        int64
	Route         string
	IP            string
	Key           string
	IsValidKey    bool
	UserId        int64
	RedisKeys     []RedisKey
	RedisStatsKey string
	RateLimit     *RateLimit
	Limit         int64
	Remaining     int64
	Reset         int64
	Bucket        string
	Window        TimeWindow
}

type RedisKey struct {
	Key      string
	ExpireAt time.Time
}

type responseWriterDelegator struct {
	http.ResponseWriter
	written     int64
	status      int
	wroteHeader bool
}

func (r *responseWriterDelegator) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}

func (r *responseWriterDelegator) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseWriterDelegator) Status() int {
	return r.status
}

var DefaultRequestCollector = func(req *http.Request) bool {
	if req.URL == nil || !strings.HasPrefix(req.URL.Path, "/api") {
		return false
	}
	return true
}

var requestSelector func(req *http.Request) bool

// Init initializes the RateLimiting middleware, the rateLimiting middleware will not work without calling Init first. The second parameter is a function the will get called on every request, it will only apply ratelimiting to requests when this func returns true.
func Init(redisAddress string, requestSelectorOpt func(req *http.Request) bool) {
	redisClient = redis.NewClient(&redis.Options{
		Addr:        redisAddress,
		ReadTimeout: time.Second * 3,
	})

	requestSelector = requestSelectorOpt

	initializedWg.Add(3)

	go func() {
		firstRun := true
		for {
			err := updateWeights(firstRun)
			if err != nil {
				logger.WithError(err).Errorf("error updating weights")
				time.Sleep(time.Second * 2)
				continue
			}
			if firstRun {
				initializedWg.Done()
				firstRun = false
			}
			time.Sleep(time.Second * 10)
		}
	}()
	go func() {
		firstRun := true

		for {
			err := updateRateLimits()
			if err != nil {
				logger.WithError(err).Errorf("error updating ratelimits")
				time.Sleep(time.Second * 2)
				continue
			}
			if firstRun {
				initializedWg.Done()
				firstRun = false
			}
			time.Sleep(time.Second * 10)
		}
	}()
	go func() {
		firstRun := true
		for {
			err := updateRedisStatus()
			if err != nil {
				logger.WithError(err).Errorf("error checking redis")
				time.Sleep(time.Second * 1)
				continue
			}
			if firstRun {
				initializedWg.Done()
				firstRun = false
			}
			time.Sleep(time.Second * 1)
		}
	}()
	go func() {
		for {
			err := updateStats()
			if err != nil {
				logger.WithError(err).Errorf("error updating stats")
			}
			time.Sleep(time.Second * 10)
		}
	}()

	initializedWg.Wait()
}

// HttpMiddleware returns an http.Handler that can be used as middleware to RateLimit requests. If redis is offline, it will use a fallback rate limiter.
func HttpMiddleware(next http.Handler) http.Handler {
	initializedWg.Wait()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestSelector(r) {
			next.ServeHTTP(w, r)
			return
		}

		if !redisIsHealthy.Load() {
			fallbackRateLimiter.Handle(w, r, next.ServeHTTP)
			return
		}

		rl, err := rateLimitRequest(r)
		if err != nil {
			// just serve the request if there is a problem with getting the rate limit
			logger.WithFields(logrus.Fields{"error": err}).Errorf("error getting rate limit")
			next.ServeHTTP(w, r)
			return
		}

		// logrus.WithFields(logrus.Fields{"route": rl.Route, "key": rl.Key, "limit": rl.Limit, "remaining": rl.Remaining, "reset": rl.Reset, "window": rl.Window, "validKey": rl.IsValidKey}).Infof("rateLimiting")

		w.Header().Set(HeaderRateLimitLimit, strconv.FormatInt(rl.Limit, 10))
		w.Header().Set(HeaderRateLimitRemaining, strconv.FormatInt(rl.Remaining, 10))
		w.Header().Set(HeaderRateLimitReset, strconv.FormatInt(rl.Reset, 10))

		if rl.RateLimit.Second > 0 {
			w.Header().Set(HeaderRateLimitLimitSecond, strconv.FormatInt(rl.RateLimit.Second, 10))
		}
		if rl.RateLimit.Hour > 0 {
			w.Header().Set(HeaderRateLimitLimitHour, strconv.FormatInt(rl.RateLimit.Hour, 10))
		}
		if rl.RateLimit.Month > 0 {
			w.Header().Set(HeaderRateLimitLimitMonth, strconv.FormatInt(rl.RateLimit.Month, 10))
		}

		if rl.Weight > rl.Remaining {
			w.Header().Set(HeaderRetryAfter, strconv.FormatInt(rl.Reset, 10))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			err = postRateLimit(rl, http.StatusTooManyRequests)
			if err != nil {
				logger.WithFields(logrus.Fields{"error": err}).Errorf("error calling postRateLimit")
			}
			return
		}

		d := &responseWriterDelegator{ResponseWriter: w}
		next.ServeHTTP(d, r)
		err = postRateLimit(rl, d.Status())
		if err != nil {
			logger.WithFields(logrus.Fields{"error": err}).Errorf("error calling postRateLimit")
		}
	})
}

// updateWeights gets the weights and buckets from postgres and updates the weights and buckets maps.
func updateWeights(firstRun bool) error {
	start := time.Now()
	defer func() {
		metrics.TaskDuration.WithLabelValues("ratelimit_updateWeights").Observe(time.Since(start).Seconds())
	}()

	dbWeights := []struct {
		Endpoint  string    `db:"endpoint"`
		Weight    int64     `db:"weight"`
		Bucket    string    `db:"bucket"`
		ValidFrom time.Time `db:"valid_from"`
	}{}
	err := db.WriterDb.Select(&dbWeights, "SELECT DISTINCT ON (endpoint) endpoint, bucket, weight, valid_from FROM api_weights WHERE valid_from <= NOW() ORDER BY endpoint, valid_from DESC")
	if err != nil {
		return err
	}
	weightsMu.Lock()
	defer weightsMu.Unlock()
	oldWeights := weights
	oldBuckets := buckets
	weights = make(map[string]int64, len(dbWeights))
	for _, w := range dbWeights {
		weights[w.Endpoint] = w.Weight
		if !firstRun && oldWeights[w.Endpoint] != weights[w.Endpoint] {
			logger.WithFields(logrus.Fields{"endpoint": w.Endpoint, "weight": w.Weight, "oldWeight": oldWeights[w.Endpoint]}).Infof("weight changed")
		}
		buckets[w.Endpoint] = strings.ReplaceAll(w.Bucket, ":", "_")
		if buckets[w.Endpoint] == "" {
			buckets[w.Endpoint] = defaultBucket
		}
		if !firstRun && oldBuckets[w.Endpoint] != buckets[w.Endpoint] {
			logger.WithFields(logrus.Fields{"endpoint": w.Endpoint, "bucket": w.Weight, "oldBucket": oldBuckets[w.Endpoint]}).Infof("bucket changed")
		}
	}
	return nil
}

// updateRedisStatus checks if redis is healthy and updates redisIsHealthy accordingly.
func updateRedisStatus() error {
	oldStatus := redisIsHealthy.Load()
	newStatus := true
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*1))
	defer cancel()
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		logger.WithError(err).Errorf("error pinging redis")
		newStatus = false
	}
	if oldStatus != newStatus {
		logger.WithFields(logrus.Fields{"oldStatus": oldStatus, "newStatus": newStatus}).Infof("redis status changed")
	}
	redisIsHealthy.Store(newStatus)
	return nil
}

// updateStats scans redis for ratelimit:stats:* keys and inserts them into postgres, if the key's truncated date is older than specified stats-truncation it will also delete the key in redis.
func updateStats() error {
	start := time.Now()
	defer func() {
		metrics.TaskDuration.WithLabelValues("ratelimit_updateStats").Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()

	var err error
	startTruncated := start.Truncate(statsTruncateDuration)

	allKeys := []string{}
	cursor := uint64(0)

	for {
		cmd := redisClient.Scan(ctx, cursor, "ratelimit:stats:*:*:*", 1000)
		if cmd.Err() != nil {
			return cmd.Err()
		}
		keys, nextCursor, err := cmd.Result()
		if err != nil {
			return err
		}
		cursor = nextCursor
		allKeys = append(allKeys, keys...)
		if cursor == 0 {
			break
		}
	}

	batchSize := 10000
	for i := 0; i <= len(allKeys); i += batchSize {
		start := i
		end := i + batchSize
		if end > len(allKeys) {
			end = len(allKeys)
		}

		if start == end {
			break
		}

		keysToDelete := []string{}
		keys := allKeys[start:end]
		entries := make([]dbEntry, len(keys))
		for i, k := range keys {
			ks := strings.Split(k, ":")
			if len(ks) != 5 {
				return fmt.Errorf("error parsing key %s: split-len != 5", k)
			}
			dateString := ks[2]
			date, err := time.Parse("2006-01-02-15", dateString)
			if err != nil {
				return fmt.Errorf("error parsing date in key %s: %v", k, err)
			}
			dateTruncated := date.Truncate(statsTruncateDuration)
			if dateTruncated.Before(startTruncated) {
				keysToDelete = append(keysToDelete, k)
			}
			entries[i] = dbEntry{
				Date:   dateTruncated,
				ApiKey: ks[3],
				Path:   ks[4],
			}
		}

		mgetSize := 500
		for j := 0; j <= len(keys); j += mgetSize {
			mgetStart := j
			mgetEnd := j + mgetSize
			if mgetEnd > len(keys) {
				mgetEnd = len(keys)
			}
			mgetRes, err := redisClient.MGet(ctx, keys[mgetStart:mgetEnd]...).Result()
			if err != nil {
				return fmt.Errorf("error getting stats-count from redis (%v-%v/%v): %w", mgetStart, mgetEnd, len(keys), err)
			}
			for k, v := range mgetRes {
				vStr, ok := v.(string)
				if !ok {
					return fmt.Errorf("error parsing stats-count from redis: value is not string: %v: %v: %w", k, v, err)
				}
				entries[mgetStart+k].Count, err = strconv.ParseInt(vStr, 10, 64)
				if err != nil {
					return fmt.Errorf("error parsing stats-count from redis: value is not int64: %v: %v: %w", k, v, err)
				}
			}
		}

		err = updateStatsEntries(entries)
		if err != nil {
			return fmt.Errorf("error updating stats entries: %w", err)
		}

		if len(keysToDelete) > 0 {
			delSize := 500
			for j := 0; j <= len(keys); j += delSize {
				delStart := j
				delEnd := j + delSize
				if delEnd > len(keysToDelete) {
					delEnd = len(keysToDelete)
				}
				_, err = redisClient.Del(ctx, keysToDelete[delStart:delEnd]...).Result()
				if err != nil {
					logger.Errorf("error deleting stats-keys from redis: %v", err)
				}
			}
		}
	}

	return nil
}

func updateStatsEntries(entries []dbEntry) error {
	tx, err := db.WriterDb.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	numArgs := 4
	batchSize := 65535 / numArgs // max 65535 params per batch, since postgres uses int16 for binding input params
	valueArgs := make([]interface{}, 0, batchSize*numArgs)
	valueStrings := make([]string, 0, batchSize)
	valueStringArr := make([]string, numArgs)
	batchIdx, allIdx := 0, 0
	for _, entry := range entries {
		for u := 0; u < numArgs; u++ {
			valueStringArr[u] = fmt.Sprintf("$%d", batchIdx*numArgs+1+u)
		}

		valueStrings = append(valueStrings, "("+strings.Join(valueStringArr, ",")+")")
		valueArgs = append(valueArgs, entry.Date)
		valueArgs = append(valueArgs, entry.ApiKey)
		valueArgs = append(valueArgs, entry.Path)
		valueArgs = append(valueArgs, entry.Count)

		// logger.WithFields(logrus.Fields{"count": entry.Count, "apikey": entry.ApiKey, "path": entry.Path, "date": entry.Date}).Infof("inserting stats entry %v/%v", allIdx+1, len(entries))

		batchIdx++
		allIdx++

		if batchIdx >= batchSize || allIdx >= len(entries) {
			stmt := fmt.Sprintf(`INSERT INTO api_statistics (ts, apikey, call, count) VALUES %s ON CONFLICT (ts, apikey, call) DO UPDATE SET count = EXCLUDED.count`, strings.Join(valueStrings, ","))
			_, err := tx.Exec(stmt, valueArgs...)
			if err != nil {
				return err
			}
			batchIdx = 0
			valueArgs = valueArgs[:0]
			valueStrings = valueStrings[:0]
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// updateRateLimits updates the maps rateLimits, rateLimitsByUserId and userIdByApiKey with data from postgres-tables api_keys and api_ratelimits.
func updateRateLimits() error {
	start := time.Now()
	defer func() {
		metrics.TaskDuration.WithLabelValues("ratelimit_updateRateLimits").Observe(time.Since(start).Seconds())
	}()

	lastRateLimitUpdateMu.Lock()
	lastTKeys := lastRateLimitUpdateKeys
	lastTRateLimits := lastRateLimitUpdateRateLimits
	lastRateLimitUpdateMu.Unlock()

	tx, err := db.WriterDb.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	dbApiKeys := []struct {
		UserID     int64     `db:"user_id"`
		ApiKey     string    `db:"api_key"`
		ValidUntil time.Time `db:"valid_until"`
		ChangedAt  time.Time `db:"changed_at"`
	}{}

	err = tx.Select(&dbApiKeys, `SELECT user_id, api_key, valid_until, changed_at FROM api_keys WHERE changed_at > $1 OR valid_until < NOW()`, lastTKeys)
	if err != nil {
		return fmt.Errorf("error getting api_keys: %w", err)
	}

	dbRateLimits := []struct {
		UserID     int64     `db:"user_id"`
		Second     int64     `db:"second"`
		Hour       int64     `db:"hour"`
		Month      int64     `db:"month"`
		ValidUntil time.Time `db:"valid_until"`
		ChangedAt  time.Time `db:"changed_at"`
	}{}

	err = tx.Select(&dbRateLimits, `SELECT user_id, second, hour, month, valid_until, changed_at FROM api_ratelimits WHERE changed_at > $1 OR valid_until < NOW()`, lastTRateLimits)
	if err != nil {
		return fmt.Errorf("error getting api_ratelimits: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	dbApiProducts, err := DBGetCurrentApiProducts()
	if err != nil {
		return err
	}

	rateLimitsMu.Lock()
	now := time.Now()

	for _, dbApiProduct := range dbApiProducts {
		if dbApiProduct.Name == "nokey" {
			NoKeyRateLimit.Second = dbApiProduct.Second
			NoKeyRateLimit.Hour = dbApiProduct.Hour
			NoKeyRateLimit.Month = dbApiProduct.Month
		}
		if dbApiProduct.Name == "free" {
			FreeRatelimit.Second = dbApiProduct.Second
			FreeRatelimit.Hour = dbApiProduct.Hour
			FreeRatelimit.Month = dbApiProduct.Month
		}
	}

	for _, dbKey := range dbApiKeys {
		if dbKey.ChangedAt.After(lastTKeys) {
			lastTKeys = dbKey.ChangedAt
		}
		if dbKey.ValidUntil.Before(now) {
			delete(userIdByApiKey, dbKey.ApiKey)
			continue
		}
		userIdByApiKey[dbKey.ApiKey] = dbKey.UserID
	}

	for _, dbRl := range dbRateLimits {
		if dbRl.ChangedAt.After(lastTRateLimits) {
			lastTRateLimits = dbRl.ChangedAt
		}
		if dbRl.ValidUntil.Before(now) {
			delete(rateLimitsByUserId, dbRl.UserID)
			continue
		}
		rlStr := fmt.Sprintf("%d/%d/%d", dbRl.Second, dbRl.Hour, dbRl.Month)
		rl, exists := rateLimits[rlStr]
		if !exists {
			rl = &RateLimit{
				Second: dbRl.Second,
				Hour:   dbRl.Hour,
				Month:  dbRl.Month,
			}
			rateLimits[rlStr] = rl
		}
		rateLimitsByUserId[dbRl.UserID] = rl
	}
	rateLimitsMu.Unlock()
	metrics.TaskDuration.WithLabelValues("ratelimit_updateRateLimits_lock").Observe(time.Since(now).Seconds())

	lastRateLimitUpdateMu.Lock()
	lastRateLimitUpdateKeys = lastTKeys
	lastRateLimitUpdateRateLimits = lastTRateLimits
	lastRateLimitUpdateMu.Unlock()

	return nil
}

// postRateLimit decrements the rate limit keys in redis if the status is not 200.
func postRateLimit(rl *RateLimitResult, status int) error {
	if status == 200 {
		return nil
	}
	// if status is not 200 decrement keys since we do not count unsuccessful requests
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pipe := redisClient.Pipeline()
	for _, k := range rl.RedisKeys {
		pipe.DecrBy(ctx, k.Key, rl.Weight)
		pipe.ExpireAt(ctx, k.Key, k.ExpireAt) // make sure all keys have a TTL
	}
	pipe.DecrBy(ctx, rl.RedisStatsKey, 1)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// rateLimitRequest is the main function for rate limiting, it will check the rate limits for the request and update the rate limits in redis.
func rateLimitRequest(r *http.Request) (*RateLimitResult, error) {
	start := time.Now()
	defer func() {
		metrics.TaskDuration.WithLabelValues("ratelimit_rateLimitRequest").Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*1000)
	defer cancel()

	res := &RateLimitResult{}
	// defer func() { logger.Infof("rateLimitRequest: %+v", *res) }()

	key, ip := getKey(r)
	res.Key = key
	res.IP = ip

	rateLimitsMu.RLock()
	userId, ok := userIdByApiKey[key]
	if !ok {
		res.UserId = -1
		res.IsValidKey = false
		res.RateLimit = NoKeyRateLimit
	} else {
		res.UserId = userId
		res.IsValidKey = true
		limit, ok := rateLimitsByUserId[userId]
		if ok {
			res.RateLimit = limit
		} else {
			res.RateLimit = FreeRatelimit
		}
	}
	rateLimitsMu.RUnlock()

	weight, route, bucket := getWeight(r)
	res.Weight = weight
	res.Route = route
	res.Bucket = bucket

	startUtc := start.UTC()
	res.Time = startUtc

	nextHourUtc := time.Now().Truncate(time.Hour).Add(time.Hour)
	nextMonthUtc := time.Date(startUtc.Year(), startUtc.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	timeUntilNextHourUtc := nextHourUtc.Sub(startUtc)
	timeUntilNextMonthUtc := nextMonthUtc.Sub(startUtc)

	rateLimitSecondKey := fmt.Sprintf("ratelimit:second:%s:%s", res.Bucket, res.UserId)
	rateLimitHourKey := fmt.Sprintf("ratelimit:hour:%04d-%02d-%02d-%02d:%s:%d", startUtc.Year(), startUtc.Month(), startUtc.Day(), startUtc.Hour(), res.Bucket, res.UserId)
	rateLimitMonthKey := fmt.Sprintf("ratelimit:month:%04d-%02d:%s:%d", startUtc.Year(), startUtc.Month(), res.Bucket, res.UserId)

	statsKey := fmt.Sprintf("ratelimit:stats:%04d-%02d-%02d-%02d:%d:%s", startUtc.Year(), startUtc.Month(), startUtc.Day(), startUtc.Hour(), res.UserId, res.Route)
	if !res.IsValidKey {
		statsKey = fmt.Sprintf("ratelimit:stats:%04d-%02d-%02d-%02d:%s:%s", startUtc.Year(), startUtc.Month(), startUtc.Day(), startUtc.Hour(), "nokey", res.Route)
	}
	res.RedisStatsKey = statsKey

	pipe := redisClient.Pipeline()

	var rateLimitSecond, rateLimitHour, rateLimitMonth *redis.IntCmd

	if res.RateLimit.Second > 0 {
		rateLimitSecond = pipe.IncrBy(ctx, rateLimitSecondKey, weight)
		pipe.ExpireNX(ctx, rateLimitSecondKey, time.Second)
	}

	if res.RateLimit.Hour > 0 {
		rateLimitHour = pipe.IncrBy(ctx, rateLimitHourKey, weight)
		pipe.ExpireAt(ctx, rateLimitHourKey, nextHourUtc.Add(time.Second*60)) // expire 1 minute after the window to make sure we do not miss any requests due to time-sync
		res.RedisKeys = append(res.RedisKeys, RedisKey{rateLimitHourKey, nextHourUtc.Add(time.Second * 60)})
	}

	if res.RateLimit.Month > 0 {
		rateLimitMonth = pipe.IncrBy(ctx, rateLimitMonthKey, weight)
		pipe.ExpireAt(ctx, rateLimitMonthKey, nextMonthUtc.Add(time.Second*60)) // expire 1 minute after the window to make sure we do not miss any requests due to time-sync
		res.RedisKeys = append(res.RedisKeys, RedisKey{rateLimitMonthKey, nextMonthUtc.Add(time.Second * 60)})
	}

	pipe.Incr(ctx, statsKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	if res.RateLimit.Second > 0 {
		if rateLimitSecond.Val() > res.RateLimit.Second {
			res.Limit = res.RateLimit.Second
			res.Remaining = 0
			res.Reset = int64(1)
			res.Window = SecondTimeWindow
			return res, nil
		} else if res.RateLimit.Second-rateLimitSecond.Val() > res.Limit {
			res.Limit = res.RateLimit.Second
			res.Remaining = res.RateLimit.Second - rateLimitSecond.Val()
			res.Reset = int64(1)
			res.Window = SecondTimeWindow
		}
	}

	if res.RateLimit.Hour > 0 {
		if rateLimitHour.Val() > res.RateLimit.Hour {
			res.Limit = res.RateLimit.Hour
			res.Remaining = 0
			res.Reset = int64(timeUntilNextHourUtc.Seconds())
			res.Window = HourTimeWindow
			return res, nil
		} else if res.RateLimit.Hour-rateLimitHour.Val() > res.Limit {
			res.Limit = res.RateLimit.Hour
			res.Remaining = res.RateLimit.Hour - rateLimitHour.Val()
			res.Reset = int64(timeUntilNextHourUtc.Seconds())
			res.Window = HourTimeWindow
		}
	}

	if res.RateLimit.Month > 0 {
		if rateLimitMonth.Val() > res.RateLimit.Month {
			res.Limit = res.RateLimit.Month
			res.Remaining = 0
			res.Reset = int64(timeUntilNextMonthUtc.Seconds())
			res.Window = MonthTimeWindow
			return res, nil
		} else if res.RateLimit.Month-rateLimitMonth.Val() > res.Limit {
			res.Limit = res.RateLimit.Month
			res.Remaining = res.RateLimit.Month - rateLimitMonth.Val()
			res.Reset = int64(timeUntilNextMonthUtc.Seconds())
			res.Window = MonthTimeWindow
		}
	}

	return res, nil
}

// getKey returns the key used for RateLimiting. It first checks the query params, then the header and finally the ip address.
func getKey(r *http.Request) (key, ip string) {
	ip = getIP(r)
	key = r.URL.Query().Get("apikey")
	if key != "" {
		return key, ip
	}
	key = r.Header.Get("X-API-KEY")
	if key != "" {
		return key, ip
	}
	return "ip_" + strings.ReplaceAll(ip, ":", "_"), ip
}

// getWeight returns the weight of an endpoint. if the weight of the endpoint is not defined, it returns 1.
func getWeight(r *http.Request) (cost int64, identifier, bucket string) {
	route := getRoute(r)
	weightsMu.RLock()
	weight, weightOk := weights[route]
	bucket, bucketOk := buckets[route]
	weightsMu.RUnlock()
	if !weightOk {
		weight = 1
	}
	if !bucketOk {
		bucket = defaultBucket
	}
	return weight, route, bucket
}

func getRoute(r *http.Request) string {
	route := mux.CurrentRoute(r)
	pathTpl, err := route.GetPathTemplate()
	if err != nil {
		return "UNDEFINED"
	}
	return pathTpl
}

// getIP returns the ip address from the http request
func getIP(r *http.Request) string {
	ips := r.Header.Get("CF-Connecting-IP")
	if ips == "" {
		ips = r.Header.Get("X-Forwarded-For")
	}
	splitIps := strings.Split(ips, ",")

	if len(splitIps) > 0 {
		// get last IP in list since ELB prepends other user defined IPs, meaning the last one is the actual client IP.
		netIP := net.ParseIP(splitIps[len(splitIps)-1])
		if netIP != nil {
			return netIP.String()
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "INVALID"
	}

	netIP := net.ParseIP(ip)
	if netIP != nil {
		ip := netIP.String()
		if ip == "::1" {
			return "127.0.0.1"
		}
		return ip
	}

	return "INVALID"
}

type FallbackRateLimiterClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type FallbackRateLimiter struct {
	clients map[string]*FallbackRateLimiterClient
	mu      sync.Mutex
}

func NewFallbackRateLimiter() *FallbackRateLimiter {
	rl := &FallbackRateLimiter{
		clients: make(map[string]*FallbackRateLimiterClient),
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, client := range rl.clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *FallbackRateLimiter) Handle(w http.ResponseWriter, r *http.Request, next func(writer http.ResponseWriter, request *http.Request)) {
	key, _ := getKey(r)
	rl.mu.Lock()
	if _, found := rl.clients[key]; !found {
		rl.clients[key] = &FallbackRateLimiterClient{limiter: rate.NewLimiter(FallbackRateLimitSecond, FallbackRateLimitBurst)}
	}
	rl.clients[key].lastSeen = time.Now()
	if !rl.clients[key].limiter.Allow() {
		rl.mu.Unlock()
		w.Header().Set(HeaderRateLimitLimit, strconv.FormatInt(FallbackRateLimitSecond, 10))
		w.Header().Set(HeaderRateLimitReset, strconv.FormatInt(1, 10))
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		return
	}
	rl.mu.Unlock()
	next(w, r)
}

type ApiProduct struct {
	Name          string    `db:"name"`
	StripePriceID string    `db:"stripe_price_id"`
	Second        int64     `db:"second"`
	Hour          int64     `db:"hour"`
	Month         int64     `db:"month"`
	ValidFrom     time.Time `db:"valid_from"`
}

func DBGetUserApiRateLimit(userId int64) (*RateLimit, error) {
	rl := &RateLimit{}
	err := db.FrontendWriterDB.Get(rl, `
		select second, hour, month 
		from api_ratelimits 
		where user_id = $1`, userId)
	return rl, err
}

func DBGetCurrentApiProducts() ([]*ApiProduct, error) {
	apiProducts := []*ApiProduct{}
	err := db.FrontendWriterDB.Select(&apiProducts, `
		select distinct on (name) name, stripe_price_id, second, hour, month, valid_from 
		from api_products 
		where valid_from <= now()
		order by name, valid_from desc`)
	return apiProducts, err
}

func DBUpdate() error {
	var err error
	now := time.Now()
	res, err := DBUpdateApiKeys()
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	logrus.Infof("updated %v api_keys in %v", ra, time.Since(now))

	_, err = DBUpdateApiRatelimits()
	if err != nil {
		return err
	}
	ra, err = res.RowsAffected()
	if err != nil {
		return err
	}
	logrus.Infof("updated %v api_ratelimits in %v", ra, time.Since(now))

	_, err = DBInvalidateApiKeys()
	if err != nil {
		return err
	}
	ra, err = res.RowsAffected()
	if err != nil {
		return err
	}
	logrus.Infof("invalidated %v api_keys in %v", ra, time.Since(now))

	return nil
}

func DBInvalidateApiKeys() (sql.Result, error) {
	return db.FrontendWriterDB.Exec(`
		update api_ratelimits 
		set changed_at = now(), valid_until = now() 
		where valid_until > now() 
			and user_id not in (select user_id from api_keys where api_key is not null)`)
}

func DBUpdateApiKeys() (sql.Result, error) {
	return db.FrontendWriterDB.Exec(
		`insert into api_keys (user_id, api_key, valid_until, changed_at)
		select 
			id as user_id, 
			api_key,
			to_timestamp('3000-01-01', 'YYYY-MM-DD') as valid_until,
			now() as changed_at
		from users 
		where api_key is not null
		on conflict (user_id, api_key) do update set
			valid_until = excluded.valid_until,
			changed_at = excluded.changed_at
		where api_keys.valid_until != excluded.valid_until`,
	)
}

func DBUpdateApiRatelimits() (sql.Result, error) {
	return db.FrontendWriterDB.Exec(
		`with 
			current_api_products as (
				select distinct on (name) name, stripe_price_id, second, hour, month, valid_from 
				from api_products 
				where valid_from <= now()
				order by name, valid_from desc
			)
		insert into api_ratelimits (user_id, second, hour, month, valid_until, changed_at)
		select
			u.id as user_id,
			greatest(coalesce(cap1.second,0),coalesce(cap2.second,0)) as second,
			greatest(coalesce(cap1.hour  ,0),coalesce(cap2.hour  ,0)) as hour,
			greatest(coalesce(cap1.month ,0),coalesce(cap2.month ,0)) as month,
			to_timestamp('3000-01-01', 'YYYY-MM-DD') as valid_until,
			now() as changed_at
		from users u
			left join users_stripe_subscriptions uss on uss.customer_id = u.stripe_customer_id and uss.active = true
			left join current_api_products cap on cap.stripe_price_id = uss.price_id
			left join current_api_products cap1 on cap1.name = coalesce(cap.name,'free')
			left join app_subs_view asv on asv.user_id = u.id and asv.active = true
			left join current_api_products cap2 on cap2.name = coalesce(asv.product_id,'free')
		on conflict (user_id) do update set
			second = excluded.second,
			hour = excluded.hour,
			month = excluded.month,
			valid_until = excluded.valid_until,
			changed_at = now()
		where
			api_ratelimits.second != excluded.second 
			or api_ratelimits.hour != excluded.hour 
			or api_ratelimits.month != excluded.month`)
}
