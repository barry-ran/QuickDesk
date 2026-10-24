[CmdletBinding()]
param(
    [string]$PackageDir = (Join-Path $PSScriptRoot '..\publish\Release'),
    [string]$DumpbinPath = ''
)

$ErrorActionPreference = 'Stop'

function Resolve-FullPath {
    param([string]$Path)

    return [IO.Path]::GetFullPath((Resolve-Path -LiteralPath $Path -ErrorAction Stop).Path)
}

function Assert-RequiredFile {
    param(
        [string]$PackageRoot,
        [string]$RelativePath
    )

    $path = Join-Path $PackageRoot $RelativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "required publish file is missing: $RelativePath"
    }

    $file = Get-Item -LiteralPath $path
    if ($file.Length -le 0) {
        throw "required publish file is empty: $RelativePath"
    }
    return $file
}

function Get-PeMachine {
    param([string]$Path)

    $stream = [IO.File]::Open(
        $Path,
        [IO.FileMode]::Open,
        [IO.FileAccess]::Read,
        [IO.FileShare]::ReadWrite
    )
    $reader = New-Object IO.BinaryReader($stream)
    try {
        if ($reader.ReadUInt16() -ne 0x5A4D) {
            throw "not a PE file: $Path"
        }

        $stream.Position = 0x3C
        $peOffset = $reader.ReadInt32()
        if ($peOffset -lt 0 -or $peOffset -gt ($stream.Length - 6)) {
            throw "invalid PE header offset: $Path"
        }

        $stream.Position = $peOffset
        if ($reader.ReadUInt32() -ne 0x00004550) {
            throw "invalid PE signature: $Path"
        }
        return $reader.ReadUInt16()
    } finally {
        $reader.Dispose()
        $stream.Dispose()
    }
}

function Find-Dumpbin {
    param([string]$ExplicitPath)

    if ($ExplicitPath) {
        if (-not (Test-Path -LiteralPath $ExplicitPath -PathType Leaf)) {
            throw "dumpbin.exe was not found: $ExplicitPath"
        }
        return Resolve-FullPath -Path $ExplicitPath
    }

    $command = Get-Command dumpbin.exe -CommandType Application -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if ($command) {
        return $command.Source
    }

    if ($env:VCToolsInstallDir) {
        $candidate = Join-Path $env:VCToolsInstallDir 'bin\Hostx64\x64\dumpbin.exe'
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            return $candidate
        }
    }

    $vswhereCandidates = @(
        (Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'),
        (Join-Path $env:ProgramFiles 'Microsoft Visual Studio\Installer\vswhere.exe')
    )
    foreach ($vswhere in $vswhereCandidates) {
        if (-not (Test-Path -LiteralPath $vswhere -PathType Leaf)) {
            continue
        }

        $installations = & $vswhere -latest -products * `
            -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 `
            -property installationPath 2>$null
        foreach ($installation in @($installations)) {
            if (-not $installation) {
                continue
            }

            $msvcRoot = Join-Path $installation 'VC\Tools\MSVC'
            if (-not (Test-Path -LiteralPath $msvcRoot -PathType Container)) {
                continue
            }

            $versions = Get-ChildItem -LiteralPath $msvcRoot -Directory |
                Sort-Object Name -Descending
            foreach ($version in $versions) {
                $candidate = Join-Path $version.FullName 'bin\Hostx64\x64\dumpbin.exe'
                if (Test-Path -LiteralPath $candidate -PathType Leaf) {
                    return $candidate
                }
            }
        }
    }

    throw 'dumpbin.exe was not found. Run this check from a Visual Studio developer shell or pass -DumpbinPath.'
}

function Get-DirectImports {
    param(
        [string]$Dumpbin,
        [string]$Path
    )

    $output = & $Dumpbin /nologo /dependents $Path 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "dumpbin failed for $Path`n$($output -join [Environment]::NewLine)"
    }

    $imports = New-Object System.Collections.Generic.List[string]
    $insideDependencies = $false
    foreach ($rawLine in @($output)) {
        $line = [string]$rawLine
        if ($line -match 'Image has the following dependencies') {
            $insideDependencies = $true
            continue
        }
        if ($insideDependencies -and $line -match '^\s+Summary\s*$') {
            break
        }
        if ($insideDependencies -and $line -match '^\s+([A-Za-z0-9_.-]+\.dll)\s*$') {
            $imports.Add($Matches[1])
        }
    }

    if ($imports.Count -eq 0) {
        throw "dumpbin returned no direct imports for $Path"
    }
    return $imports.ToArray()
}

function Assert-PackageDependencies {
    param(
        [string]$PackageRoot,
        [string]$Dumpbin,
        [string]$RelativeBinary,
        [string[]]$RequiredImports
    )

    $binaryPath = Join-Path $PackageRoot $RelativeBinary
    $imports = @(Get-DirectImports -Dumpbin $Dumpbin -Path $binaryPath)

    foreach ($required in $RequiredImports) {
        if (-not ($imports -contains $required)) {
            throw "$RelativeBinary does not declare the expected dependency $required"
        }
    }

    $localImports = $imports | Where-Object {
        $_ -match '^(Qt6|RapidOcr|MSVCP140|VCRUNTIME140|quickdesk_)'
    }
    foreach ($name in $localImports) {
        [void](Assert-RequiredFile -PackageRoot $PackageRoot -RelativePath $name)
    }
}

function Get-FileVersionText {
    param([string]$Path)

    # Deliberately the file version rather than the product version: the Qt
    # platform plugin and the Qt libraries it loads are matched at load time by
    # the build id Qt records internally, and a mismatch is what produces
    # "Available platform plugins are: windows" followed by a failure to
    # initialize - the plugin is present and is simply not usable.
    $info = Get-Item -LiteralPath $Path
    $versionInfo = $info.VersionInfo
    if ($versionInfo -and $versionInfo.FileVersion) {
        return $versionInfo.FileVersion
    }
    return ''
}

function Assert-QtVersionsAgree {
    param(
        [string]$PackageRoot,
        [string[]]$RelativeBinaries,
        # Injected so the comparison can be tested without a stamped PE, which
        # is not something a test can manufacture reliably. Defaults to the real
        # reader.
        [scriptblock]$ReadVersion = { param($Path) (Get-FileVersionText -Path $Path) }
    )

    # A Qt plugin that loads against a different Qt build than the application
    # is present, x64, and has all its imports satisfied - and still fails to
    # initialize. Every check above passes on that package, which is why the
    # failure reached users as "reinstalling may fix this" with no actionable
    # cause. Comparing the shipped Qt versions is what catches it.
    $versions = @{}
    foreach ($relative in $RelativeBinaries) {
        $path = Join-Path $PackageRoot $relative
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            continue
        }
        $version = & $ReadVersion $path
        if ([string]::IsNullOrWhiteSpace($version)) {
            # Not every Qt build stamps a file version, and a missing stamp is
            # not evidence of a mismatch. Skipping keeps a legitimate package
            # from being rejected for something the check cannot see.
            continue
        }
        if (-not $versions.ContainsKey($version)) {
            $versions[$version] = New-Object System.Collections.Generic.List[string]
        }
        $versions[$version].Add($relative)
    }

    if ($versions.Count -le 1) {
        return
    }

    $report = ($versions.GetEnumerator() | ForEach-Object {
        "{0}: {1}" -f $_.Key, ($_.Value -join ', ')
    }) -join '; '
    throw ("the package ships more than one Qt version, so the platform plugin cannot load: {0}" -f $report)
}

try {
    $normalizedPackageDir = $PackageDir.Trim().Trim('"')
    if ($normalizedPackageDir.Length -gt 3) {
        $normalizedPackageDir = $normalizedPackageDir.TrimEnd('\')
    }
    $packageRoot = Resolve-FullPath -Path $normalizedPackageDir
    if (-not (Test-Path -LiteralPath $packageRoot -PathType Container)) {
        throw "publish directory is missing: $packageRoot"
    }

    $requiredFiles = @(
        'QuickDesk.exe',
        'platforms\qwindows.dll',
        'Qt6Core.dll',
        'Qt6Gui.dll',
        'msvcp140.dll',
        'msvcp140_1.dll',
        'msvcp140_2.dll',
        'vcruntime140.dll',
        'vcruntime140_1.dll'
    )
    foreach ($relativePath in $requiredFiles) {
        [void](Assert-RequiredFile -PackageRoot $packageRoot -RelativePath $relativePath)
    }

    $dumpbin = Find-Dumpbin -ExplicitPath $DumpbinPath
    foreach ($relativeBinary in @('QuickDesk.exe', 'platforms\qwindows.dll')) {
        $binaryPath = Join-Path $packageRoot $relativeBinary
        $machine = Get-PeMachine -Path $binaryPath
        if ($machine -ne 0x8664) {
            throw ("{0} is not x64: machine=0x{1:X4}" -f $relativeBinary, $machine)
        }
    }

    Assert-PackageDependencies `
        -PackageRoot $packageRoot `
        -Dumpbin $dumpbin `
        -RelativeBinary 'platforms\qwindows.dll' `
        -RequiredImports @(
            'Qt6Gui.dll',
            'Qt6Core.dll',
            'MSVCP140.dll',
            'VCRUNTIME140.dll',
            'VCRUNTIME140_1.dll'
        )

    Assert-PackageDependencies `
        -PackageRoot $packageRoot `
        -Dumpbin $dumpbin `
        -RelativeBinary 'QuickDesk.exe' `
        -RequiredImports @(
            'Qt6QuickControls2.dll',
            'Qt6Widgets.dll',
            'Qt6Multimedia.dll',
            'Qt6WebSockets.dll',
            'RapidOcrOnnx.dll',
            'Qt6Quick.dll',
            'Qt6Qml.dll',
            'Qt6Gui.dll',
            'Qt6Network.dll',
            'Qt6Sql.dll',
            'Qt6Core.dll',
            'MSVCP140.dll',
            'VCRUNTIME140.dll',
            'VCRUNTIME140_1.dll'
        )

    Assert-QtVersionsAgree -PackageRoot $PackageRoot -RelativeBinaries @(
        'Qt6Core.dll',
        'Qt6Gui.dll',
        'Qt6Widgets.dll',
        'Qt6Qml.dll',
        'Qt6Quick.dll',
        'Qt6QuickControls2.dll',
        'Qt6Network.dll',
        'Qt6Sql.dll',
        'Qt6Multimedia.dll',
        'Qt6WebSockets.dll',
        'platforms\qwindows.dll'
    )

    Write-Host "PASS: Windows publish dependencies verified: $packageRoot"
    exit 0
} catch {
    Write-Error ("{0}`n{1}" -f $_.Exception.Message, $_.ScriptStackTrace)
    exit 1
}
