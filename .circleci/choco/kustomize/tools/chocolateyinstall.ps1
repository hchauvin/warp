$ErrorActionPreference = 'Stop';
$toolsDir     = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# release version of kustomize: https://github.com/kubernetes-sigs/kustomize/releases
$version = '3.4.0'
# pattern for exe name
$exe_name = "kustomize_$($version)_windows_amd64.exe"
# only 64bit url
$url = "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$($version)/kustomize_v$($version)_windows_amd64.tar.gz"

# use $ checksum [exe] -t=sha256
$version_checksum = '6ede9c679c9203c593373cc6c68693355ef814224b1320cc1b30e3ab102b0efc'
$checksum_type = 'sha256'

# destination for exe
$fileLocation = join-path $toolsDir $exe_name

$archiveLocation = New-TemporaryFile

$getArgs = @{
  packageName   = $env:ChocolateyPackageName
  fileFullPath  = $archiveLocation
  url64bit      = $url
  checksum64    = $checksum
  checksumType64= $type
}

Get-ChocolateyWebFile @getArgs

tar xzf $archiveLocation
Move-Item kustomize.exe $fileLocation

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  softwareName  = 'kustomize*'
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
  name = 'kustomize'
  path = $fileLocation
}

Install-BinFile @binArgs