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

-- Inline code keeps Pandoc's own \texttt escaping so that CJK characters stay
-- on the CJK font and backslashes survive verbatim. Earlier versions routed
-- inline code through hyperref's \nolinkurl, which silently dropped every CJK
-- glyph (the URL font is Latin-only) and rewrote Windows backslashes as
-- forward slashes. Break opportunities are injected between separators so long
-- paths and identifiers can still wrap instead of overflowing the text area.
local CODE_BREAK_AFTER = "[/\\\\:;,._-]"

function Code(inline)
  local text = inline.text
  if text == "" then
    return nil
  end

  local pieces = {}
  local chunk = {}
  for index = 1, #text do
    local char = text:sub(index, index)
    chunk[#chunk + 1] = char
    -- Break after a separator, but never after the final character and never
    -- between two separators, so "://" or ".." stay together.
    if char:match(CODE_BREAK_AFTER) and index < #text
        and not text:sub(index + 1, index + 1):match(CODE_BREAK_AFTER) then
      pieces[#pieces + 1] = table.concat(chunk)
      chunk = {}
    end
  end
  if #chunk > 0 then
    pieces[#pieces + 1] = table.concat(chunk)
  end
  if #pieces < 2 then
    return nil
  end

  local result = {}
  for index, piece in ipairs(pieces) do
    result[#result + 1] = pandoc.Code(piece, inline.attr)
    if index < #pieces then
      result[#result + 1] = pandoc.RawInline("latex", "\\allowbreak{}")
    end
  end
  return result
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
