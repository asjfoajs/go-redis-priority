-- lua/pop.lua
-- KEYS:
--   KEYS[1..N]    - 各个优先级队列的列表键（高优先级在前）
--   KEYS[N+1]     - 计数映射键 (例如: "baseKey:count_map")
--   KEYS[N+2]     - 等级映射键 (例如: "baseKey:level_map")
--
-- ARGV: 无

local numQueues = #KEYS - 2
for i = 1, numQueues do
    while true do
        local data = redis.call("LPOP", KEYS[i])
        if not data then
            break  -- 当前队列为空，跳出内层循环
        end

        local elem = cjson.decode(data)
        local exists = redis.call("HEXISTS", KEYS[numQueues+1], elem.ID)
        if exists == 1 then
            -- 清理映射
            redis.call("HDEL", KEYS[numQueues+2], elem.ID)
            redis.call("HDEL", KEYS[numQueues+1], elem.ID)
            return data
        end
    end
end

return nil
