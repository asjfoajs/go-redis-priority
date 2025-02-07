-- lua/countBefore.lua
-- KEYS:
--   KEYS[1] - 等级映射键 (例如: "baseKey:level_map")
--   KEYS[2] - 计数映射键 (例如: "baseKey:count_map")
--   KEYS[3...] - 各优先级队列的列表键 (顺序：优先级1, 2, ..., N)
--
-- ARGV:
--   ARGV[1] - 目标元素 ID
--   ARGV[2] - 最大计数值 (MaxCount)

local elemID = ARGV[1]
local maxCount = tonumber(ARGV[2])

local levelStr = redis.call("HGET", KEYS[1], elemID)
if not levelStr then
    return {err="element not found in level map"}
end
local level = tonumber(levelStr)

local countStr = redis.call("HGET", KEYS[2], elemID)
if not countStr then
    return {err="element not found in count map"}
end
local count = tonumber(countStr)

-- 计算当前层对应的队列键，注意：KEYS[3] 对应优先级1，依此类推
local listKey = KEYS[2 + level]
local headValue = redis.call("LINDEX", listKey, 0)
if not headValue then
    return {err="no head element in list for level " .. level}
end
local head = cjson.decode(headValue)

local headCountStr = redis.call("HGET", KEYS[2], head.ID)
if not headCountStr then
    return {err="head element not found in count map"}
end
local headCount = tonumber(headCountStr)

if count > headCount then
    count = count - headCount
else
    count = ((count - headCount) + maxCount) % maxCount
end

local total = count
for i = 2, level do
    total = total + redis.call("LLEN", KEYS[i+1])
end


return total
