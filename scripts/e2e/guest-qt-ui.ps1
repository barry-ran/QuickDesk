# Drive the QuickDesk Qt client through UI Automation (guest side).
#
#   qt-ui.ps1 -Action dump            # print the window's element tree
#   qt-ui.ps1 -Action focus           # activate + maximize the main window
#   qt-ui.ps1 -Action set -Name X -Value Y
#   qt-ui.ps1 -Action click -Name X
param(
    [Parameter(Mandatory = $true)][ValidateSet('dump', 'focus', 'set', 'setedit', 'click', 'list', 'shortcut')][string]$Action,
    [string]$Name = '',
    [string]$Value = '',
    [int]$Index = 0,
    [string]$WindowTitle = 'QuickDesk',
    [int]$Depth = 6
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes

function Write-Result {
    param($Object)
    $json = $Object | ConvertTo-Json -Depth 6 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    Write-Output ('PDJSON_B64:' + [Convert]::ToBase64String($bytes))
}

function Get-QdWindowElement {
    # Match by process first: the Qt window is created by QuickDesk.exe and this
    # is the same lookup shape the Notepad helper uses reliably.
    $uiRoot = [System.Windows.Automation.AutomationElement]::RootElement
    foreach ($process in @(Get-Process -Name QuickDesk -ErrorAction SilentlyContinue |
                           Where-Object { $_.MainWindowHandle -ne 0 })) {
        $processCondition = New-Object System.Windows.Automation.PropertyCondition(
            [System.Windows.Automation.AutomationElement]::ProcessIdProperty, $process.Id)
        $candidate = $uiRoot.FindFirst([System.Windows.Automation.TreeScope]::Children, $processCondition)
        if ($candidate) { return $candidate }
    }
    $condition = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::NameProperty, $WindowTitle)
    $byName = $uiRoot.FindFirst([System.Windows.Automation.TreeScope]::Children, $condition)
    if ($byName) { return $byName }
    # Fall back to any window whose name contains the title.
    $topLevel = $uiRoot.FindAll([System.Windows.Automation.TreeScope]::Children,
                                [System.Windows.Automation.Condition]::TrueCondition)
    foreach ($element in $topLevel) {
        if ($element.Current.Name -like "*$WindowTitle*") { return $element }
    }
    return $null
}

function Get-QdWindowElementForTitle {
    param([string]$TitlePart)
    $uiRoot = [System.Windows.Automation.AutomationElement]::RootElement
    $topLevel = $uiRoot.FindAll([System.Windows.Automation.TreeScope]::Children,
                                [System.Windows.Automation.Condition]::TrueCondition)
    foreach ($element in $topLevel) {
        if ($element.Current.Name -like "*$TitlePart*") { return $element }
    }
    return $null
}

if ($Action -eq 'shortcut') {
    # Activate a window by title fragment and send a SendKeys sequence into it.
    $titlePart = if ($Name) { $Name } else { '远程桌面' }
    $sequence = if ($Value) { $Value } else { '^+r' }
    $remoteElement = Get-QdWindowElementForTitle -TitlePart $titlePart
    if (-not $remoteElement) {
        Write-Result @{ ok = $false; error = "no window matching '$titlePart'" }
        exit 1
    }
    $remoteProcessId = $remoteElement.Current.ProcessId
    if ($remoteProcessId -le 0) {
        $live = @(Get-Process -Name QuickDesk -ErrorAction SilentlyContinue |
                  Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1)
        if ($live) { $remoteProcessId = $live.Id }
    }
    Add-Type -AssemblyName Microsoft.VisualBasic
    Add-Type -AssemblyName System.Windows.Forms
    [Microsoft.VisualBasic.Interaction]::AppActivate($remoteProcessId)
    Start-Sleep -Milliseconds 700
    try {
        $windowPattern = $remoteElement.GetCurrentPattern(
            [System.Windows.Automation.WindowPattern]::Pattern)
        $windowPattern.SetWindowVisualState(
            [System.Windows.Automation.WindowVisualState]::Maximized)
    } catch { }
    Start-Sleep -Milliseconds 500
    [System.Windows.Forms.SendKeys]::SendWait($sequence)
    Start-Sleep -Milliseconds 400
    Write-Result @{ ok = $true; action = 'shortcut'; window = $remoteElement.Current.Name
                   sequence = $sequence; pid = $remoteProcessId }
    exit 0
}

function Get-Descendants {
    param($Element, [int]$Max)
    return $Element.FindAll([System.Windows.Automation.TreeScope]::Descendants,
                            [System.Windows.Automation.Condition]::TrueCondition)
}

function Find-Element {
    param($Root, [string]$Target)
    $all = Get-Descendants -Element $Root -Max 0
    foreach ($element in $all) {
        if ($element.Current.Name -eq $Target) { return $element }
    }
    foreach ($element in $all) {
        if ($element.Current.Name -like "*$Target*") { return $element }
    }
    return $null
}

$qdWindow = Get-QdWindowElement
if (-not $qdWindow) {
    Write-Result @{ ok = $false; error = 'QuickDesk window not found' }
    exit 1
}

if ($Action -eq 'focus') {
    Add-Type -AssemblyName Microsoft.VisualBasic
    # Resolve the pid from the live process list: it is always current, while a
    # cached UIA element can go stale when the window is recreated.
    $targetPid = 0
    $live = @(Get-Process -Name QuickDesk -ErrorAction SilentlyContinue |
              Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1)
    if ($live) { $targetPid = $live.Id }
    if ($targetPid -le 0) { $targetPid = $qdWindow.Current.ProcessId }
    if ($targetPid -le 0) {
        Write-Result @{ ok = $false; error = 'could not resolve the QuickDesk process id' }
        exit 1
    }
    [Microsoft.VisualBasic.Interaction]::AppActivate($targetPid)
    Start-Sleep -Milliseconds 600
    try {
        $pattern = $qdWindow.GetCurrentPattern([System.Windows.Automation.WindowPattern]::Pattern)
        $pattern.SetWindowVisualState([System.Windows.Automation.WindowVisualState]::Maximized)
    } catch { }
    Start-Sleep -Milliseconds 800
    Write-Result @{ ok = $true; pid = $targetPid; name = $qdWindow.Current.Name }
    exit 0
}

if ($Action -eq 'list') {
    $items = @()
    foreach ($element in (Get-Descendants -Element $qdWindow -Max 0)) {
        $name = $element.Current.Name
        if ([string]::IsNullOrWhiteSpace($name)) { continue }
        $items += ("{0} | {1} | {2}" -f $element.Current.ControlType.ProgrammaticName, $name,
                   $element.Current.AutomationId)
    }
    Write-Result @{ ok = $true; count = $items.Count; items = $items }
    exit 0
}

if ($Action -eq 'dump') {
    $lines = New-Object System.Collections.ArrayList
    $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
    function Walk($element, $level) {
        if ($level -gt $Depth) { return }
        $name = $element.Current.Name
        $type = $element.Current.ControlType.ProgrammaticName
        [void]$lines.Add(('{0}{1} | {2} | {3}' -f ('  ' * $level), $type, $name,
                          $element.Current.AutomationId))
        $child = $walker.GetFirstChild($element)
        while ($child) {
            Walk $child ($level + 1)
            $child = $walker.GetNextSibling($child)
        }
    }
    Walk $qdWindow 0
    Write-Result @{ ok = $true; lines = $lines }
    exit 0
}

$target = Find-Element -Root $qdWindow -Target $Name
if (-not $target) {
    Write-Result @{ ok = $false; error = "element not found: $Name" }
    exit 1
}

if ($Action -eq 'set') {
    try {
        $pattern = $target.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
        $pattern.SetValue($Value)
    } catch {
        # QML text fields occasionally only accept focus + keystrokes.
        $target.SetFocus()
        Start-Sleep -Milliseconds 200
        Add-Type -AssemblyName System.Windows.Forms
        [System.Windows.Forms.SendKeys]::SendWait('^a')
        [System.Windows.Forms.SendKeys]::SendWait($Value)
    }
    Write-Result @{ ok = $true; action = 'set'; name = $Name; value = $Value }
    exit 0
}

if ($Action -eq 'setedit') {
    # QML text fields have no accessible name, so address them by position.
    $edits = @()
    foreach ($element in (Get-Descendants -Element $qdWindow -Max 0)) {
        if ($element.Current.ControlType -eq [System.Windows.Automation.ControlType]::Edit) {
            $edits += $element
        }
    }
    if ($Index -ge $edits.Count) {
        Write-Result @{ ok = $false; error = "edit index $Index out of range (found $($edits.Count))" }
        exit 1
    }
    $edit = $edits[$Index]
    $edit.SetFocus()
    Start-Sleep -Milliseconds 250
    try {
        $pattern = $edit.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
        $pattern.SetValue($Value)
    } catch {
        Write-Result @{ ok = $false; error = "edit $Index has no ValuePattern: $($_.Exception.Message)" }
        exit 1
    }
    Start-Sleep -Milliseconds 250
    $check = $edit.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
    Write-Result @{ ok = $true; action = 'setedit'; index = $Index; value = $check.Current.Value }
    exit 0
}

if ($Action -eq 'click') {
    try {
        $pattern = $target.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $pattern.Invoke()
    } catch {
        $rect = $target.Current.BoundingRectangle
        Add-Type -AssemblyName System.Windows.Forms
        Add-Type -Namespace PD -Name Cursor -MemberDefinition @'
[DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
[DllImport("user32.dll")] public static extern void mouse_event(uint f, uint x, uint y, uint d, int i);
'@
        $x = [int]($rect.X + $rect.Width / 2)
        $y = [int]($rect.Y + $rect.Height / 2)
        [PD.Cursor]::SetCursorPos($x, $y) | Out-Null
        [PD.Cursor]::mouse_event(0x0002, 0, 0, 0, 0)
        [PD.Cursor]::mouse_event(0x0004, 0, 0, 0, 0)
    }
    Write-Result @{ ok = $true; action = 'click'; name = $Name }
    exit 0
}
