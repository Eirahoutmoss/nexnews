$ErrorActionPreference = 'Stop'
$body = @{ q='Hello world'; source='auto'; target='tr'; format='text' } | ConvertTo-Json
$r = Invoke-WebRequest -Uri 'http://127.0.0.1:5000/translate' -Method Post -ContentType 'application/json' -Body $body -TimeoutSec 30 -UseBasicParsing
Write-Host $r.Content
