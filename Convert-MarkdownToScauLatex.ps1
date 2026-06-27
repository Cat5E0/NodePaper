param(
    [Parameter(Mandatory = $true)]
    [Alias("Input")]
    [string]$MarkdownPath,

    [string]$Output = ".\build\Paper.tex",

    [ValidateSet("thesis", "assignment", "experiment")]
    [string]$TemplateName = "thesis",

    [string]$Template = "",

    [string]$BuildDirectory = ".\build",

    [switch]$AllowSystemPandoc,

    [string]$CoverPdf = "",

    [string]$LastPagePdf = "",

    [string]$CoverLastPdf = ""
)

$ErrorActionPreference = "Stop"

function Resolve-ProjectPath {
    param([string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot $Path))
}

function Find-Executable {
    param(
        [string]$ProjectRelativePath,
        [string]$CommandName,
        [switch]$AllowSystem
    )

    $projectPath = Join-Path $PSScriptRoot $ProjectRelativePath
    if (Test-Path -LiteralPath $projectPath -PathType Leaf) {
        return (Resolve-Path -LiteralPath $projectPath).Path
    }

    if ($AllowSystem) {
        $cmd = Get-Command $CommandName -ErrorAction SilentlyContinue
        if ($cmd) {
            return $cmd.Source
        }
    }

    throw "Missing $CommandName. Expected project tool at '$projectPath'. Run .\Bootstrap-Tools.ps1 or pass -AllowSystemPandoc for development."
}

function Invoke-Checked {
    param(
        [string]$FilePath,
        [string[]]$Arguments
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
    }
}

function Get-MarkdownWithoutYaml {
    param([string]$Text)

    if ($Text -match "(?s)\A---\s*\r?\n.*?\r?\n---\s*(\r?\n|$)") {
        return $Text.Substring($Matches[0].Length)
    }
    return $Text
}

function Get-MarkdownDocumentParts {
    param([string]$Text)

    $lines = $Text -split "\r?\n", -1
    $metadataBlocks = New-Object System.Collections.Generic.List[string]
    $bodyLines = New-Object System.Collections.Generic.List[string]
    $i = 0

    while ($i -lt $lines.Count) {
        if ($lines[$i] -match "^\s*---\s*$") {
            $end = $i + 1
            while ($end -lt $lines.Count -and $lines[$end] -notmatch "^\s*---\s*$") {
                $end++
            }

            if ($end -lt $lines.Count) {
                $blockLines = @()
                if ($end -gt ($i + 1)) {
                    $blockLines = $lines[($i + 1)..($end - 1)]
                }
                $blockText = ($blockLines -join "`n").Trim()

                if ($blockText -match "(?m)^\s*[A-Za-z_][A-Za-z0-9_]*\s*:") {
                    $metadataBlocks.Add($blockText)
                    $i = $end + 1
                    continue
                }
            }
        }

        $bodyLines.Add($lines[$i])
        $i++
    }

    $bodyText = ($bodyLines -join "`n")
    $bodyText = [regex]::Replace($bodyText, "\A\s*#\s*(\r?\n)+", "")

    [pscustomobject]@{
        MetadataYaml = ($metadataBlocks -join "`n")
        Body = $bodyText
    }
}

function Split-MarkdownContent {
    param([string]$Markdown)

    $referencesHeading = -join ([char[]](0x53C2, 0x8003, 0x6587, 0x732E))
    $acknowledgementsHeading = -join ([char[]](0x81F4, 0x8C22))
    $lines = $Markdown -split "`r?`n"
    $body = New-Object System.Collections.Generic.List[string]
    $refs = New-Object System.Collections.Generic.List[string]
    $ack = New-Object System.Collections.Generic.List[string]
    $mode = "body"

    foreach ($line in $lines) {
        if ($line -match "^\s{0,3}#{1,6}\s+($referencesHeading|References)\s*$") {
            $mode = "refs"
            continue
        }
        if ($line -match "^\s{0,3}#{1,6}\s+($acknowledgementsHeading|Acknowledgements?)\s*$") {
            $mode = "ack"
            continue
        }

        switch ($mode) {
            "body" { $body.Add($line) }
            "refs" { $refs.Add($line) }
            "ack" { $ack.Add($line) }
        }
    }

    [pscustomobject]@{
        Body = ($body -join "`n").Trim()
        References = ($refs -join "`n").Trim()
        Acknowledgements = ($ack -join "`n").Trim()
    }
}

function ConvertFrom-PandocMetaValue {
    param($Value)

    if ($null -eq $Value) {
        return $null
    }

    if ($Value.PSObject.Properties.Name -contains "t") {
        switch ($Value.t) {
            "MetaString" { return [string]$Value.c }
            "MetaBool" { return [bool]$Value.c }
            "MetaInlines" { return (($Value.c | ForEach-Object { ConvertFrom-PandocInline $_ }) -join "").Trim() }
            "MetaBlocks" { return (($Value.c | ForEach-Object { ConvertFrom-PandocBlock $_ }) -join "`n").Trim() }
            "MetaList" { return @($Value.c | ForEach-Object { ConvertFrom-PandocMetaValue $_ }) }
            "MetaMap" {
                $map = @{}
                foreach ($prop in $Value.c.PSObject.Properties) {
                    $map[$prop.Name] = ConvertFrom-PandocMetaValue $prop.Value
                }
                return $map
            }
            default { return $Value.c }
        }
    }

    return $Value
}

function ConvertFrom-PandocInline {
    param($Inline)

    if ($null -eq $Inline) {
        return ""
    }

    switch ($Inline.t) {
        "Str" { return [string]$Inline.c }
        "Space" { return " " }
        "SoftBreak" { return "`n" }
        "LineBreak" { return "`n" }
        "Code" { return [string]$Inline.c[1] }
        "Math" { return ('$' + [string]$Inline.c[1] + '$') }
        "RawInline" { return [string]$Inline.c[1] }
        default {
            if ($Inline.c -is [System.Array]) {
                return (($Inline.c | ForEach-Object {
                    if ($_.PSObject.Properties.Name -contains "t") {
                        ConvertFrom-PandocInline $_
                    }
                }) -join "")
            }
            return ""
        }
    }
}

function ConvertFrom-PandocBlock {
    param($Block)

    if ($null -eq $Block) {
        return ""
    }

    switch ($Block.t) {
        "Plain" { return (($Block.c | ForEach-Object { ConvertFrom-PandocInline $_ }) -join "") }
        "Para" { return (($Block.c | ForEach-Object { ConvertFrom-PandocInline $_ }) -join "") }
        "RawBlock" { return [string]$Block.c[1] }
        default { return "" }
    }
}

function Read-PandocMetadata {
    param(
        [string]$PandocPath,
        [string]$MetadataYaml,
        [string]$BuildDir
    )

    if ([string]::IsNullOrWhiteSpace($MetadataYaml)) {
        return @{}
    }

    $metadataInputPath = Join-Path $BuildDir "metadata.md"
    $jsonPath = Join-Path $BuildDir "metadata.json"
    $metadataMarkdown = "---`n$($MetadataYaml.Trim())`n---`n"
    [System.IO.File]::WriteAllText($metadataInputPath, $metadataMarkdown, [System.Text.UTF8Encoding]::new($false))

    Invoke-Checked $PandocPath @(
        $metadataInputPath,
        "--from", "markdown+yaml_metadata_block+tex_math_dollars+raw_tex+link_attributes+implicit_figures+pipe_tables+fenced_divs",
        "--to", "json",
        "--output", $jsonPath
    )

    $doc = Get-Content -LiteralPath $jsonPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $metadata = @{}
    foreach ($prop in $doc.meta.PSObject.Properties) {
        $metadata[$prop.Name] = ConvertFrom-PandocMetaValue $prop.Value
    }
    return $metadata
}

function Convert-MarkdownSnippetToLatex {
    param(
        [string]$Text,
        [string]$PandocPath,
        [string]$BuildDir,
        [string]$Name
    )

    if ([string]::IsNullOrWhiteSpace($Text)) {
        return ""
    }

    $inputPath = Join-Path $BuildDir "$Name.md"
    $outputPath = Join-Path $BuildDir "$Name.tex"
    [System.IO.File]::WriteAllText($inputPath, $Text.Trim() + "`n", [System.Text.UTF8Encoding]::new($false))
    Invoke-Checked $PandocPath @(
        $inputPath,
        "--from", "markdown+tex_math_dollars+raw_tex+link_attributes+implicit_figures+pipe_tables+fenced_divs",
        "--to", "latex",
        "--output", $outputPath
    )
    return (Get-Content -LiteralPath $outputPath -Raw -Encoding UTF8).Trim()
}

function Escape-LatexText {
    param($Value)

    if ($null -eq $Value) {
        return ""
    }

    $text = [string]$Value
    $text = $text.Replace('\', '\textbackslash{}')
    $text = $text.Replace('&', '\&')
    $text = $text.Replace('%', '\%')
    $text = $text.Replace('$', '\$')
    $text = $text.Replace('#', '\#')
    $text = $text.Replace('_', '\_')
    $text = $text.Replace('{', '\{')
    $text = $text.Replace('}', '\}')
    $text = $text.Replace('~', '\textasciitilde{}')
    $text = $text.Replace('^', '\textasciicircum{}')
    return $text
}

function Get-MetaString {
    param(
        [hashtable]$Metadata,
        [string]$Name,
        [string]$Default = ""
    )

    if ($Metadata.ContainsKey($Name) -and $null -ne $Metadata[$Name]) {
        return [string]$Metadata[$Name]
    }
    return $Default
}

function Join-Keywords {
    param($Value)

    if ($null -eq $Value) {
        return ""
    }
    if ($Value -is [System.Array]) {
        return (($Value | ForEach-Object { Escape-LatexText $_ }) -join "; ")
    }
    return (Escape-LatexText $Value)
}

function Replace-Placeholder {
    param(
        [string]$TemplateText,
        [string]$Name,
        [string]$Value
    )
    return $TemplateText.Replace(("%%__{0}__" -f $Name), $Value)
}

$inputPath = Resolve-ProjectPath $MarkdownPath
$outputPath = Resolve-ProjectPath $Output
if ([string]::IsNullOrWhiteSpace($Template)) {
    if ($TemplateName -eq "assignment") {
        $Template = ".\templates\scau-assignment.template.tex"
    }
    elseif ($TemplateName -eq "experiment") {
        $Template = ".\templates\scau-experiment.template.tex"
    }
    else {
        $Template = ".\templates\scau-thesis.template.tex"
    }
}
$templatePath = Resolve-ProjectPath $Template
$buildDir = Resolve-ProjectPath $BuildDirectory

if (-not (Test-Path -LiteralPath $inputPath -PathType Leaf)) {
    throw "Input Markdown not found: $inputPath"
}
if (-not (Test-Path -LiteralPath $templatePath -PathType Leaf)) {
    throw "Template not found: $templatePath"
}

New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path ([System.IO.Path]::GetDirectoryName($outputPath)) | Out-Null

$pandoc = Find-Executable "tools\windows-x64\pandoc\pandoc.exe" "pandoc.exe" -AllowSystem:$AllowSystemPandoc
$crossref = Find-Executable "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe" "pandoc-crossref.exe" -AllowSystem:$AllowSystemPandoc
$luaFilter = Join-Path $PSScriptRoot "filters\scau-blocks.lua"
if (-not (Test-Path -LiteralPath $luaFilter -PathType Leaf)) {
    throw "Lua filter not found: $luaFilter"
}

$markdownText = [System.IO.File]::ReadAllText($inputPath, [System.Text.Encoding]::UTF8)
$documentParts = Get-MarkdownDocumentParts $markdownText
$metadata = Read-PandocMetadata $pandoc $documentParts.MetadataYaml $buildDir
$content = Split-MarkdownContent $documentParts.Body
$inputDir = [System.IO.Path]::GetDirectoryName($inputPath)
$resourcePath = @(
    $PSScriptRoot,
    (Join-Path $PSScriptRoot "image"),
    $inputDir,
    (Join-Path $inputDir "image")
) -join ";"

$bodyInput = Join-Path $buildDir "body.md"
$bodyOutput = Join-Path $buildDir "body.tex"
[System.IO.File]::WriteAllText($bodyInput, $content.Body + "`n", [System.Text.UTF8Encoding]::new($false))

Invoke-Checked $pandoc @(
    $bodyInput,
    "--from", "markdown+yaml_metadata_block+tex_math_dollars+raw_tex+link_attributes+implicit_figures+pipe_tables+fenced_divs",
    "--to", "latex",
    "--top-level-division=section",
    "--filter", $crossref,
    "--lua-filter", $luaFilter,
        "--metadata", "cref=True",
        "--metadata", "linkReferences=True",
        "--syntax-highlighting", "tango",
        "--resource-path", $resourcePath,
        "--output", $bodyOutput
)

$bodyLatex = Get-Content -LiteralPath $bodyOutput -Raw -Encoding UTF8

$referencesLatex = ""
$hasReferences = $true
if ($metadata.ContainsKey("references_tex") -and -not [string]::IsNullOrWhiteSpace([string]$metadata["references_tex"])) {
    $referencesLatex = [string]$metadata["references_tex"]
}
elseif (-not [string]::IsNullOrWhiteSpace($content.References)) {
    $referencesLatex = Convert-MarkdownSnippetToLatex $content.References $pandoc $buildDir "references"
}
else {
    $hasReferences = $false
    $referencesLatex = "\bibitem{no-reference} No references provided."
}

$referencesSectionLatex = ""
if ($hasReferences) {
    $referencesTitle = -join ([char[]](0x53C2, 0x8003, 0x6587, 0x732E))
    $referencesSectionLatex = @"
\clearpage
\phantomsection
\addcontentsline{toc}{section}{$referencesTitle}
\centerline{\zihao{4}\heiti $referencesTitle}
\vspace{1em}
{\zihao{-4}\songti\rmfamily
\begin{thebibliography}{0}
$($referencesLatex.Trim())
\end{thebibliography}
\par}
"@
}

$ackText = Get-MetaString $metadata "acknowledgements" ""
if ([string]::IsNullOrWhiteSpace($ackText) -and -not [string]::IsNullOrWhiteSpace($content.Acknowledgements)) {
    $ackText = $content.Acknowledgements
}

$abstractZh = Convert-MarkdownSnippetToLatex (Get-MetaString $metadata "abstract_zh" "") $pandoc $buildDir "abstract_zh"
$abstractEn = Convert-MarkdownSnippetToLatex (Get-MetaString $metadata "abstract_en" "") $pandoc $buildDir "abstract_en"
$ackLatex = Convert-MarkdownSnippetToLatex $ackText $pandoc $buildDir "acknowledgements"

$templateText = Get-Content -LiteralPath $templatePath -Raw -Encoding UTF8
$replacements = @{
    "TITLE_ZH" = Escape-LatexText (Get-MetaString $metadata "title_zh" "Thesis title")
    "TITLE_EN" = Escape-LatexText (Get-MetaString $metadata "title_en" "Thesis Title")
    "AUTHOR_ZH" = Escape-LatexText (Get-MetaString $metadata "author_zh" "Author")
    "AUTHOR_EN" = Escape-LatexText (Get-MetaString $metadata "author_en" "Author")
    "COLLEGE" = Escape-LatexText (Get-MetaString $metadata "college" "College")
    "MAJOR" = Escape-LatexText (Get-MetaString $metadata "major" "Major")
    "STUDENT_ID" = Escape-LatexText (Get-MetaString $metadata "student_id" "Student ID")
    "SUPERVISOR" = Escape-LatexText (Get-MetaString $metadata "supervisor" "Supervisor")
    "SUPERVISOR_TITLE" = Escape-LatexText (Get-MetaString $metadata "supervisor_title" "Title")
    "TEACHER" = Escape-LatexText (Get-MetaString $metadata "teacher" (Get-MetaString $metadata "supervisor" "Teacher"))
    "COURSE" = Escape-LatexText (Get-MetaString $metadata "course" "Course")
    "ASSIGNMENT_TYPE" = Escape-LatexText (Get-MetaString $metadata "assignment_type" "Course Assignment")
    "DATE" = Escape-LatexText (Get-MetaString $metadata "date" "")
    "ABSTRACT_ZH" = $abstractZh
    "KEYWORDS_ZH" = Join-Keywords $metadata["keywords_zh"]
    "ABSTRACT_EN" = $abstractEn
    "KEYWORDS_EN" = Join-Keywords $metadata["keywords_en"]
    "BODY" = $bodyLatex.Trim()
    "REFERENCES" = $referencesLatex.Trim()
    "REFERENCES_SECTION" = $referencesSectionLatex.Trim()
    "ACKNOWLEDGEMENTS" = $ackLatex.Trim()
}

foreach ($key in $replacements.Keys) {
    $templateText = Replace-Placeholder $templateText $key ([string]$replacements[$key])
}

# Handle %%__TEACHER_LINE__: show teacher row only when teacher is non-empty
$teacherValue = $metadata["teacher"]
$teacherEscaped = Escape-LatexText (Get-MetaString $metadata "teacher" "")
if ([string]::IsNullOrWhiteSpace($teacherEscaped)) {
    $templateText = Replace-Placeholder $templateText "TEACHER_LINE" ""
}
else {
    $teacherLine = "        \makebox[4em][s]{任课教师}： & \underline{\makebox[9.5cm]{$teacherEscaped}} \\"
    $templateText = Replace-Placeholder $templateText "TEACHER_LINE" $teacherLine
}

# Handle %%__COVER_PDF__ / %%__LAST_PAGE_PDF__ placeholders
# Priority: CoverLastPdf overrides CoverPdf + LastPagePdf
$coverLatex = ""
$lastPageLatex = ""

if ($CoverLastPdf) {
    $coverLastAbs = Resolve-ProjectPath $CoverLastPdf
    if (-not (Test-Path -LiteralPath $coverLastAbs -PathType Leaf)) {
        throw "CoverLastPdf not found: $coverLastAbs"
    }
    # pages={1,last} — first page as cover, last page as last page
    $coverLatex = "\clearpage`n\thispagestyle{empty}`n\includepdf[pages={1}]{$coverLastAbs}"
    $lastPageLatex = "\clearpage`n\thispagestyle{empty}`n\includepdf[pages={last}]{$coverLastAbs}"
}
else {
    if ($CoverPdf) {
        $coverAbs = Resolve-ProjectPath $CoverPdf
        if (-not (Test-Path -LiteralPath $coverAbs -PathType Leaf)) {
            throw "CoverPdf not found: $coverAbs"
        }
        $coverLatex = "\clearpage`n\thispagestyle{empty}`n\includepdf{{$coverAbs}}"
    }
    if ($LastPagePdf) {
        $lastAbs = Resolve-ProjectPath $LastPagePdf
        if (-not (Test-Path -LiteralPath $lastAbs -PathType Leaf)) {
            throw "LastPagePdf not found: $lastAbs"
        }
        $lastPageLatex = "\clearpage`n\thispagestyle{empty}`n\includepdf{{$lastAbs}}"
    }
}

$templateText = Replace-Placeholder $templateText "COVER_PDF" $coverLatex
$templateText = Replace-Placeholder $templateText "LAST_PAGE_PDF" $lastPageLatex

$unresolved = [regex]::Matches($templateText, "%%__[A-Z0-9_]+__") | ForEach-Object { $_.Value } | Sort-Object -Unique
if ($unresolved.Count -gt 0) {
    throw "Unresolved template placeholders: $($unresolved -join ', ')"
}

[System.IO.File]::WriteAllText($outputPath, $templateText, [System.Text.UTF8Encoding]::new($false))
Write-Host "Generated LaTeX: $outputPath"
