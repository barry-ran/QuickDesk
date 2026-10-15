# Spawn a skill binary exactly the way quickdesk-skill-host does - with real
# pipes, not file redirection - and report what it does with an MCP initialize.
#
#   guest-skill-spawn-probe.ps1 -Exe "C:\Program Files\QuickDesk\skills\file-ops\file-ops.exe"
param(
    [string]$Exe = 'C:\Program Files\QuickDesk\skills\file-ops\file-ops.exe'
)

$ErrorActionPreference = 'Stop'

function Write-Result {
    param($Object)
    $json = $Object | ConvertTo-Json -Depth 6 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    Write-Output ('PDJSON_B64:' + [Convert]::ToBase64String($bytes))
}

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $Exe
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true

$p = [System.Diagnostics.Process]::Start($psi)

$initialize = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"probe","version":"1"}}}'
$toolsList = '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
$p.StandardInput.WriteLine($initialize)
$p.StandardInput.Flush()
Start-Sleep -Milliseconds 800
$p.StandardInput.WriteLine($toolsList)
$p.StandardInput.Flush()
Start-Sleep -Milliseconds 1500

$exitedEarly = $p.HasExited
$exitCode = $null
$stdout = ''
if ($exitedEarly) {
    $exitCode = $p.ExitCode
    $stdout = $p.StandardOutput.ReadToEnd()
} else {
    $line = $p.StandardOutput.ReadLine()
    if ($line) { $stdout = $line }
    $p.Kill()
}
$stderr = $p.StandardError.ReadToEnd()

Write-Result @{
    exe = $Exe
    exited_early = $exitedEarly
    exit_code = $exitCode
    stdout = $stdout
    stderr = $stderr
}
