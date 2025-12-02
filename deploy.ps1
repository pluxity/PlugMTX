# Set the output encoding of this script to UTF-8 to prevent character corruption.
$OutputEncoding = [System.Text.Encoding]::UTF8

# =======================================================
#               Script Configuration Variables
# =======================================================
# Remote Server Information
$RemoteUser = "pluxity"
$RemoteHost = "192.168.10.181"
$RemotePassword = $env:plx_pw

# 환경 변수가 없는 경우 에러 처리
if ([string]::IsNullOrEmpty($RemotePassword)) {
    Write-Host "Error: Environment variable 'plx_pw' is not set." -ForegroundColor Red
    exit 1
}

# Local File and Image Information
$LocalComposeFile = "docker-compose.mediamtx.yml"
$LocalConfigFile = "mediamtx.yml"
$ImageName = "mediamtx:local"
$ImageTarFile = "mediamtx.tar"

# Remote Server Path and File Information
$RemotePath = "/home/pluxity/docker/aiot"
$RemoteComposeFile = "docker-compose.mediamtx.yml"
$RemoteConfigFile = "mediamtx.yml"

# =======================================================
#               Deployment Script Start
# =======================================================
try {
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "   MediaMTX Deployment Script" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    # Step 1: Build Docker Image
    Write-Host "[Step 1/7] Building Docker image locally..." -ForegroundColor Yellow
    docker build -t $ImageName .
    if ($LASTEXITCODE -ne 0) { throw "Docker image build failed." }
    Write-Host "✓ Image built successfully: $ImageName" -ForegroundColor Green
    Write-Host ""

    # Step 2: Save Docker Image to TAR
    Write-Host "[Step 2/7] Saving Docker image to '$ImageTarFile'..." -ForegroundColor Yellow
    docker save -o $ImageTarFile $ImageName
    if ($LASTEXITCODE -ne 0) { throw "Failed to save Docker image." }

    $TarSize = (Get-Item $ImageTarFile).Length / 1MB
    Write-Host "✓ Image saved: $ImageTarFile ($([math]::Round($TarSize, 2)) MB)" -ForegroundColor Green
    Write-Host ""

    # Step 3: Transfer TAR to Remote Server
    Write-Host "[Step 3/7] Transferring '$ImageTarFile' to remote server..." -ForegroundColor Yellow
    Write-Host "   → Destination: $($RemoteUser)@$($RemoteHost):$($RemotePath)" -ForegroundColor Gray
    sshpass -p $RemotePassword scp $ImageTarFile ($RemoteUser + "@" + $RemoteHost + ":" + $RemotePath)
    if ($LASTEXITCODE -ne 0) { throw "Failed to transfer the image file to the remote server." }
    Write-Host "✓ Image transferred successfully" -ForegroundColor Green
    Write-Host ""

    # Step 4: Transfer docker-compose file
    Write-Host "[Step 4/7] Transferring docker-compose file to remote server..." -ForegroundColor Yellow
    sshpass -p $RemotePassword scp $LocalComposeFile ($RemoteUser + "@" + $RemoteHost + ":" + $RemotePath + "/" + $RemoteComposeFile)
    if ($LASTEXITCODE -ne 0) { throw "Failed to transfer docker-compose file to the remote server." }
    Write-Host "✓ docker-compose file transferred" -ForegroundColor Green
    Write-Host ""

    # Step 5: Transfer config file
    Write-Host "[Step 5/7] Transferring configuration file to remote server..." -ForegroundColor Yellow
    sshpass -p $RemotePassword scp $LocalConfigFile ($RemoteUser + "@" + $RemoteHost + ":" + $RemotePath + "/" + $RemoteConfigFile)
    if ($LASTEXITCODE -ne 0) { throw "Failed to transfer config file to the remote server." }
    Write-Host "✓ Configuration file transferred" -ForegroundColor Green
    Write-Host ""

    # Step 6: Transfer .env file if exists
#     if (Test-Path ".env") {
#         Write-Host "[Step 6/7] Transferring .env file to remote server..." -ForegroundColor Yellow
#         sshpass -p $RemotePassword scp ".env" ($RemoteUser + "@" + $RemoteHost + ":" + $RemotePath + "/.env")
#         if ($LASTEXITCODE -ne 0) {
#             Write-Host "⚠ Warning: Failed to transfer .env file" -ForegroundColor Yellow
#         } else {
#             Write-Host "✓ .env file transferred" -ForegroundColor Green
#         }
#     } else {
#         Write-Host "[Step 6/7] No .env file found, skipping..." -ForegroundColor Gray
#     }
#     Write-Host ""

    # Step 7: Deploy on Remote Server
    Write-Host "[Step 7/7] Deploying container on remote server..." -ForegroundColor Yellow

    # Define the commands to be executed on the remote server.
    $RemoteCommands = "cd $RemotePath; " +
                      "echo '→ Stopping existing container...'; " +
                      "docker compose -f $RemoteComposeFile down; " +
                      "echo '→ Loading new image...'; " +
                      "docker load -i $ImageTarFile; " +
                      "echo '→ Starting new container...'; " +
                      "docker compose -f $RemoteComposeFile up -d; " +
                      "echo '→ Cleaning up tar file...'; " +
                      "rm -f $ImageTarFile; " +
                      "echo ''; " +
                      "echo '✓ Deployment completed!'; " +
                      "echo ''; " +
                      "echo 'Container status:'; " +
                      "docker compose -f $RemoteComposeFile ps; " +
                      "echo ''; " +
                      "echo 'Following logs (Ctrl+C to exit):'; " +
                      "docker compose -f $RemoteComposeFile logs -f"

    # Execute the remote commands via ssh.
    sshpass -p $RemotePassword ssh -t ($RemoteUser + "@" + $RemoteHost) $RemoteCommands

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "   Deployment Completed Successfully!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host ""

} catch {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "   Deployment Failed!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host ""

    # Cleanup local tar file on error
    if (Test-Path $ImageTarFile) {
        Write-Host "Cleaning up local tar file..." -ForegroundColor Yellow
        Remove-Item $ImageTarFile -Force
    }

    exit 1
} finally {
    # Cleanup local tar file
    if (Test-Path $ImageTarFile) {
        Write-Host "Cleaning up local tar file..." -ForegroundColor Gray
        Remove-Item $ImageTarFile -Force
        Write-Host "✓ Cleanup completed" -ForegroundColor Green
    }
}
