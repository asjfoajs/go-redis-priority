-- lua/push.lua
-- KEYS:
--   KEYS[1] - 计数器键 (例如: "baseKey:count:<level>")
--   KEYS[2] - 队列列表键 (例如: "baseKey:<level>")
--   KEYS[3] - 计数映射键 (例如: "baseKey:count_map")
--   KEYS[4] - 等级映射键 (例如: "baseKey:level_map")
--
-- ARGV:
--   ARGV[1] - 序列化后的元素数据 (JSON 字符串)
--   ARGV[2] - 元素的唯一标识 (ID)
--   ARGV[3] - 当前元素所属的优先级 (数字字符串)
--   ARGV[4] - 最大计数值 (MaxCount)

local count = redis.call("INCR", KEYS[1])
redis.call("RPUSH", KEYS[2], ARGV[1])
redis.call("HSET", KEYS[3], ARGV[2], count)
redis.call("HSET", KEYS[4], ARGV[2], ARGV[3])

if tonumber(count) >= tonumber(ARGV[4]) then
    redis.call("SET", KEYS[1], 0)
end

return count
