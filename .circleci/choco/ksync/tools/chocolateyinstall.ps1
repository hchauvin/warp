$ErrorActionPreference = 'Stop';
$toolsDir     = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# release version of ksync: https://github.com/ksync/ksync/releases
$version = '0.4.1'
# pattern for exe name
$exe_name = "ksync_windows_amd64.exe"
# only 64bit url
$url = "https://github.com/ksync/ksync/releases/download/$($version)/$($exe_name)"

# use $ checksum [exe] -t=sha256
$version_checksum = '1e2d26ca9e24c20f4419485a2d0aa8c2e78edfeabac85ae00488d7fd1c7ad4c5'
$checksum_type = 'sha256'

# destination for exe
$fileLocation = join-path $toolsDir $exe_name

$getArgs = @{
  packageName   = $env:ChocolateyPackageName
  fileFullPath  = $fileLocation
  url64bit      = $url
  checksum64    = $checksum
  checksumType64= $type
}

Get-ChocolateyWebFile @getArgs

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  softwareName  = 'ksync*'
  fileType      = 'exe'
  silentArgs    = ""
  validExitCodes= @(0)
  file64        = $fileLocation
  checksum64    = $checksum
  checksumType64= $type
  destination   = $toolsDir
}

Install-ChocolateyInstallPackage @packageArgs

$binargs = @{
  name = 'ksync'
  path = $fileLocation
}

Install-BinFile @binArgs