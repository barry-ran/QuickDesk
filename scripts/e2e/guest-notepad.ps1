<#
Guest-side Notepad helper for the remote-control input test.

    guest-notepad.ps1 -Action start   # open a fresh Notepad on the console desktop
    guest-notepad.ps1 -Action read    # print the current document text

Prints one machine readable line: PDJSON_B64:<base64(utf8 json)>.
Run it as the logged-in console user (prlctl exec --current-user) so Notepad
appears on the desktop that the remote session is streaming and controlling.
#>
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('start', 'focus', 'read')]
    [string]$Action,
    [int]$TimeoutSec = 20
)

$ErrorActionPreference = 'Stop'

function Write-Result {
    param($Object)
    $json = $Object | ConvertTo-Json -Depth 8 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    Write-Output ('PDJSON_B64:' + [Convert]::ToBase64String($bytes))
}

function Get-UiaWindow {
    param([int]$ProcessId)
    Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
    $root = [System.Windows.Automation.AutomationElement]::RootElement
    $cond = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::ProcessIdProperty, $ProcessId)
    return $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $cond)
}

function Get-DocumentText {
    param($Window)
    if (-not $Window) { return @{ ok = $false; text = $null; error = 'no window element' } }
    foreach ($type in @([System.Windows.Automation.ControlType]::Document, [System.Windows.Automation.ControlType]::Edit)) {
        $cond = New-Object System.Windows.Automation.PropertyCondition(
            [System.Windows.Automation.AutomationElement]::ControlTypeProperty, $type)
        $element = $Window.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $cond)
        if (-not $element) { continue }
        try {
            $value = $element.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
            return @{ ok = $true; text = $value.Current.Value; via = 'ValuePattern' }
        } catch { }
        try {
            $text = $element.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
            return @{ ok = $true; text = $text.DocumentRange.GetText(-1); via = 'TextPattern' }
        } catch { }
    }
    return @{ ok = $false; text = $null; error = 'no ValuePattern/TextPattern document found' }
}

function Get-NotepadWindow {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        foreach ($p in @(Get-Process -Name Notepad -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 })) {
            $w = Get-UiaWindow -ProcessId $p.Id
            if ($w) { return @{ window = $w; process = $p } }
        }
        Start-Sleep -Milliseconds 500
    }
    return $null
}

if ($Action -eq 'start') {
    Get-Process -Name Notepad -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 600
    Start-Process notepad.exe | Out-Null
}

$found = Get-NotepadWindow
if (-not $found) {
    Write-Result @{ action = $Action; ok = $false; error = 'notepad window not found via UI Automation' }
    exit 1
}

if ($Action -eq 'focus') {
    # The remote session sends input to whatever window owns the foreground on
    # the console desktop, so Notepad has to be activated before typing.
    Add-Type -AssemblyName Microsoft.VisualBasic
    [Microsoft.VisualBasic.Interaction]::AppActivate($found.process.Id)
    Start-Sleep -Milliseconds 600
    # Maximize so a click in the middle of the streamed desktop lands on
    # Notepad instead of whatever window happens to sit at those coordinates.
    try {
        $pattern = $found.window.GetCurrentPattern([System.Windows.Automation.WindowPattern]::Pattern)
        $pattern.SetWindowVisualState([System.Windows.Automation.WindowVisualState]::Maximized)
    } catch { }
    Start-Sleep -Milliseconds 800
    $foreground = [System.Windows.Automation.AutomationElement]::FocusedElement
    Write-Result @{
        action     = 'focus'
        ok         = $true
        pid        = $found.process.Id
        foreground = $foreground.Current.Name
    }
    exit 0
}

if ($Action -eq 'start') {
    Write-Result @{ action = 'start'; ok = $true; pid = $found.process.Id }
    exit 0
}

$doc = Get-DocumentText $found.window
Write-Result @{
    action = 'read'
    ok     = [bool]$doc.ok
    via    = $doc.via
    pid    = $found.process.Id
    text   = $doc.text
    error  = $doc.error
}
