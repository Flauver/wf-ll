local function reverse_commit_processor(key, env)
  local engine = env.engine
  local context = engine.context
  local config = engine.schema.config

  local hotkey = config:get_string('reverse_commit/hotkey') or "Control+u"

  if key:repr() == hotkey then
    local last_input = context:get_property("_last_input")
    local last_commit_text = context:get_property("_last_commit_text")

    if last_input and #last_input > 0 and last_commit_text then
      -- 这里没法删除宿主应用里的文字，只恢复输入
      context:clear()
      context.input = last_input

      -- 清除记录
      context:set_property("_last_input", "")
      context:set_property("_last_commit_text", "")
      return 1 -- 吞掉这个按键
    end
  end

  return 2
end

local function reverse_commit_translator(translation, env)
  local context = env.engine.context
  if context.input and #context.input > 0 then
    context:set_property("_last_input", context.input)
  end
  return translation
end

local function reverse_commit_commit_notification(env, commit_text)
  local context = env.engine.context
  if commit_text and #commit_text > 0 then
    context:set_property("_last_commit_text", commit_text)
  end
end

return {
  processor = reverse_commit_processor,
  translator = reverse_commit_translator,
  commit_notification = reverse_commit_commit_notification,
}