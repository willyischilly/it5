$Base = "http://localhost:8080"
$passed = 0
$failed = 0
$results = @()

function Record($name, $ok, $detail = "") {
    $script:results += [pscustomobject]@{ Test = $name; OK = $ok; Detail = $detail }
    if ($ok) { $script:passed++; Write-Host "[OK] $name" -ForegroundColor Green }
    else { $script:failed++; Write-Host "[FAIL] $name - $detail" -ForegroundColor Red }
}

function Invoke-Api($Method, $Path, $Body = $null, $Token = $null) {
    $headers = @{ "Content-Type" = "application/json" }
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }
    $uri = "$Base$Path"
    try {
        if ($Body) {
            $json = $Body | ConvertTo-Json -Depth 6 -Compress
            $r = Invoke-WebRequest -Uri $uri -Method $Method -Headers $headers -Body $json -UseBasicParsing
        } else {
            $r = Invoke-WebRequest -Uri $uri -Method $Method -Headers $headers -UseBasicParsing
        }
        return @{ Ok = $true; Status = [int]$r.StatusCode; Body = $r.Content; Headers = $r.Headers; Raw = $r }
    } catch {
        $resp = $_.Exception.Response
        $status = if ($resp) { [int]$resp.StatusCode } else { 0 }
        $body = ""
        if ($resp) {
            $sr = New-Object System.IO.StreamReader($resp.GetResponseStream())
            $body = $sr.ReadToEnd()
            $sr.Close()
        }
        return @{ Ok = $false; Status = $status; Body = $body; Error = $_.Exception.Message }
    }
}

Write-Host "=== Planner Backend Full Checkup ===" -ForegroundColor Cyan

$h = Invoke-Api "GET" "/health"
Record "Health endpoint" ($h.Ok -and $h.Status -eq 200) "status=$($h.Status)"

$oa = Invoke-Api "GET" "/api/openapi.yaml"
Record "OpenAPI spec" ($oa.Ok -and $oa.Body.Length -gt 500) "len=$($oa.Body.Length)"

try {
    $sw = Invoke-WebRequest "$Base/swagger" -UseBasicParsing
    Record "Swagger UI" ($sw.StatusCode -eq 200)
} catch { Record "Swagger UI" $false $_.Exception.Message }

$admin = (Invoke-Api "POST" "/api/login" @{ email = "admin@planner.local"; password = "admin123456" }).Body | ConvertFrom-Json
Record "Admin login" ($null -ne $admin.token) ""
$adminTok = $admin.token
$me = (Invoke-Api "GET" "/api/me" $null $adminTok).Body | ConvertFrom-Json
Record "GET /api/me" ($me.role -eq "admin") $me.role

$badLogin = Invoke-Api "POST" "/api/login" @{ email = "admin@planner.local"; password = "wrong" }
Record "Login rejects bad password" ($badLogin.Status -eq 401) "status=$($badLogin.Status)"

$noAuth = Invoke-Api "GET" "/api/admin/users"
Record "Protected route 401" ($noAuth.Status -eq 401) "status=$($noAuth.Status)"

$works = (Invoke-Api "GET" "/api/admin/works" $null $adminTok).Body | ConvertFrom-Json
Record "Seed works >= 7" ($works.Count -ge 7) "count=$($works.Count)"

$contours = (Invoke-Api "GET" "/api/admin/contours" $null $adminTok).Body | ConvertFrom-Json
Record "Seed contours = 4" ($contours.Count -ge 4) "count=$($contours.Count)"

$badWork = Invoke-Api "POST" "/api/admin/works" @{ name = ""; normative_hours = -5 } $adminTok
Record "Validation negative hours" ($badWork.Status -eq 400) "status=$($badWork.Status)"

# перед сценарием оставляем одного исполнителя
$s = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$custEmail = "check_cust_$s@test.local"
$execEmail = "check_exec_$s@test.local"
Invoke-Api "POST" "/api/register" @{ email = $custEmail; password = "123456"; name = "C"; role = "customer" } | Out-Null
for ($try = 0; $try -lt 15; $try++) {
    $allUsers = (Invoke-Api "GET" "/api/admin/users" $null $adminTok).Body | ConvertFrom-Json
    $execs = @($allUsers | Where-Object { $_.role -eq "executor" })
    if ($execs.Count -eq 0) { break }
    foreach ($u in $execs) {
        Invoke-Api "DELETE" "/api/admin/users/$($u.id)" $null $adminTok | Out-Null
    }
}
$createExec = Invoke-Api "POST" "/api/admin/users" @{
    email = $execEmail; password = "123456"; name = "E"; role = "executor"
} $adminTok
$custTok = ((Invoke-Api "POST" "/api/login" @{ email = $custEmail; password = "123456" }).Body | ConvertFrom-Json).token
$execTok = ((Invoke-Api "POST" "/api/login" @{ email = $execEmail; password = "123456" }).Body | ConvertFrom-Json).token
$execCount = @((Invoke-Api "GET" "/api/admin/users" $null $adminTok).Body | ConvertFrom-Json | Where-Object { $_.role -eq "executor" }).Count
Record "Customer/executor register+login" ($custTok -and $execTok -and $createExec.Ok -and $execCount -eq 1) "executors=$execCount"

$contourId = $contours[0].id
$req = (Invoke-Api "POST" "/api/requests" @{ title = "Checkup plan"; contour_id = $contourId } $custTok).Body | ConvertFrom-Json
Record "Create draft request" ($req.status -eq "draft") $req.status

$wids = @($works[0].id, $works[1].id)
Invoke-Api "POST" "/api/requests/$($req.id)/tasks" @{ work_ids = $wids } $custTok | Out-Null
$dup = Invoke-Api "POST" "/api/requests/$($req.id)/tasks" @{ work_ids = @($works[0].id) } $custTok
Record "Duplicate work blocked" ($dup.Status -eq 400) "status=$($dup.Status)"

$sub = (Invoke-Api "POST" "/api/requests/$($req.id)/submit" $null $custTok).Body | ConvertFrom-Json
Record "Submit -> submitted" ($sub.status -eq "submitted") $sub.status

$tasks = (Invoke-Api "GET" "/api/tasks" $null $execTok).Body | ConvertFrom-Json
Record "Executor has assigned tasks" ($tasks.Count -ge 2) "count=$($tasks.Count)"

$tid = $tasks[0].id
Invoke-Api "PUT" "/api/tasks/$tid/status" @{ status = "in_progress" } $execTok | Out-Null
$badTrans = Invoke-Api "PUT" "/api/tasks/$tid/status" @{ status = "pending" } $execTok
Record "Invalid status transition" ($badTrans.Status -eq 400) "status=$($badTrans.Status)"
Invoke-Api "PUT" "/api/tasks/$tid/status" @{ status = "completed" } $execTok | Out-Null
foreach ($t in $tasks) {
    if ($t.id -ne $tid) {
        Invoke-Api "PUT" "/api/tasks/$($t.id)/status" @{ status = "in_progress" } $execTok | Out-Null
        Invoke-Api "PUT" "/api/tasks/$($t.id)/status" @{ status = "completed" } $execTok | Out-Null
    }
}

$final = (Invoke-Api "GET" "/api/requests/$($req.id)" $null $custTok).Body | ConvertFrom-Json
Record "All tasks done -> request completed" ($final.status -eq "completed") $final.status
Record "Total hours > 0" ($final.total_hours -gt 0) "hours=$($final.total_hours)"

$rep = Invoke-Api "GET" "/api/requests/$($req.id)/report?format=json" $null $custTok
Record "Report JSON" ($rep.Ok -and $rep.Status -eq 200) ""
try {
    $pdf = Invoke-WebRequest "$Base/api/requests/$($req.id)/report?format=pdf" -Headers @{ Authorization = "Bearer $custTok" } -UseBasicParsing
    Record "Report PDF" ($pdf.Headers["Content-Type"] -like "*pdf*" -and $pdf.RawContentLength -gt 200)
} catch { Record "Report PDF" $false $_.Exception.Message }

$execView = Invoke-Api "GET" "/api/requests/$($req.id)" $null $execTok
Record "Executor read-only request" ($execView.Ok -and $execView.Status -eq 200) ""

$custOnAdmin = Invoke-Api "GET" "/api/admin/users" $null $custTok
Record "Role middleware (customer!=admin)" ($custOnAdmin.Status -eq 403) "status=$($custOnAdmin.Status)"

$reqLogs = (Invoke-Api "GET" "/api/admin/request-logs?request_id=$($req.id)" $null $adminTok).Body | ConvertFrom-Json
Record "Admin request logs" ($reqLogs.Count -ge 2) "count=$($reqLogs.Count)"

$taskLogs = (Invoke-Api "GET" "/api/admin/task-logs?request_id=$($req.id)" $null $adminTok).Body | ConvertFrom-Json
Record "Admin task logs" ($taskLogs.Count -ge 2) "count=$($taskLogs.Count)"

Write-Host "`n=== SUMMARY: $passed passed, $failed failed ===" -ForegroundColor Cyan
$results | Format-Table -AutoSize
exit $(if ($failed -gt 0) { 1 } else { 0 })
