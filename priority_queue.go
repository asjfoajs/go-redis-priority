// main/priority_queue.go
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math"
	"sync/atomic"
)

// 内嵌 Lua 脚本
//
//go:embed lua/push.lua
var pushScript string

//go:embed lua/pop.lua
var popScript string

//go:embed lua/countBefore.lua
var countBeforeScript string

//go:embed lua/len.lua
var lenScript string

const (
	// KeyLevelSuffix Redis Key 常量定义
	KeyLevelSuffix    = "{%s}:%d"        // 优先级队列层级键格式
	KeyCountSuffix    = "{%s}:count:%d"  // 计数器键格式
	KeyCountMapSuffix = "{%s}:count_map" // 计数映射表键格式
	KeyLevelMapSuffix = "{%s}:level_map" // 等级映射表键格式
	MaxCount          = 1<<62 - 1        // Redis 的 Int64 最大值
)

// PriorityQueue 表示基于 Redis 的优先级队列
type PriorityQueue struct {
	client         *redis.Client // Redis 客户端实例
	baseKey        string        // 队列基础键名
	levels         []string      // 各优先级层级的键名
	maxLevel       int64         // 最大优先级层级
	levelsCount    []int64       // 每个优先级队列的计数
	pushCount      int64         //最近有多少数据push
	popCount       int64         //最近有多少数据pop
	thresholdCount int64         //超过阈值进行更新
	updater        *Updater      //定时器更新前缀和数组
}

// NewPriorityQueue 创建新的优先级队列实例
func NewPriorityQueue(baseKey string, maxLevel, regular, everyTime, thresholdCount int64, client *redis.Client) *PriorityQueue {
	pq := &PriorityQueue{
		client:         client,
		baseKey:        baseKey,
		maxLevel:       maxLevel,
		levels:         make([]string, maxLevel),
		thresholdCount: thresholdCount,
	}
	// 初始化各层级队列键名
	for i := int64(0); i < maxLevel; i++ {
		pq.levels[i] = fmt.Sprintf(KeyLevelSuffix, baseKey, i+1)
	}

	//定时刷新各层级计数器==>
	pq.updater = NewUpdater(regular, everyTime, pq.CheckRefresh, pq.RefreshLevelsCount)
	return pq
}

// 辅助方法：生成各键名
func (pq *PriorityQueue) countKey(level int64) string {
	return fmt.Sprintf(KeyCountSuffix, pq.baseKey, level)
}
func (pq *PriorityQueue) countMapKey() string {
	return fmt.Sprintf(KeyCountMapSuffix, pq.baseKey)
}
func (pq *PriorityQueue) levelMapKey() string {
	return fmt.Sprintf(KeyLevelMapSuffix, pq.baseKey)
}

// Element 表示队列元素
type Element struct {
	ID    string      // 元素唯一标识
	Value interface{} // 元素值
}

// Push 调用 Lua 脚本完成入队操作
func (pq *PriorityQueue) Push(level int64, elem Element) error {
	if level < 1 || level > pq.maxLevel {
		return fmt.Errorf("invalid level %d, must be 1-%d", level, pq.maxLevel)
	}
	ctx := context.Background()
	data, err := json.Marshal(elem)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	// 构造 Lua 脚本的 KEYS 与 ARGV
	keys := []string{
		pq.countKey(level), // 计数器键
		pq.levels[level-1], // 队列列表键
		pq.countMapKey(),   // 计数映射键
		pq.levelMapKey(),   // 等级映射键
	}
	argv := []interface{}{
		string(data),                // 序列化后的元素数据
		elem.ID,                     // 元素 ID
		fmt.Sprintf("%d", level),    // 元素所属层级
		fmt.Sprintf("%d", MaxCount), // 最大计数值
	}

	// 使用 Eval 调用 push.lua 脚本
	_, err = pq.client.Eval(ctx, pushScript, keys, argv...).Result()

	atomic.AddInt64(&pq.pushCount, 1)

	return err
}

// Pop 调用 Lua 脚本从各级队列中弹出第一个有效元素
func (pq *PriorityQueue) Pop() (*Element, error) {
	ctx := context.Background()

	// 构造 KEYS：
	// 第 1..N 个为各级队列键（这里假设高优先级在前，即从低索引开始）
	keys := make([]string, 0, len(pq.levels)+2)
	for _, key := range pq.levels {
		keys = append(keys, key)
	}
	// 添加计数映射键与等级映射键（放在最后）
	keys = append(keys, pq.countMapKey(), pq.levelMapKey())

	res, err := pq.client.Eval(ctx, popScript, keys).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	data, ok := res.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", res)
	}

	var elem Element
	if err := json.Unmarshal([]byte(data), &elem); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	atomic.AddInt64(&pq.popCount, 1)

	return &elem, nil
}

// CountBefore 调用 Lua 脚本计算指定元素前的数量
func (pq *PriorityQueue) CountBefore(elemID string) (int64, error) {
	ctx := context.Background()
	// 构造 KEYS：
	// KEYS[1]：等级映射键
	// KEYS[2]：计数映射键
	// KEYS[3...]：各级队列键（依次为优先级1, 2, ..., N）
	keys := make([]string, 0, len(pq.levels)+2)
	keys = append(keys, pq.levelMapKey(), pq.countMapKey())
	keys = append(keys, pq.levels...)
	argv := []interface{}{
		elemID,
		fmt.Sprintf("%d", MaxCount),
	}

	res, err := pq.client.Eval(ctx, countBeforeScript, keys, argv...).Int64Slice()
	if err != nil {
		return -1, err
	}

	if len(res) != 2 {
		return -1, fmt.Errorf("two values are expected to be returned, which are actually returned: %v", res)
	}

	countVal := res[0]
	levelVal := res[1]

	if levelVal <= 0 {
		return countVal, nil
	}

	return countVal + pq.levelsCount[levelVal-1], nil
}

// Pull 延迟删除元素
func (pq *PriorityQueue) Pull(elemID string) error {
	return pq.client.HDel(context.Background(), pq.countMapKey(), elemID).Err()
}

// RefreshLevelsCount 刷新各层级计数器
func (pq *PriorityQueue) RefreshLevelsCount() error {
	ctx := context.Background()
	res, err := pq.client.Eval(ctx, lenScript, pq.levels).Int64Slice()
	if err != nil {
		return fmt.Errorf("eval error: %v", err)
	}

	if res == nil {
		return nil
	}

	//刷新完之后清空值
	pq.levelsCount = CalculatePrefixSum(res)
	pq.pushCount = 0
	pq.popCount = 0
	return nil
}

// CheckRefresh 检查是否需要刷新
func (pq *PriorityQueue) CheckRefresh() bool {
	return float64(atomic.LoadInt64(&pq.thresholdCount)) < math.Abs(float64(atomic.LoadInt64(&pq.pushCount)-atomic.LoadInt64(&pq.popCount)))
}

// Stop 关闭定时器
func (pq *PriorityQueue) Stop() {
	pq.updater.Stop()
}
