#!/usr/bin/env pwsh
# Copyright 2018 the Deno authors. All rights reserved. MIT license.
# TODO(everyone): Keep this script simple and easily auditable.
# if Permission issue says that "ps1 cannot be loaded because running scripts is disabled on this system" run the below command as Admin and then execute the script
#Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy Unrestricted

$ErrorActionPreference = 'Stop'
$Version = if ($v) {
  $v
} elseif ($args.Length -eq 1) {
  $args.Get(0)
} else {
  "latest"
}

$NifeInstall = $env:NIFE_INSTALL
$BinDir = if ($NifeInstall) {
  "$NifeInstall\bin"
} else {
  "$Home\.nife\bin"
}

$NifeZip = "$BinDir\nifectl.zip"
$NifeExe = "$BinDir\nifectl.exe"
$WintunDll = "$BinDir\wintun.dll"
$NifectlExe = "$BinDir\nife.exe"

# Nife & GitHub require TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

try {
  $Response = Invoke-WebRequest "https://api.nife.io/release/windows/x86_64/$Version" -UseBasicParsing
  $NifeUri = "$Response"#.Content
}
catch {
  $StatusCode = $_.Exception.Response.StatusCode.value__
  if ($StatusCode -eq 404) {
    Write-Error "Unable to find a Nife release on GitHub for version:$Version - see github.com/nifetency/nife_releases for all versions"
  } else {
    $Request = $_.Exception
    Write-Error "Error while fetching releases: $Request"
  }
  Exit 1
}

if (!(Test-Path $BinDir)) {
  New-Item $BinDir -ItemType Directory | Out-Null
}
Invoke-WebRequest $NifeUri -OutFile $NifeZip -UseBasicParsing

if (Get-Command Expand-Archive -ErrorAction SilentlyContinue) {
  Expand-Archive $NifeZip -Destination $BinDir -Force
} else {
  Remove-Item $NifectlExe -ErrorAction SilentlyContinue
  Remove-Item $NifeExe -ErrorAction SilentlyContinue
  Remove-Item $WintunDll -ErrorAction SilentlyContinue
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  [IO.Compression.ZipFile]::ExtractToDirectory($NifeZip, $BinDir)
}

Remove-Item $NifeZip

$User = [EnvironmentVariableTarget]::User
$Path = [Environment]::GetEnvironmentVariable('Path', $User)
if (!(";$Path;".ToLower() -like "*;$BinDir;*".ToLower())) {
  [Environment]::SetEnvironmentVariable('Path', "$Path;$BinDir", $User)
  $Env:Path += ";$BinDir"
}

Start-Process -FilePath "$env:comspec" -ArgumentList "/c", "mklink", $NifeExe, $NifectlExe

Write-Output "Nifectl was installed successfully to $NifectlExe"
Write-Output "Run 'nifectl --help' to get started"
