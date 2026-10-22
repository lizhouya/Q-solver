# 默认只检查；关闭相关编辑器、终端和 Git 客户端后，从仓库外以 -Apply 执行。
[CmdletBinding()]
param([switch]$Apply)

$ErrorActionPreference = 'Stop'
$SourceRoot = 'E:\DATA\求职_AI'
$TargetRoot = Join-Path $SourceRoot '求职笔试\Q-Solver'
$RelativeReportDir = '_dev\changes\011-job-workspace-layout'
$Entries = @(
    '.git', '.github', '.gitignore', 'AGENT.md', 'AGENTS.md', 'CLAUDE.md',
    'LICENSE', 'README.md', 'README_EN.md', '_dev', 'app', 'assets', 'build',
    'config', 'frontend', 'go.mod', 'go.sum', 'main.go', 'pkg', 'script', 'wails.json'
)
$Reserved = @('求职笔试', '求职网申', '输出', '自动网申')
$SkipHashDirs = @('.git', 'frontend\node_modules', 'frontend\dist', 'build\bin') + $Reserved

function Invoke-RepoGit {
    param([string]$Root, [string[]]$GitArgs)
    $lines = @(& git -C $Root @GitArgs)
    if ($LASTEXITCODE -ne 0) { throw "Git 命令失败：git $($GitArgs -join ' ')" }
    return $lines
}

function Show-Precheck {
    param([string]$Root)
    Invoke-RepoGit -Root $Root -GitArgs @('status') | Out-Host
    Invoke-RepoGit -Root $Root -GitArgs @('branch') | Out-Host
    Invoke-RepoGit -Root $Root -GitArgs @('remote', '-v') | Out-Host
    # 读取完整差异但不落盘或打印，避免扩散可能存在的私人内容。
    $null = Invoke-RepoGit -Root $Root -GitArgs @('diff')
}

function Get-SourceHashes {
    $hashes = @{}
    $queue = New-Object 'System.Collections.Generic.Queue[string]'
    $queue.Enqueue($SourceRoot)
    while ($queue.Count -gt 0) {
        $dir = $queue.Dequeue()
        foreach ($item in Get-ChildItem -LiteralPath $dir -Force) {
            $relative = $item.FullName.Substring($SourceRoot.Length + 1)
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
                throw "发现重解析点，停止自动迁移：$relative"
            }
            if ($item.PSIsContainer) {
                if ($SkipHashDirs -notcontains $relative) { $queue.Enqueue($item.FullName) }
            } else {
                $hashes[$relative] = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash
            }
        }
    }
    foreach ($name in @('HEAD', 'config', 'index', 'packed-refs')) {
        $relative = '.git\' + $name
        $path = Join-Path $SourceRoot $relative
        if (Test-Path -LiteralPath $path -PathType Leaf) {
            $hashes[$relative] = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
        }
    }
    return $hashes
}

function Move-ExactEntry {
    param([string]$From, [string]$To)
    if (Test-Path -LiteralPath $To) { throw "目标已存在，禁止覆盖：$To" }
    $item = Get-Item -LiteralPath $From -Force
    if ($item.PSIsContainer) { [IO.Directory]::Move($From, $To) }
    else { [IO.File]::Move($From, $To) }
}

if (-not (Test-Path -LiteralPath (Join-Path $SourceRoot '.git') -PathType Container)) {
    if (Test-Path -LiteralPath (Join-Path $TargetRoot '.git') -PathType Container) {
        foreach ($name in $Entries) {
            if (-not (Test-Path -LiteralPath (Join-Path $TargetRoot $name))) {
                throw "目标有 .git 但缺少 $name，可能是部分迁移；请保留现场，勿重复运行。"
            }
            if (Test-Path -LiteralPath (Join-Path $SourceRoot $name)) {
                throw "原位置仍有 $name，可能是部分迁移；请保留现场核查。"
            }
        }
        Write-Host "全部项目根项已经位于 $TargetRoot。本脚本不重复迁移，请在新目录继续验证。"
        exit 0
    }
    throw '原位置没有 .git，目标也不是已迁移仓库；停止操作。'
}
if (-not (Test-Path -LiteralPath $TargetRoot -PathType Container)) {
    throw "目标目录不存在，请先核实：$TargetRoot"
}
if (@(Get-ChildItem -LiteralPath $TargetRoot -Force).Count -ne 0) {
    throw '目标目录不为空，禁止合并或覆盖。'
}
foreach ($path in @($SourceRoot, (Join-Path $SourceRoot '求职笔试'), $TargetRoot)) {
    $item = Get-Item -LiteralPath $path -Force
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "迁移路径含重解析点，停止：$path"
    }
}
foreach ($item in Get-ChildItem -LiteralPath $SourceRoot -Force) {
    if (($Entries -notcontains $item.Name) -and ($Reserved -notcontains $item.Name)) {
        throw "根目录新增了未审计内容，停止：$($item.Name)"
    }
}
foreach ($name in $Entries) {
    $item = Get-Item -LiteralPath (Join-Path $SourceRoot $name) -Force
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "根项含重解析点，停止：$name"
    }
}
if (Test-Path -LiteralPath (Join-Path $SourceRoot '.git\index.lock')) {
    throw '发现 Git index.lock，请先确认没有正在执行的 Git 操作；本脚本不会删除锁。'
}

Show-Precheck -Root $SourceRoot
$branch = (Invoke-RepoGit -Root $SourceRoot -GitArgs @('branch', '--show-current')) -join ''
if ($branch -ne 'dev/qsolver-custom') { throw "当前分支不是预期分支：$branch" }
$head = (Invoke-RepoGit -Root $SourceRoot -GitArgs @('rev-parse', 'HEAD')) -join ''
$statusBefore = (Invoke-RepoGit -Root $SourceRoot -GitArgs @('status', '--porcelain=v1', '-uall')) -join "`n"
$stagedBefore = (Invoke-RepoGit -Root $SourceRoot -GitArgs @('diff', '--cached', '--binary')) -join "`n"

Write-Host "源目录：$SourceRoot"
Write-Host "目标目录：$TargetRoot"
Write-Host '仅移动当前 Q-Solver 根项；不移动输出、网申或其他笔试工具。'
if (-not $Apply) {
    Write-Host 'CHECK PASS：本次仅检查，没有移动文件。目录可移动性仍需实际执行才能确认。'
    Write-Host '关闭相关 VS Code、终端和 Git 客户端后，从仓库外以 -Apply 执行。'
    exit 0
}

$hashes = Get-SourceHashes
$moved = New-Object 'System.Collections.Generic.List[string]'
try {
    # .git 最先移动；如果仍被拒绝，其余源码完全不动。
    foreach ($name in $Entries) {
        Move-ExactEntry -From (Join-Path $SourceRoot $name) -To (Join-Path $TargetRoot $name)
        $moved.Add($name)
        Write-Host "已移动：$name"
    }
} catch {
    $moveError = $_.Exception.Message
    for ($i = $moved.Count - 1; $i -ge 0; $i--) {
        $name = $moved[$i]
        try {
            Move-ExactEntry -From (Join-Path $TargetRoot $name) -To (Join-Path $SourceRoot $name)
        } catch {
            throw "移动失败且回退 $name 失败。不要覆盖或清理任何目录，请保留现场。移动错误：$moveError；回退错误：$($_.Exception.Message)"
        }
    }
    throw "迁移失败，已移动的根项全部退回原位；未改 ACL 或终止进程。请排查占用/权限后再试。原错误：$moveError"
}

# 先验证字节，再执行可能刷新 Git 索引的只读预检。
foreach ($relative in $hashes.Keys) {
    $path = Join-Path $TargetRoot $relative
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "文件已迁入但缺少校验文件：$relative。请停止后续操作并保留现场。"
    }
    if ((Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash -ne $hashes[$relative]) {
        throw "文件已迁入但哈希不一致：$relative。可能存在并发编辑，请保留现场核查。"
    }
}
Show-Precheck -Root $TargetRoot
$headAfter = (Invoke-RepoGit -Root $TargetRoot -GitArgs @('rev-parse', 'HEAD')) -join ''
$statusAfter = (Invoke-RepoGit -Root $TargetRoot -GitArgs @('status', '--porcelain=v1', '-uall')) -join "`n"
$stagedAfter = (Invoke-RepoGit -Root $TargetRoot -GitArgs @('diff', '--cached', '--binary')) -join "`n"
if (($headAfter -ne $head) -or ($statusAfter -ne $statusBefore) -or ($stagedAfter -ne $stagedBefore)) {
    throw '文件已迁入，但 Git 状态与原位置不一致。请停止后续操作并保留现场核查。'
}

$legacy = Join-Path $SourceRoot '自动网申'
$legacyState = 'NOT PRESENT'
if (Test-Path -LiteralPath $legacy) {
    $legacyItem = Get-Item -LiteralPath $legacy -Force
    if ($legacyItem.PSIsContainer -and (($legacyItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -eq 0) -and (@(Get-ChildItem -LiteralPath $legacy -Force).Count -eq 0)) {
        try { [IO.Directory]::Delete($legacy, $false); $legacyState = 'REMOVED EMPTY DIRECTORY' }
        catch { $legacyState = 'BLOCKED'; Write-Warning '旧自动网申空目录仍被占用，已保留；仓库迁移不回退。' }
    } else {
        $legacyState = 'PRESERVED: NOT EMPTY OR REPARSE POINT'
        Write-Warning '自动网申不再是普通空目录，已保留，未删除任何内容。'
    }
}

# 在所有迁移一致性校验完成后，只更新已批准的当前本地路径，不重写历史说明。
$claudePath = Join-Path $TargetRoot 'CLAUDE.md'
$text = [IO.File]::ReadAllText($claudePath)
$oldLine = '- **本地仓库**：`E:\DATA\求职_AI`。'
$newLine = '- **本地仓库**：`E:\DATA\求职_AI\求职笔试\Q-Solver`。'
$utf8 = New-Object System.Text.UTF8Encoding($false)
if ($text.Contains($oldLine)) { [IO.File]::WriteAllText($claudePath, $text.Replace($oldLine, $newLine), $utf8) }
else { Write-Warning 'CLAUDE.md 本地路径行与预期不同，未自动改写，请人工核对。' }

$report = [ordered]@{
    time = (Get-Date).ToString('o')
    source = $SourceRoot
    target = $TargetRoot
    branch = $branch
    head = $head
    moved = @($moved.ToArray())
    verifiedHashCount = $hashes.Count
    gitStatePreservedBeforePathUpdate = $true
    legacyEmptyDirectory = $legacyState
    status = 'MIGRATED; post-migration build and GUI acceptance not run'
}
$reportPath = Join-Path (Join-Path $TargetRoot $RelativeReportDir) ('migration-result-' + (Get-Date -Format 'yyyyMMdd-HHmmss') + '.json')
if (Test-Path -LiteralPath $reportPath) { throw '校验完成，但结果文件已存在；未覆盖旧结果。' }
[IO.File]::WriteAllText($reportPath, ($report | ConvertTo-Json -Depth 5), $utf8)
Write-Host "迁移和一致性校验完成：$TargetRoot"
Write-Host "已验证 $($hashes.Count) 项文件哈希，保留原 Git 历史、暂存和未提交改动。"
Write-Host '请在 VS Code 打开新项目目录，再执行迁移后测试/构建并更新 Change 状态。'
Write-Host '没有执行暂存、commit、push、ACL 修改或进程终止。'
