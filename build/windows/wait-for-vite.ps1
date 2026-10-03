param(
    [Parameter(Mandatory = $true)]
    [ValidateRange(1, 65535)]
    [int] $Port,
    [ValidateRange(1, 300)]
    [int] $TimeoutSeconds = 60
)

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
$uri = "http://127.0.0.1:$Port/"

do {
    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $uri -TimeoutSec 1
        if ($response.StatusCode -eq 200) {
            exit 0
        }
    }
    catch {
        # Vite may still be starting. Retry until the deadline.
    }

    Start-Sleep -Milliseconds 100
} while ((Get-Date) -lt $deadline)

Write-Error "Vite dev server did not become ready at $uri within $TimeoutSeconds seconds."
exit 1
