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
--
-- Bibliography placement (nodepaper-bib-method):
--   Unset on the main build route, where Citeproc fills the ::: {#refs} div in
--   place and nothing here has to move. `nodepaper export` instead runs Pandoc
--   with --natbib or --biblatex, which leave that div empty and hand the
--   reference list to bibtex/biber -- a list that LaTeX only emits where
--   \bibliography / \printbibliography appears. Putting that command in the
--   export templates would drop it after $body$, i.e. behind the appendix, and
--   leave the 参考文献 heading standing over a blank page. Emitting it here
--   instead keeps the list directly under its own heading.

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

-- Citeproc renders its list into the ::: {#refs} div, so that div is where the
-- reference list belongs in every mode. It survives as an empty Div node here
-- even under --natbib/--biblatex, because Citeproc runs after the Lua filters,
-- and its identifier carries no language -- unlike the section title, which is
-- 参考文献 only by default. The title match stays as a fallback for projects
-- that write the heading without the div.
local function is_references_div(block)
  return block.t == "Div" and block.identifier == "refs"
end

-- Only the two modes `nodepaper export` can request. An unset or unknown value
-- yields nil, which leaves every block below untouched.
--
-- Both packages draw a section heading of their own. When the list is placed
-- under the document's own references heading that heading is a duplicate, so
-- the anchored form suppresses it -- \bibsection is natbib's heading hook, and
-- renewing it inside a group keeps the change from leaking. The trailing form
-- is used when the document has no references heading at all, and there the
-- package-drawn heading is the only one, so it is kept.
--
-- nocite carries the citations a Citeproc build would resolve silently from
-- the YAML `nocite:` field. --natbib and --biblatex do not read that field, so
-- without help here a nocite-only project (no inline [@key] anywhere) exports
-- a .tex that contains no \citation command at all, and bibtex/biber then
-- fails with "I found no \citation commands". nocite_block turns each Cite
-- node the Citeproc route would have honoured into a \nocite{key} raw block,
-- emitted immediately before \bibliography/\printbibliography so it lands
-- inside the same references scope. It runs only on the two export routes:
-- the Citeproc build resolves nocite itself and must stay untouched.
local BIB_COMMANDS = {
  natbib = {
    anchored = "\\begingroup\\renewcommand{\\bibsection}{}\\bibliography{references}\\endgroup",
    trailing = "\\bibliography{references}",
  },
  biblatex = {
    anchored = "\\printbibliography[heading=none]",
    trailing = "\\printbibliography",
  },
}

-- nocite_keys walks the `nocite` metadata the same way Citeproc does: it keeps
-- only Cite nodes (the form `@key` parses to) and ignores bare strings such
-- as `'*'` or a key written without `@`, which Citeproc itself does not treat as
-- a nocite entry either. In Pandoc's Lua API a `MetaInlines` value is an Inlines
-- list and a `MetaList` value is a list of MetaValues, so the two forms are
-- walked uniformly by recursing one level. Duplicate keys are removed once, in
-- first-seen order, so the exported .aux lists each entry exactly once.
local function collect_cite_ids(node, seen, keys)
  if node == nil or type(node) ~= "table" and type(node) ~= "userdata" then
    return
  end
  if node.t == "Cite" and node.citations ~= nil then
    for _, citation in ipairs(node.citations) do
      local id = citation.id
      if id ~= nil and id ~= "" and not seen[id] then
        seen[id] = true
        table.insert(keys, id)
      end
    end
    return
  end
  for _, child in ipairs(node) do
    collect_cite_ids(child, seen, keys)
  end
end

local function nocite_keys(meta)
  if meta == nil then
    return {}
  end
  local seen = {}
  local keys = {}
  collect_cite_ids(meta, seen, keys)
  return keys
end

-- has_inline_citation reports whether the body carries a real inline citation.
-- On the export routes this decides whether emitting any bibliography machinery
-- is worth it: a paper that cites nothing and nocites nothing gives bibtex and
-- biber no work, and emitting \bibliography regardless made the compile chain
-- in the export's own README.txt fail on a legitimate project - C063 keeps its
-- five references as a hand-typed numbered list, which cannot be cited because
-- there are no bib entries to cite. That is the author's choice, not an error,
-- so the export follows it instead of insisting on a bibliography pass.
local function has_inline_citation(blocks)
  local found = false
  for _, block in ipairs(blocks) do
    if found then
      break
    end
    pandoc.walk_block(block, {
      Cite = function(_)
        found = true
      end,
    })
  end
  return found
end

local function nocite_block(keys)
  if #keys == 0 then
    return nil
  end
  return pandoc.RawBlock("latex", "\\nocite{" .. table.concat(keys, ",") .. "}")
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

-- Pandoc decides whether pipe-table columns are natural-width or fixed-width
-- from its output wrapping threshold. That makes a harmless Markdown edit able
-- to stretch a compact table across the page. NodePaper table attributes turn
-- the choice into an explicit author contract while keeping ordinary tables in
-- Markdown:
--
--   : Caption {#tbl:sample width=auto}
--   : Caption {#tbl:sample width=80%}
--   : Caption {#tbl:sample width=full}
--   : Caption {#tbl:sample width=80% ratios="1:4"}
--
-- Percentage widths use an explicit ratios attribute when present, otherwise
-- preserve Pandoc's parsed relative column ratios. When neither is available,
-- columns share the requested width equally. The explicit form matters because
-- Pandoc discards pipe-table separator lengths for short, natural-width tables.
local function requested_table_width(value)
  value = value:lower():gsub("^%s+", ""):gsub("%s+$", "")
  if value == "auto" then
    return "auto"
  end
  if value == "full" then
    return 1
  end
  local percent = tonumber(value:match("^(%d+%.?%d*)%%$"))
  if percent ~= nil and percent > 0 and percent <= 100 then
    return percent / 100
  end
  return nil
end

local function requested_table_ratios(value, column_count)
  local ratios = {}
  local total = 0
  for item in value:gmatch("[^,:]+") do
    local ratio = tonumber(item:match("^%s*(.-)%s*$"))
    if ratio == nil or ratio <= 0 then
      return nil, nil
    end
    ratios[#ratios + 1] = ratio
    total = total + ratio
  end
  if #ratios ~= column_count or total <= 0 then
    return nil, nil
  end
  return ratios, total
end

function Table(tbl)
  local value = tbl.attributes.width
  if value == nil then
    if tbl.attributes.ratios ~= nil then
      error("table ratios requires width=full or a percentage width")
    end
    return nil
  end

  local requested = requested_table_width(value)
  if requested == nil then
    error("table width must be auto, full, or a percentage greater than 0% and at most 100%: " .. value)
  end

  local specs = {}
  if requested == "auto" then
    if tbl.attributes.ratios ~= nil then
      error("table ratios cannot be combined with width=auto")
    end
    for index, spec in ipairs(tbl.colspecs) do
      specs[index] = { spec[1] }
    end
  else
    local explicit_ratios = nil
    local explicit_total = nil
    if tbl.attributes.ratios ~= nil then
      explicit_ratios, explicit_total = requested_table_ratios(tbl.attributes.ratios, #tbl.colspecs)
      if explicit_ratios == nil then
        error("table ratios must contain one positive number per column, separated by ':' or ','")
      end
    end

    local parsed_total = 0
    local parsed_count = 0
    for _, spec in ipairs(tbl.colspecs) do
      if type(spec[2]) == "number" and spec[2] > 0 then
        parsed_total = parsed_total + spec[2]
        parsed_count = parsed_count + 1
      end
    end

    for index, spec in ipairs(tbl.colspecs) do
      local ratio
      if explicit_ratios ~= nil then
        ratio = explicit_ratios[index] / explicit_total
      elseif parsed_count == #tbl.colspecs and parsed_total > 0 then
        ratio = spec[2] / parsed_total
      else
        ratio = 1 / #tbl.colspecs
      end
      specs[index] = { spec[1], requested * ratio }
    end
  end

  tbl.colspecs = specs
  tbl.attributes.width = nil
  tbl.attributes.ratios = nil
  return tbl
end

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

  local bib_commands = BIB_COMMANDS[stringify(doc.meta["nodepaper-bib-method"])]
  -- Nothing to cite means nothing for BibTeX or biber to do. Dropping the
  -- bibliography command here also drops the bibtex/biber step from the chain
  -- the Go side writes into README.txt, so the exported project compiles with
  -- the commands it documents.
  local nocite_list = {}
  if bib_commands ~= nil then
    nocite_list = nocite_keys(doc.meta["nocite"])
    if #nocite_list == 0 and not has_inline_citation(doc.blocks) then
      bib_commands = nil
    end
  end

  local appendix_index = nil
  local references_index = nil
  local references_div_index = nil
  for index, block in ipairs(doc.blocks) do
    if is_appendix_header(block) then
      appendix_index = index
    end
    if is_references_header(block, references_title) then
      references_index = index
    end
    if is_references_div(block) then
      references_div_index = index
    end
  end
  if bib_commands == nil and appendix_index == nil and references_index == nil then
    return doc
  end

  -- The div sits directly under the heading, so anchoring on it puts the list
  -- exactly where Citeproc would have put it. A document with neither anchor
  -- still gets its list, at the end, rather than silently losing it.
  local bib_index = references_div_index or references_index
  local bib_command = nil
  if bib_commands ~= nil then
    bib_command = bib_index ~= nil and bib_commands.anchored or bib_commands.trailing
  end
  -- natbib/biblatex do not read Pandoc's `nocite` metadata, so a nocite-only
  -- project would export without a single \citation command. The raw block
  -- is emitted just before \bibliography/\printbibliography so it shares the
  -- same references scope; on the Citeproc route bib_commands is nil and this
  -- stays a no-op, leaving the build's behaviour untouched.
  local nocite_raw = nil
  if bib_commands ~= nil then
    nocite_raw = nocite_block(nocite_list)
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
    if bib_command ~= nil and index == bib_index then
      if nocite_raw ~= nil then
        output:insert(nocite_raw)
      end
      output:insert(pandoc.RawBlock("latex", bib_command))
    end
  end
  if bib_command ~= nil and bib_index == nil then
    if nocite_raw ~= nil then
      output:insert(nocite_raw)
    end
    output:insert(pandoc.RawBlock("latex", bib_command))
  end
  doc.blocks = output
  return doc
end
