# Test if Chaturbate cookies are working
param(
    [Parameter(Mandatory=$true)]
    [string]$Cookies
)

$headers = @{
    "User-Agent" = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
    "Cookie" = $Cookies
    "Referer" = "https://chaturbate.com/"
}

Write-Host "Testing Chaturbate API with provided cookies..." -ForegroundColor Cyan

try {
    $response = Invoke-WebRequest -Uri "https://chaturbate.com/api/chatvideocontext/rusiksb31/" -Headers $headers -UseBasicParsing
    
    if ($response.Content -like "*Just a moment*") {
        Write-Host "❌ FAILED: Cloudflare challenge detected" -ForegroundColor Red
        Write-Host "Your cookies are invalid or expired" -ForegroundColor Yellow
    } elseif ($response.StatusCode -eq 200) {
        Write-Host "✅ SUCCESS: Cookies are working!" -ForegroundColor Green
        Write-Host "Response preview:" -ForegroundColor Cyan
        Write-Host ($response.Content.Substring(0, [Math]::Min(200, $response.Content.Length)))
    }
} catch {
    Write-Host "❌ ERROR: $($_.Exception.Message)" -ForegroundColor Red
}
