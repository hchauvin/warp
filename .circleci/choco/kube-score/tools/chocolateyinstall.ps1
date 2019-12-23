$ErrorActionPreference = 'Stop';
$toolsDir     = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# release version of kustomize: https://github.com/kubernetes-sigs/kustomize/releases
$version = '1.4.0'
# pattern for exe name
$exe_name = "kube-score_$($version)_windows_amd64.exe"
# only 64bit url
$url = "https://github.com/zegl/kube-score/releases/download/v$($version)/kube-score_$($version)_windows_amd64.tar.gz"

# use $ checksum [exe] -t=sha256
$version_checksum = 'c7b47001121589b41e4f3017a07035382681fee90e97d7e84e117a18bafcbb89'
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
Move-Item kube-score.exe $fileLocation

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  softwareName  = 'kube-score*'
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
  name = 'kube-score'
  path = $fileLocation
}

Install-BinFile @binArgs