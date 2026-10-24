# Verifies the Qt version agreement check in verify_qd_win_release.ps1.
#
# Why this needs a test at all: the check exists for a package shape every other
# check in the verifier passes. A Qt platform plugin from a different Qt build
# than the application is present, is x64, and has all of its imports satisfied
# - and still fails to initialize, which is what "Available platform plugins
# are: windows" followed by a startup failure actually means.
#
# It runs under plain PowerShell on Linux, so the DLLs used here are the .NET
# runtime's own files rather than Qt's. That is fine and deliberate: the check
# reads file versions and compares them, so any stamped PE exercises the same
# path a Qt DLL would.

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$verifier = Join-Path $repoRoot 'scripts\verify_qd_win_release.ps1'

function Get-VerifierFunctions {
    # Extracts the functions under test by dot-sourcing the verifier's body in a
    # scope where the entry-point guard is neutralised. The script runs its checks
    # at the bottom, so it cannot be dot-sourced directly.
    $body = Get-Content -LiteralPath $verifier -Raw
    # Drop the param block: it evaluates $PSScriptRoot at dot-source time, where
    # it is empty because this test, not the verifier, is the running script.
    # The anchor is the line after the block rather than the first ')', which
    # belongs to a Join-Path inside the default value.
    $anchor = "`$ErrorActionPreference = 'Stop'"
    $bodyAt = $body.IndexOf($anchor)
    if ($bodyAt -lt 0) { throw 'could not find the body of the verifier' }
    $body = $body.Substring($bodyAt)

    # The last `try {` in the file is the entry-point guard; the earlier ones
    # belong to the helper functions and would truncate them.
    $marker = 'try {'
    $at = $body.LastIndexOf($marker)
    if ($at -lt 0) { throw 'could not find the verifier entry point' }
    $functionsOnly = $body.Substring(0, $at)
    return $functionsOnly
}

$workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("qd215-qtver-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $workRoot -Force | Out-Null
try {
    $package = Join-Path $workRoot 'publish'
    New-Item -ItemType Directory -Path (Join-Path $package 'platforms') -Force | Out-Null

    # Plain files: the comparison never parses a PE, it only asks the injected
    # reader for a version string. Manufacturing a stamped DLL is not something a
    # test can do reliably, and pretending otherwise would make this test a lie.
    $names = @('Qt6Core.dll', 'Qt6Gui.dll', 'Qt6Widgets.dll', 'Qt6Qml.dll',
               'Qt6Quick.dll', 'Qt6QuickControls2.dll', 'Qt6Network.dll',
               'Qt6Sql.dll', 'Qt6Multimedia.dll', 'Qt6WebSockets.dll')
    foreach ($name in $names) {
        Set-Content -LiteralPath (Join-Path $package $name) -Value 'stub'
    }
    Set-Content -LiteralPath (Join-Path $package 'platforms\qwindows.dll') -Value 'stub'

    . ([scriptblock]::Create((Get-VerifierFunctions)))

    # A reader that reports one agreed version for every Qt file.
    $agreed = { param($Path) '6.7.1.0' }

    Assert-QtVersionsAgree -PackageRoot $package -ReadVersion $agreed -RelativeBinaries @(
        'Qt6Core.dll', 'Qt6Gui.dll', 'Qt6Widgets.dll'
    )
    Write-Host 'PASS: matching Qt versions are accepted'

    # One plugin from a different build. This is the shape every other check in
    # the verifier passes, and it is the one that fails to initialize on a user's
    # machine - present, x64, imports satisfied, and unusable.
    $mismatched = { param($Path)
        if ($Path -like '*platforms*') { return '6.4.2.0' }
        return '6.7.1.0'
    }
    $threw = $false
    $message = ''
    try {
        Assert-QtVersionsAgree -PackageRoot $package -ReadVersion $mismatched -RelativeBinaries @(
            'Qt6Core.dll', 'Qt6Gui.dll', 'platforms\qwindows.dll'
        )
    } catch {
        $threw = $true
        $message = $_.Exception.Message
    }
    if (-not $threw) {
        throw 'a Qt platform plugin from a different build was accepted'
    }
    if ($message -notmatch 'more than one Qt version') {
        throw "rejection message does not name the cause: $message"
    }
    if ($message -notmatch '6\.4\.2\.0' -or $message -notmatch 'qwindows') {
        throw "rejection message does not name the offending file and version: $message"
    }
    Write-Host 'PASS: a mismatched platform plugin is rejected with the file and version named'

    # A build that stamps no file version at all must not be read as a mismatch:
    # a legitimate package would otherwise be rejected for something the check
    # cannot see.
    $unstamped = { param($Path)
        if ($Path -like '*Qt6Sql*') { return '' }
        return '6.7.1.0'
    }
    Assert-QtVersionsAgree -PackageRoot $package -ReadVersion $unstamped -RelativeBinaries @(
        'Qt6Core.dll', 'Qt6Sql.dll'
    )
    Write-Host 'PASS: an unstamped DLL is skipped rather than counted as a mismatch'

    # A file that is not in the package at all is skipped, so the list of names
    # can be broader than what a given build ships.
    Assert-QtVersionsAgree -PackageRoot $package -ReadVersion $agreed -RelativeBinaries @(
        'Qt6Core.dll', 'NotShipped.dll'
    )
    Write-Host 'PASS: an absent file is skipped rather than failing the check'

    Write-Host 'Qt version agreement tests passed'
    exit 0
} finally {
    Remove-Item -Recurse -Force -LiteralPath $workRoot -ErrorAction SilentlyContinue
}
