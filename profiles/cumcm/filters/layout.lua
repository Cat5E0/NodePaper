-- NodePaper CUMCM layout transformations.
--
-- Markdown contract:
--   # 附录
--   ## 测试数据
--   ## 程序代码
--
-- appendix.numbering selects alpha (default), continuous, or none. Code blocks
-- without a language are mapped to Pandoc's built-in text syntax so all code
-- uses the same breakable Highlighting environment.

local function stringify(value)
  if value == nil then
    return ""
  end
  return pandoc.utils.stringify(value)
end

local function is_appendix_header(block)
  return block.t == "Header"
    and block.level == 1
    and stringify(block.content):match("^%s*附录%s*$") ~= nil
end

local function add_unnumbered(header)
  if not header.classes:includes("unnumbered") then
    header.classes:insert("unnumbered")
  end
end

function Code(inline)
  -- xurl's nolinkurl permits safe line breaks in long inline code and Windows
  -- paths without interpreting the content as TeX commands.
  local text = inline.text:gsub("\\", "/")
  return pandoc.RawInline("latex", "\\nolinkurl{" .. text .. "}")
end

function CodeBlock(block)
  if #block.classes == 0 then
    block.classes:insert("text")
  end
  return block
end

function Pandoc(doc)
  local mode = stringify(doc.meta["nodepaper-appendix-numbering"])
  if mode == "" then
    mode = "alpha"
  end

  local appendix_index = nil
  for index, block in ipairs(doc.blocks) do
    if is_appendix_header(block) then
      appendix_index = index
      break
    end
  end
  if appendix_index == nil or mode == "continuous" then
    return doc
  end

  local output = pandoc.List()
  for index, block in ipairs(doc.blocks) do
    if index == appendix_index then
      add_unnumbered(block)
      output:insert(block)
      if mode == "alpha" then
        output:insert(pandoc.RawBlock("latex", "\\nodepaperAppendixAlpha"))
      else
        output:insert(pandoc.RawBlock("latex", "\\nodepaperAppendixNone"))
      end
    elseif index > appendix_index and block.t == "Header" then
      if mode == "alpha" and block.level > 1 then
        block.level = block.level - 1
      elseif mode == "none" then
        add_unnumbered(block)
      end
      output:insert(block)
    else
      output:insert(block)
    end
  end
  doc.blocks = output
  return doc
end
