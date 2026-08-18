local block_environments = {
  theorem = "theorem",
  thm = "theorem",
  lemma = "lemma",
  lem = "lemma",
  remark = "remark",
  proof = "proof"
}

local function has_class(classes, class)
  for _, value in ipairs(classes) do
    if value == class then
      return true
    end
  end
  return false
end

local function latex_blocks_for_environment(env, blocks)
  local inner = pandoc.write(pandoc.Pandoc(blocks), "latex")
  return {
    pandoc.RawBlock("latex", "\\begin{" .. env .. "}"),
    pandoc.RawBlock("latex", inner),
    pandoc.RawBlock("latex", "\\end{" .. env .. "}")
  }
end

function Div(el)
  for class, env in pairs(block_environments) do
    if has_class(el.classes, class) then
      return latex_blocks_for_environment(env, el.content)
    end
  end
  return nil
end
