<#
Guest-side input target for the remote-control input test.

A plain WinForms window with one multiline TextBox. Windows 11 Notepad keeps
multiple restored tabs around, which makes "which document got my keystrokes"
ambiguous; this target is a single, deterministic window.

    guest-input-target.ps1 -Action show    # launch the window (blocks)
    guest-input-target.ps1 -Action focus   # activate + maximize it
    guest-input-target.ps1 -Action read    # print its text

Prints one machine readable line for focus/read: PDJSON_B64:<base64(utf8 json)>.
#>
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('show', 'focus', 'read')]
    [string]$Action,
    [string]$Title = 'QD E2E Input Target',
    [string]$Spool = 'C:\QuickDeskTest\input-target.txt'
)

$ErrorActionPreference = 'Stop'

function Write-Result {
    param($Object)
    $json = $Object | ConvertTo-Json -Depth 8 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    Write-Output ('PDJSON_B64:' + [Convert]::ToBase64String($bytes))
}

function Get-TargetWindow {
    Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
    $root = [System.Windows.Automation.AutomationElement]::RootElement
    $cond = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::NameProperty, $Title)
    return $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $cond)
}

function Get-TargetTextBox {
    param($Window)
    # WinForms exposes its TextBox as ControlType.Pane with a Win32 "EDIT"
    # class name, so scan descendants for an "EDIT" class before falling back
    # to ControlType.Edit.
    $all = $Window.FindAll(
        [System.Windows.Automation.TreeScope]::Descendants,
        [System.Windows.Automation.Condition]::TrueCondition)
    foreach ($el in $all) {
        if ($el.Current.ClassName -match 'EDIT') { return $el }
    }
    $cond = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::ControlTypeProperty,
        [System.Windows.Automation.ControlType]::Edit)
    return $Window.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $cond)
}

if ($Action -eq 'show') {
    Add-Type -AssemblyName System.Windows.Forms, System.Drawing
    [System.Windows.Forms.Application]::EnableVisualStyles()
    $form = New-Object System.Windows.Forms.Form
    $form.Text = $Title
    $form.Width = 900
    $form.Height = 500
    $box = New-Object System.Windows.Forms.TextBox
    $box.Multiline = $true
    $box.AcceptsReturn = $true
    $box.ScrollBars = 'Vertical'
    $box.Dock = 'Fill'
    $box.Font = New-Object System.Drawing.Font('Consolas', 14)
    # WinForms TextBox values are not exposed through UI Automation
    # ValuePattern, so spool every change to a file the host can read.
    $box.Add_TextChanged({
        try { Set-Content -Path $Spool -Value $box.Text -Encoding UTF8 -NoNewline } catch { }
    })
    $form.Controls.Add($box)
    Set-Content -Path $Spool -Value '' -Encoding UTF8 -NoNewline
    [void]$form.ShowDialog()
    exit 0
}

$window = Get-TargetWindow
if (-not $window) {
    Write-Result @{ action = $Action; ok = $false; error = 'input target window not found' }
    exit 1
}

if ($Action -eq 'focus') {
    Add-Type -AssemblyName Microsoft.VisualBasic
    [Microsoft.VisualBasic.Interaction]::AppActivate($window.Current.ProcessId)
    Start-Sleep -Milliseconds 500
    try {
        $pattern = $window.GetCurrentPattern([System.Windows.Automation.WindowPattern]::Pattern)
        $pattern.SetWindowVisualState([System.Windows.Automation.WindowVisualState]::Maximized)
    } catch { }
    Start-Sleep -Milliseconds 800
    $focused = [System.Windows.Automation.AutomationElement]::FocusedElement
    Write-Result @{ action = 'focus'; ok = $true; focused = $focused.Current.Name }
    exit 0
}

$text = ''
if (Test-Path $Spool) { $text = (Get-Content -Path $Spool -Raw -Encoding UTF8) }
Write-Result @{ action = 'read'; ok = $true; text = $text }
