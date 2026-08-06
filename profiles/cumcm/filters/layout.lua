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
--
-- Page breaks:
--   - The references section (reference-section-title, default 参考文献)
--     always starts on a fresh page.
--   - The retained 附录 heading starts on a fresh page unless
--     nodepaper-appendix-newpage is false (appendix.newPage: false).

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

local function is_references_header(block, title)
  return block.t == "Header"
    and block.level == 1
    and stringify(block.content) == title
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

  local references_title = stringify(doc.meta["reference-section-title"])
  if references_title == "" then
    references_title = "参考文献"
  end

  local appendix_new_page = true
  if doc.meta["nodepaper-appendix-newpage"] ~= nil then
    appendix_new_page = doc.meta["nodepaper-appendix-newpage"] ~= false
  end

  local appendix_index = nil
  local references_index = nil
  for index, block in ipairs(doc.blocks) do
    if is_appendix_header(block) then
      appendix_index = index
    end
    if is_references_header(block, references_title) then
      references_index = index
    end
  end
  if appendix_index == nil and references_index == nil then
    return doc
  end

  local output = pandoc.List()
  for index, block in ipairs(doc.blocks) do
    if index == references_index then
      output:insert(pandoc.RawBlock("latex", "\\clearpage"))
      output:insert(block)
    elseif index == appendix_index then
      if appendix_new_page then
        output:insert(pandoc.RawBlock("latex", "\\clearpage"))
      end
      if mode ~= "continuous" then
        add_unnumbered(block)
      end
      output:insert(block)
      if mode == "alpha" then
        output:insert(pandoc.RawBlock("latex", "\\nodepaperAppendixAlpha"))
      elseif mode == "none" then
        output:insert(pandoc.RawBlock("latex", "\\nodepaperAppendixNone"))
      end
    elseif appendix_index ~= nil and index > appendix_index and block.t == "Header" then
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
