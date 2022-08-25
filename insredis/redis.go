package insredis

import (
	"time"

	"github.com/go-redis/redis"
)

// RedisInterface is a universal golang redis interface.
type RedisInterface interface {
	Watch(fn func(*redis.Tx) error, keys ...string) error
	Process(cmd redis.Cmder) error
	WrapProcess(fn func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error)
	WrapProcessPipeline(fn func(oldProcess func([]redis.Cmder) error) func([]redis.Cmder) error)
	Subscribe(channels ...string) *redis.PubSub
	PSubscribe(channels ...string) *redis.PubSub
	Pipeline() redis.Pipeliner
	Pipelined(fn func(redis.Pipeliner) error) ([]redis.Cmder, error)
	TxPipelined(fn func(redis.Pipeliner) error) ([]redis.Cmder, error)
	TxPipeline() redis.Pipeliner
	Command() *redis.CommandsInfoCmd
	ClientGetName() *redis.StringCmd
	Echo(message interface{}) *redis.StringCmd
	Ping() *redis.StatusCmd
	Quit() *redis.StatusCmd
	Del(keys ...string) *redis.IntCmd
	Unlink(keys ...string) *redis.IntCmd
	Dump(key string) *redis.StringCmd
	Exists(keys ...string) *redis.IntCmd
	Expire(key string, expiration time.Duration) *redis.BoolCmd
	ExpireAt(key string, tm time.Time) *redis.BoolCmd
	Keys(pattern string) *redis.StringSliceCmd
	Migrate(host, port, key string, db int64, timeout time.Duration) *redis.StatusCmd
	Move(key string, db int64) *redis.BoolCmd
	ObjectRefCount(key string) *redis.IntCmd
	ObjectEncoding(key string) *redis.StringCmd
	ObjectIdleTime(key string) *redis.DurationCmd
	Persist(key string) *redis.BoolCmd
	PExpire(key string, expiration time.Duration) *redis.BoolCmd
	PExpireAt(key string, tm time.Time) *redis.BoolCmd
	PTTL(key string) *redis.DurationCmd
	RandomKey() *redis.StringCmd
	Rename(key, newkey string) *redis.StatusCmd
	RenameNX(key, newkey string) *redis.BoolCmd
	Restore(key string, ttl time.Duration, value string) *redis.StatusCmd
	RestoreReplace(key string, ttl time.Duration, value string) *redis.StatusCmd
	Sort(key string, sort *redis.Sort) *redis.StringSliceCmd
	SortStore(key, store string, sort *redis.Sort) *redis.IntCmd
	SortInterfaces(key string, sort *redis.Sort) *redis.SliceCmd
	Touch(keys ...string) *redis.IntCmd
	TTL(key string) *redis.DurationCmd
	Type(key string) *redis.StatusCmd
	Scan(cursor uint64, match string, count int64) *redis.ScanCmd
	SScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd
	HScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd
	ZScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd
	Append(key, value string) *redis.IntCmd
	BitCount(key string, bitCount *redis.BitCount) *redis.IntCmd
	BitOpAnd(destKey string, keys ...string) *redis.IntCmd
	BitOpOr(destKey string, keys ...string) *redis.IntCmd
	BitOpXor(destKey string, keys ...string) *redis.IntCmd
	BitOpNot(destKey string, key string) *redis.IntCmd
	BitPos(key string, bit int64, pos ...int64) *redis.IntCmd
	Decr(key string) *redis.IntCmd
	DecrBy(key string, decrement int64) *redis.IntCmd
	Get(key string) *redis.StringCmd
	GetBit(key string, offset int64) *redis.IntCmd
	GetRange(key string, start, end int64) *redis.StringCmd
	GetSet(key string, value interface{}) *redis.StringCmd
	Incr(key string) *redis.IntCmd
	IncrBy(key string, value int64) *redis.IntCmd
	IncrByFloat(key string, value float64) *redis.FloatCmd
	MGet(keys ...string) *redis.SliceCmd
	MSet(pairs ...interface{}) *redis.StatusCmd
	MSetNX(pairs ...interface{}) *redis.BoolCmd
	Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetBit(key string, offset int64, value int) *redis.IntCmd
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	SetXX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	SetRange(key string, offset int64, value string) *redis.IntCmd
	StrLen(key string) *redis.IntCmd
	HDel(key string, fields ...string) *redis.IntCmd
	HExists(key, field string) *redis.BoolCmd
	HGet(key, field string) *redis.StringCmd
	HGetAll(key string) *redis.StringStringMapCmd
	HIncrBy(key, field string, incr int64) *redis.IntCmd
	HIncrByFloat(key, field string, incr float64) *redis.FloatCmd
	HKeys(key string) *redis.StringSliceCmd
	HLen(key string) *redis.IntCmd
	HMGet(key string, fields ...string) *redis.SliceCmd
	HMSet(key string, fields map[string]interface{}) *redis.StatusCmd
	HSet(key, field string, value interface{}) *redis.BoolCmd
	HSetNX(key, field string, value interface{}) *redis.BoolCmd
	HVals(key string) *redis.StringSliceCmd
	BLPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd
	BRPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd
	BRPopLPush(source, destination string, timeout time.Duration) *redis.StringCmd
	LIndex(key string, index int64) *redis.StringCmd
	LInsert(key, op string, pivot, value interface{}) *redis.IntCmd
	LInsertBefore(key string, pivot, value interface{}) *redis.IntCmd
	LInsertAfter(key string, pivot, value interface{}) *redis.IntCmd
	LLen(key string) *redis.IntCmd
	LPop(key string) *redis.StringCmd
	LPush(key string, values ...interface{}) *redis.IntCmd
	LPushX(key string, value interface{}) *redis.IntCmd
	LRange(key string, start, stop int64) *redis.StringSliceCmd
	LRem(key string, count int64, value interface{}) *redis.IntCmd
	LSet(key string, index int64, value interface{}) *redis.StatusCmd
	LTrim(key string, start, stop int64) *redis.StatusCmd
	RPop(key string) *redis.StringCmd
	RPopLPush(source, destination string) *redis.StringCmd
	RPush(key string, values ...interface{}) *redis.IntCmd
	RPushX(key string, value interface{}) *redis.IntCmd
	SAdd(key string, members ...interface{}) *redis.IntCmd
	SCard(key string) *redis.IntCmd
	SDiff(keys ...string) *redis.StringSliceCmd
	SDiffStore(destination string, keys ...string) *redis.IntCmd
	SInter(keys ...string) *redis.StringSliceCmd
	SInterStore(destination string, keys ...string) *redis.IntCmd
	SIsMember(key string, member interface{}) *redis.BoolCmd
	SMembers(key string) *redis.StringSliceCmd
	SMembersMap(key string) *redis.StringStructMapCmd
	SMove(source, destination string, member interface{}) *redis.BoolCmd
	SPop(key string) *redis.StringCmd
	SPopN(key string, count int64) *redis.StringSliceCmd
	SRandMember(key string) *redis.StringCmd
	SRandMemberN(key string, count int64) *redis.StringSliceCmd
	SRem(key string, members ...interface{}) *redis.IntCmd
	SUnion(keys ...string) *redis.StringSliceCmd
	SUnionStore(destination string, keys ...string) *redis.IntCmd
	XAdd(a *redis.XAddArgs) *redis.StringCmd
	XDel(stream string, ids ...string) *redis.IntCmd
	XLen(stream string) *redis.IntCmd
	XRange(stream, start, stop string) *redis.XMessageSliceCmd
	XRangeN(stream, start, stop string, count int64) *redis.XMessageSliceCmd
	XRevRange(stream string, start, stop string) *redis.XMessageSliceCmd
	XRevRangeN(stream string, start, stop string, count int64) *redis.XMessageSliceCmd
	XRead(a *redis.XReadArgs) *redis.XStreamSliceCmd
	XReadStreams(streams ...string) *redis.XStreamSliceCmd
	XGroupCreate(stream, group, start string) *redis.StatusCmd
	XGroupCreateMkStream(stream, group, start string) *redis.StatusCmd
	XGroupSetID(stream, group, start string) *redis.StatusCmd
	XGroupDestroy(stream, group string) *redis.IntCmd
	XGroupDelConsumer(stream, group, consumer string) *redis.IntCmd
	XReadGroup(a *redis.XReadGroupArgs) *redis.XStreamSliceCmd
	XAck(stream, group string, ids ...string) *redis.IntCmd
	XPending(stream, group string) *redis.XPendingCmd
	XPendingExt(a *redis.XPendingExtArgs) *redis.XPendingExtCmd
	XClaim(a *redis.XClaimArgs) *redis.XMessageSliceCmd
	XClaimJustID(a *redis.XClaimArgs) *redis.StringSliceCmd
	XTrim(key string, maxLen int64) *redis.IntCmd
	XTrimApprox(key string, maxLen int64) *redis.IntCmd
	BZPopMax(timeout time.Duration, keys ...string) *redis.ZWithKeyCmd
	BZPopMin(timeout time.Duration, keys ...string) *redis.ZWithKeyCmd
	ZAdd(key string, members ...redis.Z) *redis.IntCmd
	ZAddNX(key string, members ...redis.Z) *redis.IntCmd
	ZAddXX(key string, members ...redis.Z) *redis.IntCmd
	ZAddCh(key string, members ...redis.Z) *redis.IntCmd
	ZAddNXCh(key string, members ...redis.Z) *redis.IntCmd
	ZAddXXCh(key string, members ...redis.Z) *redis.IntCmd
	ZIncr(key string, member redis.Z) *redis.FloatCmd
	ZIncrNX(key string, member redis.Z) *redis.FloatCmd
	ZIncrXX(key string, member redis.Z) *redis.FloatCmd
	ZCard(key string) *redis.IntCmd
	ZCount(key, min, max string) *redis.IntCmd
	ZLexCount(key, min, max string) *redis.IntCmd
	ZIncrBy(key string, increment float64, member string) *redis.FloatCmd
	ZInterStore(destination string, store redis.ZStore, keys ...string) *redis.IntCmd
	ZPopMax(key string, count ...int64) *redis.ZSliceCmd
	ZPopMin(key string, count ...int64) *redis.ZSliceCmd
	ZRange(key string, start, stop int64) *redis.StringSliceCmd
	ZRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd
	ZRangeByScore(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRangeByLex(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRangeByScoreWithScores(key string, opt redis.ZRangeBy) *redis.ZSliceCmd
	ZRank(key, member string) *redis.IntCmd
	ZRem(key string, members ...interface{}) *redis.IntCmd
	ZRemRangeByRank(key string, start, stop int64) *redis.IntCmd
	ZRemRangeByScore(key, min, max string) *redis.IntCmd
	ZRemRangeByLex(key, min, max string) *redis.IntCmd
	ZRevRange(key string, start, stop int64) *redis.StringSliceCmd
	ZRevRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd
	ZRevRangeByScore(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRevRangeByLex(key string, opt redis.ZRangeBy) *redis.StringSliceCmd
	ZRevRangeByScoreWithScores(key string, opt redis.ZRangeBy) *redis.ZSliceCmd
	ZRevRank(key, member string) *redis.IntCmd
	ZScore(key, member string) *redis.FloatCmd
	ZUnionStore(dest string, store redis.ZStore, keys ...string) *redis.IntCmd
	PFAdd(key string, els ...interface{}) *redis.IntCmd
	PFCount(keys ...string) *redis.IntCmd
	PFMerge(dest string, keys ...string) *redis.StatusCmd
	BgRewriteAOF() *redis.StatusCmd
	BgSave() *redis.StatusCmd
	ClientKill(ipPort string) *redis.StatusCmd
	ClientKillByFilter(keys ...string) *redis.IntCmd
	ClientList() *redis.StringCmd
	ClientPause(dur time.Duration) *redis.BoolCmd
	ClientID() *redis.IntCmd
	ConfigGet(parameter string) *redis.SliceCmd
	ConfigResetStat() *redis.StatusCmd
	ConfigSet(parameter, value string) *redis.StatusCmd
	ConfigRewrite() *redis.StatusCmd
	DBSize() *redis.IntCmd
	FlushAll() *redis.StatusCmd
	FlushAllAsync() *redis.StatusCmd
	FlushDB() *redis.StatusCmd
	FlushDBAsync() *redis.StatusCmd
	Info(section ...string) *redis.StringCmd
	LastSave() *redis.IntCmd
	Save() *redis.StatusCmd
	Shutdown() *redis.StatusCmd
	ShutdownSave() *redis.StatusCmd
	ShutdownNoSave() *redis.StatusCmd
	SlaveOf(host, port string) *redis.StatusCmd
	Time() *redis.TimeCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(hashes ...string) *redis.BoolSliceCmd
	ScriptFlush() *redis.StatusCmd
	ScriptKill() *redis.StatusCmd
	ScriptLoad(script string) *redis.StringCmd
	DebugObject(key string) *redis.StringCmd
	Publish(channel string, message interface{}) *redis.IntCmd
	PubSubChannels(pattern string) *redis.StringSliceCmd
	PubSubNumSub(channels ...string) *redis.StringIntMapCmd
	PubSubNumPat() *redis.IntCmd
	ClusterSlots() *redis.ClusterSlotsCmd
	ClusterNodes() *redis.StringCmd
	ClusterMeet(host, port string) *redis.StatusCmd
	ClusterForget(nodeID string) *redis.StatusCmd
	ClusterReplicate(nodeID string) *redis.StatusCmd
	ClusterResetSoft() *redis.StatusCmd
	ClusterResetHard() *redis.StatusCmd
	ClusterInfo() *redis.StringCmd
	ClusterKeySlot(key string) *redis.IntCmd
	ClusterGetKeysInSlot(slot int, count int) *redis.StringSliceCmd
	ClusterCountFailureReports(nodeID string) *redis.IntCmd
	ClusterCountKeysInSlot(slot int) *redis.IntCmd
	ClusterDelSlots(slots ...int) *redis.StatusCmd
	ClusterDelSlotsRange(min, max int) *redis.StatusCmd
	ClusterSaveConfig() *redis.StatusCmd
	ClusterSlaves(nodeID string) *redis.StringSliceCmd
	ClusterFailover() *redis.StatusCmd
	ClusterAddSlots(slots ...int) *redis.StatusCmd
	ClusterAddSlotsRange(min, max int) *redis.StatusCmd
	GeoAdd(key string, geoLocation ...*redis.GeoLocation) *redis.IntCmd
	GeoPos(key string, members ...string) *redis.GeoPosCmd
	GeoRadius(key string, longitude, latitude float64, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd
	GeoRadiusRO(key string, longitude, latitude float64, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd
	GeoRadiusByMember(key, member string, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd
	GeoRadiusByMemberRO(key, member string, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd
	GeoDist(key string, member1, member2, unit string) *redis.FloatCmd
	GeoHash(key string, members ...string) *redis.StringSliceCmd
	ReadOnly() *redis.StatusCmd
	ReadWrite() *redis.StatusCmd
	MemoryUsage(key string, samples ...int) *redis.IntCmd
	Close() error
}

// redisClient is the singleton client created by InitRedis function.
var redisClient *redis.Client

// Init creates a client pool for redis connections.
func Init(redisHost string, poolSize int) *redis.Client {
	if redisClient != nil {
		return redisClient
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost,
		PoolSize: poolSize,
	})

	return redisClient
}

func GetClient() *redis.Client {
	return redisClient
}
