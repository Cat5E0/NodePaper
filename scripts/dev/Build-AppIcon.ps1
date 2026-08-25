<#
.SYNOPSIS
Regenerates the Windows application icon from the logo artwork.

.DESCRIPTION
Produces two committed binaries from one source PNG:

  packaging/windows/nodepaper.ico   - used by Inno Setup (SetupIconFile) and
                                      shipped in the release payload
  cmd/nodepaper/rsrc_windows_amd64.syso
                                   - linked into nodepaper.exe by the Go
                                      linker, purely by filename convention.
                                      This is what actually gives the exe,
                                      Explorer, the taskbar, the shortcuts and
                                      Add/Remove Programs their icon.

Both were previously hand-made, with the procedure recorded only in a commit
message. That is how the shipped icon came to be built from the transparent
artwork - fully transparent background, black strokes, invisible on a dark
taskbar - without anyone being able to check. This script exists so the ico and
the syso can be re-derived and reviewed.

Requires Python with Pillow, and akavel/rsrc on PATH (go install
github.com/akavel/rsrc@latest). Pass -SkipSyso to build only the .ico.
#>
param(
    [string]$SourcePng = "docs/assets/logo/logo-white.png",
    [string]$IcoPath = "packaging/windows/nodepaper.ico",
    [string]$SysoPath = "cmd/nodepaper/rsrc_windows_amd64.syso",
    [string]$ManifestPath = "cmd/nodepaper/app.manifest",
    [string]$PythonExe = "python",
    [string]$RsrcExe = "rsrc",
    [switch]$SkipSyso
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
Set-Location $repoRoot

foreach ($required in @($SourcePng, $ManifestPath)) {
    if (-not (Test-Path -LiteralPath $required)) {
        throw "required input is missing: $required"
    }
}

# The frame set is the one the shipped icon has always had. Windows picks a
# frame by display size; dropping one makes it scale a neighbour instead.
#
# `thicken` is a MinFilter kernel applied to the 512px source before the
# downscale, and it is not optional. The artwork is a thin black outline on
# white: a plain LANCZOS downscale to 16px leaves the darkest pixel at 142/255
# - not one pixel below mid-grey - so the icon reads as a grey smudge on a
# white tile, which is the same wash-out the transparent version suffered. The
# kernels below were chosen by inspecting magnified frames: each is the largest
# that still keeps the page outline, the folded corner and the N distinct.
# Larger kernels blob the fold shut, smaller ones do not reach solid black.
$frames = @(
    @{ Size = 16;  Thicken = 15 },
    @{ Size = 24;  Thicken = 11 },
    @{ Size = 32;  Thicken = 9 },
    @{ Size = 48;  Thicken = 7 },
    @{ Size = 64;  Thicken = 5 },
    @{ Size = 128; Thicken = 3 },
    @{ Size = 256; Thicken = 1 }
)

$ladder = ($frames | ForEach-Object { "$($_.Size):$($_.Thicken)" }) -join ","

$python = @"
import io
import struct
import sys
from PIL import Image, ImageFilter

source, target, ladder = sys.argv[1], sys.argv[2], sys.argv[3]

src = Image.open(source).convert('RGBA')
if src.size[0] != src.size[1]:
    raise SystemExit('source artwork must be square, got %r' % (src.size,))
if min(pixel[3] for pixel in src.getdata()) != 255:
    raise SystemExit('source artwork must be fully opaque; a transparent icon '
                     'is what this script exists to replace')

frames = []
for pair in ladder.split(','):
    size, thicken = (int(part) for part in pair.split(':'))
    rgb = src.convert('RGB')
    if thicken > 1:
        rgb = rgb.filter(ImageFilter.MinFilter(thicken))
    frame = rgb.resize((size, size), Image.LANCZOS).convert('RGBA')
    darkest = min(frame.convert('L').getdata())
    if darkest > 96:
        raise SystemExit('%dpx frame never reaches a dark stroke (darkest=%d); '
                         'raise its thicken kernel' % (size, darkest))
    frames.append(frame)

# The ICO container is written by hand because Image.save(..., sizes=[...])
# rescales the one image it is handed, which would throw away the per-size
# thickening above and reintroduce the wash-out this script exists to fix.
# Each frame is stored as a PNG, which is what the previous icon used too.
payloads = []
for frame in frames:
    buffer = io.BytesIO()
    frame.save(buffer, format='PNG', optimize=True)
    payloads.append(buffer.getvalue())

header = struct.pack('<HHH', 0, 1, len(frames))
offset = len(header) + 16 * len(frames)
directory = b''
for frame, payload in zip(frames, payloads):
    width, height = frame.size
    directory += struct.pack('<BBBBHHII',
                             0 if width == 256 else width,
                             0 if height == 256 else height,
                             0, 0, 1, 32, len(payload), offset)
    offset += len(payload)

with open(target, 'wb') as handle:
    handle.write(header + directory + b''.join(payloads))

# Read it back and prove the frames are the ones that were written.
written = Image.open(target)
sizes = sorted(written.info['sizes'])
expected = sorted(frame.size for frame in frames)
if sizes != expected:
    raise SystemExit('ICO frames %r do not match the requested ladder %r'
                     % (sizes, expected))
for size in sizes:
    written.size = size
    probe = written.convert('RGBA')
    if probe.getpixel((0, 0))[3] != 255:
        raise SystemExit('%r frame is not opaque' % (size,))
print('wrote %s with %d frames: %s'
      % (target, len(sizes), ' '.join('%dx%d' % s for s in sizes)))
"@

$pythonFile = Join-Path ([System.IO.Path]::GetTempPath()) "nodepaper-appicon-$PID.py"
try {
    Set-Content -LiteralPath $pythonFile -Value $python -Encoding UTF8
    & $PythonExe $pythonFile $SourcePng $IcoPath $ladder
    if ($LASTEXITCODE -ne 0) { throw "icon generation failed with exit code $LASTEXITCODE" }
}
finally {
    Remove-Item -LiteralPath $pythonFile -ErrorAction SilentlyContinue
}

if ($SkipSyso) {
    Write-Host "Skipped the .syso; nodepaper.exe keeps its previous icon."
    return
}

& $RsrcExe -ico $IcoPath -manifest $ManifestPath -arch amd64 -o $SysoPath
if ($LASTEXITCODE -ne 0) { throw "rsrc failed with exit code $LASTEXITCODE" }

Write-Host "Wrote $IcoPath and $SysoPath."
Write-Host "Rebuild nodepaper.exe for the new icon to take effect."
