-- lua/len.lua
-- KEYS:
--   KEYS[1] - 所有list的key
--
-- ARGV:
--   ARGV[1] - 所有list的len

local result = {}
for i, key in ipairs(KEYS) do
    result[i] = redis.call("LLEN", key)
end
return result
