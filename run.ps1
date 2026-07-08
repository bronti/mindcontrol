# run.ps1 — dev runner with auto-restart.
#
# Launch this once instead of `go run .`:
#     ./run.ps1
# It builds the bot, runs it, and watches the Go source. On any saved change
# (which is what happens right before I push) it rebuilds and restarts the bot,
# so the running instance is always the latest code. Stop everything with Ctrl+C.
#
# Only ONE instance may talk to Telegram at a time, so don't also run `go run .`
# in another terminal — you'd get "Conflict: terminated by other getUpdates".

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$exe = Join-Path $root 'bin\dev-bot.exe'   # gitignored (see .gitignore: /bin/, *.exe)
$proc = $null

function Stop-Bot {
    if ($script:proc -and -not $script:proc.HasExited) {
        Write-Host "-> stopping bot (pid $($script:proc.Id))" -ForegroundColor DarkGray
        Stop-Process -Id $script:proc.Id -Force -ErrorAction SilentlyContinue
        $script:proc.WaitForExit()
    }
    $script:proc = $null
}

function Start-Bot {
    Write-Host "-> building..." -ForegroundColor Cyan
    & go build -o $exe .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "x build failed - leaving the bot down until the next save" -ForegroundColor Red
        return
    }
    Write-Host "-> starting bot" -ForegroundColor Green
    # Run from the repo root so the bot finds .env, google-cloud-key.json, settings.json.
    $script:proc = Start-Process -FilePath $exe -WorkingDirectory $root -NoNewWindow -PassThru
}

function Latest-GoWrite {
    (Get-ChildItem -Path $root -Filter *.go -Recurse |
        Measure-Object -Property LastWriteTimeUtc -Maximum).Maximum
}

try {
    New-Item -ItemType Directory -Force (Split-Path $exe) | Out-Null
    Start-Bot
    $last = Latest-GoWrite
    Write-Host "watching *.go - save a change to trigger a restart (Ctrl+C to quit)" -ForegroundColor DarkGray
    while ($true) {
        Start-Sleep -Seconds 1
        $now = Latest-GoWrite
        if ($now -ne $last) {
            $last = $now
            Write-Host "`n* change detected - restarting" -ForegroundColor Yellow
            Stop-Bot
            Start-Bot
        }
    }
}
finally {
    Stop-Bot
    Write-Host "bot stopped." -ForegroundColor DarkGray
}
