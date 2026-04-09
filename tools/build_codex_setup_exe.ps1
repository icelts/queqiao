param(
    [string]$PythonCommand = "python",
    [string]$Name = "queqiao-codex-setup"
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
$venvDir = Join-Path $projectRoot ".local\codex-setup-build-venv"
$entryScript = Join-Path $scriptDir "codex_setup_gui.py"
$distDir = Join-Path $projectRoot "dist"
$buildDir = Join-Path $projectRoot "build\pyinstaller-codex-setup"

if (-not (Test-Path $venvDir)) {
    & $PythonCommand -m venv $venvDir
}

$venvPython = Join-Path $venvDir "Scripts\python.exe"
if (-not (Test-Path $venvPython)) {
    throw "未找到虚拟环境 Python: $venvPython"
}

& $venvPython -m pip install --upgrade pip
& $venvPython -m pip install pyinstaller

if (-not (Test-Path $distDir)) {
    New-Item -ItemType Directory -Path $distDir -Force | Out-Null
}
if (-not (Test-Path $buildDir)) {
    New-Item -ItemType Directory -Path $buildDir -Force | Out-Null
}

& $venvPython -m PyInstaller `
    --noconfirm `
    --clean `
    --onefile `
    --windowed `
    --name $Name `
    --specpath $buildDir `
    --distpath $distDir `
    --workpath $buildDir `
    $entryScript

Write-Host "Build completed:" -ForegroundColor Green
Write-Host (Join-Path $distDir ($Name + ".exe"))
