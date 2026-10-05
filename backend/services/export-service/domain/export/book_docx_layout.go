package export

// bookDOCXLayoutFilter uses Pandoc's parsed headings and resolved identifiers.
// Unlike a TOC field, these links are populated before Word opens the file;
// page numbers remain native PAGE fields in the reference footer. No manuscript
// text, source path or executable is interpolated into this trusted filter.
const bookDOCXLayoutFilter = `
function Pandoc(doc)
  -- Bound only the display units. Paragraph boundaries replace existing line
  -- breaks; neither manuscript bytes nor executable artifact files are edited.
  doc = doc:walk({CodeBlock = function(block)
    local lines = {}
    for line in (block.text .. '\n'):gmatch('(.-)\n') do
      table.insert(lines, line)
    end
    if #lines <= 30 then return block end
    local units = {}
    local first = 1
    while first <= #lines do
      local last = math.min(first + 29, #lines)
      if last < #lines then
        -- Prefer an existing blank line, without interpreting source syntax.
        for candidate = last, first + 9, -1 do
          -- Pandoc drops a final newline in each CodeBlock; keep the blank
          -- line at the start of the following unit to preserve exact text.
          if lines[candidate] == '' then last = candidate - 1; break end
        end
      end
      local text = table.concat(lines, '\n', first, last)
      table.insert(units, pandoc.CodeBlock(text, block.attr))
      first = last + 1
    end
    return units
  end})
  doc = doc:walk({Inlines = function(inlines)
    local separated = {}
    local previous_note = false
    for _, inline in ipairs(inlines) do
      if previous_note and inline.t == 'Note' then
        table.insert(separated, pandoc.Superscript({pandoc.Str(',')}))
      end
      table.insert(separated, inline)
      previous_note = inline.t == 'Note'
    end
    return separated
  end})
  local title = doc.blocks[1]
  if not title or title.t ~= 'Header' or title.level ~= 1 then
    error('canonical book title is missing')
  end
  table.remove(doc.blocks, 1)
  local contents = {}
  doc:walk({Header = function(h)
    if h.level <= 2 and h.identifier ~= '' then
      local label = pandoc.Str(pandoc.utils.stringify(h.content))
      local link = pandoc.Link({label}, '#' .. h.identifier)
      table.insert(contents, pandoc.Div({pandoc.Para({link})},
        pandoc.Attr('', {}, {['custom-style'] = 'TOC ' .. h.level})))
    end
  end})
  local blocks = {
    pandoc.Div({pandoc.Para(title.content)},
      pandoc.Attr('', {}, {['custom-style'] = 'Title'}))
  }
  if #contents > 0 then
    table.insert(blocks, pandoc.Div({pandoc.Para({pandoc.Str('目录')})},
      pandoc.Attr('', {}, {['custom-style'] = 'TOC Heading'})))
    for _, block in ipairs(contents) do table.insert(blocks, block) end
    table.insert(blocks, pandoc.RawBlock('openxml',
      '<w:p><w:r><w:br w:type="page"/></w:r></w:p>'))
  end
  for _, block in ipairs(doc.blocks) do table.insert(blocks, block) end
  return pandoc.Pandoc(blocks, doc.meta)
end
`
