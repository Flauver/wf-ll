-- LL Quick Code Filter
-- Copyright (c) 2023, 2024 ksqsf
-- Licensed under GPLv3
--
-- 功能：识别来自 LL_linglong 词库的候选词，在 comment 处添加 "⚡️"

local Top = {}

function Top.init(env)
   env.quick_code_indicator = env.engine.schema.config:get_string("ll/quick_code_indicator") or "✨"
   -- LL_linglong 词库的初始 quality 设置为 100000，用于识别来源
   env.linglong_quality_threshold = 100000
end

function Top.fini(env)
end

function Top.func(t_input, env)
   for cand in t_input:iter() do
      -- 检查候选词是否来自 LL_linglong 词库
      -- LL_linglong 词库的候选词有很高的初始 quality (100000)
      if cand.quality >= env.linglong_quality_threshold then
         -- 只在没有注释或注释为空时添加闪电标记
         if not cand.comment or cand.comment == "" then
            cand.comment = env.quick_code_indicator
         elseif not cand.comment:find(env.quick_code_indicator) then
            -- 如果已有注释但不包含闪电标记，则添加到注释前面
            cand.comment = env.quick_code_indicator .. " " .. cand.comment
         end
      end
      yield(cand)
   end
end

return Top