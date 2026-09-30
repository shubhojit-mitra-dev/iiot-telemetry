$toolsDir = "$PSScriptRoot\.tools"

# Add local MinGit to this session's PATH
$env:Path = "$toolsDir\mingit\cmd;" + $env:Path

# Point GitHub CLI to a local configuration folder instead of the global user folder
$env:GH_CONFIG_DIR = "$toolsDir\gh-config"

Write-Host "--------------------------------------------------------"
Write-Host "🚀 Local Project Environment Activated!" -ForegroundColor Green
Write-Host "--------------------------------------------------------"
Write-Host "1. Git is running portably from the .tools directory."
Write-Host "2. GitHub CLI (gh) authentication is sandboxed to this folder."
Write-Host "3. The laptop owner's global settings remain untouched."
Write-Host "--------------------------------------------------------"
Write-Host "NOTE: You must run '.\start-dev.ps1' every time you open a new terminal for this project." -ForegroundColor Yellow
