param(
    [Parameter(Mandatory = $true)]
    [string] $Output
)

$ErrorActionPreference = "Stop"

$versionPath = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot "../../internal/appmeta/VERSION"))
$version = [IO.File]::ReadAllText($versionPath).Trim()
if (-not ($version -match '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$')) {
    throw "Version '$version' in $versionPath is not MAJOR.MINOR.PATCH with an optional -PRERELEASE suffix."
}

# Windows stores the numeric version as four 16-bit parts and cannot hold a prerelease suffix.
$parts = @($Matches[1], $Matches[2], $Matches[3])
foreach ($part in $parts) {
    if ([long] $part -gt 65535) {
        throw "Version '$version' in $versionPath has a part above 65535, which Windows cannot store."
    }
}
$numeric = ($parts + "0") -join "."

$fixed = [ordered]@{ file_version = $numeric; product_version = $numeric }
if ($Matches[4]) {
    $fixed.flags = "Prerelease"
}

$info = [IO.File]::ReadAllText((Join-Path $PSScriptRoot "info.json")) | ConvertFrom-Json
$info | Add-Member -NotePropertyName "fixed" -NotePropertyValue $fixed
$strings = $info.info."0409"
$strings | Add-Member -NotePropertyName "FileVersion" -NotePropertyValue $version
$strings | Add-Member -NotePropertyName "ProductVersion" -NotePropertyValue $version

# WriteAllText writes UTF-8 without a byte order mark, which the JSON decoder requires.
[IO.File]::WriteAllText($Output, ($info | ConvertTo-Json -Depth 5))
