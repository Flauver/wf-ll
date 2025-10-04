-- moran_pin.lua
-- version: 0.2.0
-- author: kuroame (modified)
-- license: GPLv3
-- You may copy, distribute and modify the software as long as you track
-- changes/dates in source files. Any modifications to or software including
-- (via compiler) GPL-licensed code must also be made available under the GPL
-- along with build & install instructions.

-- changelog
-- 0.2.0: fix pin order (first pinned = first position), add manual position adjustment
-- 0.1.2: add freestyle mode, add switch to enable/disable pin
-- 0.1.1: simple configuration
-- 0.1.0: init

-- 从原 moran.lua 整合的模块功能
local moran_pin_aux = {}

---Check if a Unicode codepoint is a Chinese character. Up to Unicode 17.
---@param codepoint integer
---@return boolean
function moran_pin_aux.unicode_code_point_is_chinese(codepoint)
   return (codepoint >= 0x4E00 and codepoint <= 0x9FFF)   -- basic
      or (codepoint >= 0x3400 and codepoint <= 0x4DBF)    -- ext a
      or (codepoint >= 0x20000 and codepoint <= 0x2A6DF)  -- ext b
      or (codepoint >= 0x2A700 and codepoint <= 0x2B73F)  -- ext c
      or (codepoint >= 0x2B740 and codepoint <= 0x2B81F)  -- ext d
      or (codepoint >= 0x2B820 and codepoint <= 0x2CEAF)  -- ext e
      or (codepoint >= 0x2CEB0 and codepoint <= 0x2EBE0)  -- ext f
      or (codepoint >= 0x30000 and codepoint <= 0x3134A)  -- ext g
      or (codepoint >= 0x31350 and codepoint <= 0x323AF)  -- ext h
      or (codepoint >= 0x2EBF0 and codepoint <= 0x2EE5F)  -- ext i
      or (codepoint >= 0x323B0 and codepoint <= 0x3347f)  -- ext j
end

---Get a stateful iterator of each unicode codepoint in a string
---@param word string
---@return function():number?
function moran_pin_aux.codepoints(word)
    local f, s, i = utf8.codes(word)
    local value = nil
    return function()
        i, value = f(s, i)
        if i then
            return i, value
        else
            return nil
        end
    end
end

---Return true if @str is purely Chinese.
---@param str str
---@return boolean
function moran_pin_aux.str_is_chinese(str)
   for _, cp in moran_pin_aux.codepoints(str) do
      if not moran_pin_aux.unicode_code_point_is_chinese(cp) then
         return false
      end
   end
   return true
end

-- userdb
-- 将用户的pin记录存储在userdb中
local user_db = {}
local sep_t = " \t"
-- epoch : 2024/11/11 00:00 in min
local epoch = 28854240
local ref_count = 0
local pin_db = nil
function user_db.release()
    ref_count = ref_count - 1
    if ref_count == 0 then
        collectgarbage()
        if pin_db:loaded() then
            pin_db:close()
        end
        pin_db = nil
    end
end

function user_db.acquire()
    if ref_count == 0 then
        pin_db = LevelDb("moran_pin")
        if not pin_db:loaded() then
            pin_db:open()
            if not pin_db:loaded() then
                return
            end
        end
    end
    ref_count = ref_count + 1
end

---@param input string
---@return function iterator
function user_db.query_and_unpack(input)
    local res = pin_db:query(input .. sep_t)
    local function iter()
        if not res then return nil end
        local next_func, self = res:iter()
        return function()
            while true do
                local key, value = next_func(self)
                if key == nil then
                    return nil
                end
                local entry = user_db.unpack_entry(key, value)
                if entry ~= nil then
                    return entry
                end
            end
        end
    end
    return iter()
end

function user_db.timestamp_now()
    return math.floor((os.time()) / 60) - epoch
end

---@param n string weight/output commits
---@param m string timestamp in min from epoch
---@return str encoded commits
function user_db.encode(n, m)
    local n_prime = n + 8 -- move the range to [0, 15]
    if n >= 0 then
        return m * 16 + n_prime
    else
        return -(m * 16 + n_prime)
    end
end

---@param x string encoded commits
---@return n string weight/output commits
---@return m string timestamp in min from epoch
function user_db.decode(x)
    local n, m
    if x >= 0 then
        m = math.floor(x / 16)
        n = (x % 16) - 8
    else
        local x_abs = -x
        m = math.floor(x_abs / 16)
        n = (x_abs % 16) - 8
    end
    return n, m
end

---@param key string
---@param value string
---@return table|nil
function user_db.unpack_entry(key, value)
    local result = {}

    local code, phrase = key:match("^(.-)%s+(.+)$")
    if code and phrase then
        result.code = code
        result.phrase = phrase
    else
        return nil
    end

    local commits, dee, tick, position = 0, 0.0, 0, nil
    for k, v in value:gmatch("(%a+)=(%S+)") do
        if k == "c" then
            commits = tonumber(v) or 0
        elseif k == "d" then
            dee = math.min(10000.0, tonumber(v) or 0.0)
        elseif k == "t" then
            tick = tonumber(v) or 0
        elseif k == "p" then
            position = tonumber(v)
        end
    end
    local output_commits, timestamp = user_db.decode(commits)

    -- just neglect tombstoned entries
    if output_commits < 0 then
        return nil
    end

    result.raw_commits = commits
    result.commits = output_commits
    result.timestamp = timestamp
    result.dee = dee
    result.tick = tick
    result.position = position

    return result
end

---@param input string
---@param cand_text string
function user_db.toggle_pin_status(input, cand_text)
    local pinned_res = pin_db:query(input .. sep_t)
    if pinned_res ~= nil then
        local key = input .. sep_t .. cand_text
        local entries = {}
        local found_existing = false
        
        for k, v in pinned_res:iter() do
            local unpacked = user_db.unpack_entry(k, v)
            if unpacked then
                table.insert(entries, {key = k, entry = unpacked})
                if key == k then
                    found_existing = true
                    -- if it's an active one, tombstone it
                    if unpacked.commits >= 0 then
                        user_db.tombstone(key)
                        return
                    end
                end
            end
        end

        if not found_existing then
            -- new pin, assign next available position
            local next_position = #entries + 1
            user_db.upsert(key, 0, next_position)
        end
    else
        -- first pin for this input
        user_db.upsert(input .. sep_t .. cand_text, 0, 1)
    end
end

---@param input string
---@param cand_text string
---@param direction number 1 for up, -1 for down
function user_db.adjust_position(input, cand_text, direction)
    local pinned_res = pin_db:query(input .. sep_t)
    if not pinned_res then return end
    
    local entries = {}
    local target_key = input .. sep_t .. cand_text
    local target_entry = nil
    
    -- collect all entries
    for k, v in pinned_res:iter() do
        local unpacked = user_db.unpack_entry(k, v)
        if unpacked and unpacked.commits >= 0 then
            table.insert(entries, {key = k, entry = unpacked})
            if k == target_key then
                target_entry = entries[#entries]
            end
        end
    end
    
    if not target_entry then return end
    
    -- sort by position
    table.sort(entries, function(a, b)
        local pos_a = a.entry.position or math.huge
        local pos_b = b.entry.position or math.huge
        return pos_a < pos_b
    end)
    
    -- find current index
    local current_index = nil
    for i, e in ipairs(entries) do
        if e.key == target_key then
            current_index = i
            break
        end
    end
    
    if not current_index then return end
    
    -- calculate new index
    local new_index = current_index + direction
    if new_index < 1 or new_index > #entries then
        return -- can't move beyond bounds
    end
    
    -- swap positions
    local temp = entries[current_index]
    entries[current_index] = entries[new_index]
    entries[new_index] = temp
    
    -- update positions in database
    for i, e in ipairs(entries) do
        user_db.update_position(e.key, i)
    end
end

function user_db.update_position(key, position)
    local res = pin_db:query(key)
    if res then
        for k, v in res:iter() do
            if k == key then
                local unpacked = user_db.unpack_entry(k, v)
                if unpacked then
                    local encoded_commit = user_db.encode(unpacked.commits, unpacked.timestamp)
                    pin_db:update(key, "c=" .. encoded_commit .. " d=" .. unpacked.dee .. " t=" .. unpacked.tick .. " p=" .. position)
                end
                break
            end
        end
    end
end

function user_db.dump_raw()
    local res = pin_db:query("")
    local function iter()
        local next_func, self = res:iter()
        return function()
            while true do
                local key, value = next_func(self)
                if key == nil then
                    return nil
                end
                return key, value
            end
        end
    end
    return iter()
end

function user_db.upsert(key, output_commits, position)
    local encoded_commit = user_db.encode(output_commits, user_db.timestamp_now())
    local value = "c=" .. encoded_commit .. " d=0 t=1"
    if position then
        value = value .. " p=" .. position
    end
    pin_db:update(key, value)
end

function user_db.tombstone(key)
    user_db.upsert(key, -1)
end

-- pin_processor
-- 处理ctrl+t和位置调整
local kAccepted = 1
local kNoop = 2
local pin_processor = {}

function pin_processor.init(env)
    env.pin_enable = env.engine.schema.config:get_bool("moran/pin/enable") or false
    if not env.pin_enable then
        return
    end
    user_db.acquire()
end

function pin_processor.fini(env)
    if not env.pin_enable then
        return
    end
    user_db.release()
end

function pin_processor.func(key_event, env)
    if not env.pin_enable then
        return kNoop
    end
    -- ctrl + x to trigger
    if not key_event:ctrl() or key_event:release() then
        return kNoop
    end
    
    local context = env.engine.context
    local input = context.input
    local cand = context:get_selected_candidate()
    
    -- + t to toggle pin
    if key_event.keycode == 0x74 then
        if cand == nil then
            return kNoop
        end
        local text = cand.text
        -- 1) Special-case pure Chinese candidates: the text could be
        -- output from OpenCC, so pin the genuine candidate instead to
        -- preserve word frequency.
        --
        -- 2) If we know for sure this is a pinned candidate, always
        -- retrieve the genuine candidate to correctly delete it.
        if cand.type == 'pinned' or moran_pin_aux.str_is_chinese(text) then
            text = cand:get_genuine().text
        end
        user_db.toggle_pin_status(input, text)
        context:refresh_non_confirmed_composition()
        return kAccepted
    -- + shift + up to move pin up
    elseif key_event:shift() and key_event.keycode == 0xff52 then
        if cand and cand.type == 'pinned' then
            local text = cand:get_genuine().text
            user_db.adjust_position(input, text, -1)
            context:refresh_non_confirmed_composition()
            return kAccepted
        end
    -- + shift + down to move pin down
    elseif key_event:shift() and key_event.keycode == 0xff54 then
        if cand and cand.type == 'pinned' then
            local text = cand:get_genuine().text
            user_db.adjust_position(input, text, 1)
            context:refresh_non_confirmed_composition()
            return kAccepted
        end
    -- + a
    elseif key_event.keycode == 0x61 then
        -- todo: add quick code
        return kNoop
    else
        return kNoop
    end
    return kNoop
end

-- pin_filter
-- 从pin记录中读取候选项，并将其插入到候选列表的最前面
local pin_filter = {}

function pin_filter.init(env)
    env.pin_enable = env.engine.schema.config:get_bool("moran/pin/enable") or false
    if not env.pin_enable then
        return
    end
    env.indicator = env.engine.schema.config:get_string("moran/pin/indicator") or "📌"
    user_db.acquire()
end

function pin_filter.fini(env)
    if not env.pin_enable then
        return
    end
    user_db.release()
end

function pin_filter.func(t_input, env)
    if env.pin_enable and env.engine.context.composition:toSegmentation():get_confirmed_position() == 0 then
        local input = env.engine.context.input
        local commits = {}
        local entries = user_db.query_and_unpack(input)
        if entries then
            for unpacked in entries do
                table.insert(commits, unpacked)
            end
        end
        -- sort by position (ascending)
        table.sort(commits, function(a, b)
            local pos_a = a.position or math.huge
            local pos_b = b.position or math.huge
            if pos_a ~= pos_b then
                return pos_a < pos_b
            end
            -- fallback to timestamp if positions are equal
            return a.timestamp < b.timestamp
        end)
        for _, unpacked in ipairs(commits) do
            local cand = Candidate("pinned", 0, #input, unpacked.phrase, env.indicator)
            cand.preedit = input
            yield(cand)
        end
    end
    for cand in t_input:iter() do
        yield(cand)
    end
end

-- panacea_translator
-- 基于pin功能 以 编码[infix]词 的形式触发，灵活造词
local panacea_translator = {}

function panacea_translator.init(env)
    env.pin_enable = env.engine.schema.config:get_bool("moran/pin/enable") or false
    if not env.pin_enable then
        return
    end
    env.infix = env.engine.schema.config:get_string("moran/pin/panacea/infix") or '//'
    env.escaped_infix = string.gsub(env.infix, "([%^%$%(%)%%%.%[%]%*%+%-%?])", "%%%1")
    env.prompt = env.engine.schema.config:get_string("moran/pin/panacea/prompt") or "〔加词〕"
    env.indicator = env.engine.schema.config:get_string("moran/pin/indicator") or "📌"
    env.freestyle = env.engine.schema.config:get_bool("moran/pin/panacea/freestyle") or false

    env.freestyle_state = false
    env.freestyle_text = ""
    env.freestyle_code = ""

    user_db.acquire()
    local pattern = string.format("(.+)%s(.+)", env.escaped_infix)
    local function on_commit(ctx)
        local selected_cand = ctx:get_selected_candidate()
        if selected_cand ~= nil then
            local gen_cand = selected_cand:get_genuine()
            if env.freestyle and gen_cand.type == "pin_tip" then
                if env.freestyle_state then
                    if env.freestyle_code and env.freestyle_code ~= "" and env.freestyle_text and env.freestyle_text ~= "" then
                        user_db.toggle_pin_status(env.freestyle_code, env.freestyle_text)
                        env.freestyle_code = ""
                        env.freestyle_text = ""
                    end
                else
                    if string.sub(ctx.input, - #env.infix) == env.infix then
                        env.freestyle_code = string.sub(ctx.input, 1, #ctx.input - #env.infix)
                    end

                    if env.freestyle_code == "" then
                        return
                    end
                end
                env.freestyle_state = not env.freestyle_state
                return
            end
        end

        local commit_text = ctx:get_commit_text()
        if moran_pin_aux.str_is_chinese(commit_text) then
            local segmentation = ctx.composition:toSegmentation()
            local segs = segmentation:get_segments()
            local genuine_text = ""
            local ok = true
            for _, seg in pairs(segs) do
                local c = seg:get_selected_candidate()
                if c == nil then
                    ok = false
                    break
                end
                local g = c:get_genuine()
                genuine_text = genuine_text .. g.text
            end
            if ok then
                commit_text = genuine_text
            end
        end

        if env.freestyle_state then
            env.freestyle_text = env.freestyle_text .. commit_text
            return
        end


        local code, original_code = ctx.input:match(pattern)
        if original_code and original_code ~= "" and
            code and code ~= "" and
            commit_text and commit_text ~= "" then
            user_db.toggle_pin_status(code, commit_text)
        end
    end

    local function on_update_or_select(ctx)
        if not ctx.input then
            return
        end

        if env.freestyle_state then
            local segment = ctx.composition:back()
            if segment ~= nil then
                segment.prompt = env.prompt .. " | " .. env.freestyle_text .. " | " .. env.freestyle_code
            end
            return
        end

        local code, original_code = ctx.input:match(pattern)
        if original_code and code then
            local segment = ctx.composition:back()
            segment.prompt = env.prompt .. " | " .. code
        end
    end

    env.commit_notifier = env.engine.context.commit_notifier:connect(on_commit)
    env.select_notifier = env.engine.context.select_notifier:connect(on_update_or_select)
    env.update_notifier = env.engine.context.update_notifier:connect(on_update_or_select)
end

function panacea_translator.fini(env)
    if not env.pin_enable then
        return
    end
    env.commit_notifier:disconnect()
    env.select_notifier:disconnect()
    env.update_notifier:disconnect()
    user_db.release()
end

function panacea_translator.func(input, seg, env)
    if not env.pin_enable then
        return
    end
    local pattern = "[a-zA-Z;,./]+" .. env.escaped_infix
    local match = input:match(pattern)

    if match then
        local comment = "➕" .. env.indicator
        if env.freestyle then
            if env.freestyle_state then
                comment = "完成加词" .. comment
            else
                comment = "开始加词" .. comment
            end
        end
        local tip_cand = Candidate("pin_tip", 0, #match, "", comment)
        tip_cand.quality = math.huge
        yield(tip_cand)
    end
end

return {
    pin_filter = pin_filter,
    pin_processor = pin_processor,
    panacea_translator = panacea_translator,
}

-- Local Variables:
-- lua-indent-level: 4
-- End: