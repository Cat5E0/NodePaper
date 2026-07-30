-- Move the first level-one “摘要” section into Pandoc's abstract metadata.
-- The source Markdown remains unchanged; this filter only transforms the
-- in-memory document used for the CUMCM electronic-paper build.

local function is_abstract_header(block)
  return block.t == "Header"
    and block.level == 1
    and pandoc.utils.stringify(block.content):match("^%s*摘要%s*$") ~= nil
end

function Pandoc(doc)
  local start_index = nil
  for index, block in ipairs(doc.blocks) do
    if is_abstract_header(block) then
      start_index = index
      break
    end
  end

  if start_index == nil then
    return doc
  end

  local finish_index = #doc.blocks + 1
  for index = start_index + 1, #doc.blocks do
    local block = doc.blocks[index]
    if block.t == "Header" and block.level == 1 then
      finish_index = index
      break
    end
  end

  local abstract = pandoc.List()
  for index = start_index + 1, finish_index - 1 do
    abstract:insert(doc.blocks[index])
  end
  doc.meta.abstract = pandoc.MetaBlocks(abstract)

  local body = pandoc.List()
  for index, block in ipairs(doc.blocks) do
    if index < start_index or index >= finish_index then
      body:insert(block)
    end
  end
  doc.blocks = body
  return doc
end
