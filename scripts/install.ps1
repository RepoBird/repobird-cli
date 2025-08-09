# RepoBird CLI Windows Installer
# Usage: iwr -useb https://get.repobird.ai/windows | iex
# Or: Invoke-WebRequest -Uri https://get.repobird.ai/windows -UseBasicParsing | Invoke-Expression

param(
    [string]$InstallDir = "$env:USERPROFILE\.local\bin",
    [string]$Version = "latest",
    [switch]$Force
)

# Constants
$GITHUB_REPO = "repobird/repobird-cli"
$BINARY_NAME = "repobird"

# Colors
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"

function Write-Log {
    param([string]$Message, [string]$Color = "White")
    Write-Host "[INFO] $Message" -ForegroundColor $Color
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor $Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Red -ErrorAction Continue
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) {
        $arch = $env:PROCESSOR_ARCHITEW6432
    }
    
    switch ($arch) {
        "AMD64" { return "amd64" }
        "X86" { return "386" }
        "ARM64" { return "amd64" }  # Fallback to amd64 for ARM64
        default { 
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Get-LatestVersion {
    try {
        $apiUrl = "https://api.github.com/repos/$GITHUB_REPO/releases/latest"
        $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
        return $response.tag_name
    } catch {
        Write-Error "Failed to get latest version: $($_.Exception.Message)"
        return $null
    }
}

function Test-AdminRights {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Install-Binary {
    param(
        [string]$Version,
        [string]$InstallPath
    )
    
    $arch = Get-Architecture
    $platform = "windows_$arch"
    
    Write-Log "Detected platform: $platform" -Color $Blue
    Write-Log "Target version: $Version" -Color $Blue
    Write-Log "Install directory: $InstallPath" -Color $Blue
    
    # Create temporary directory
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    
    try {
        # Construct download URL
        $filename = "${BINARY_NAME}_${platform}.zip"
        $downloadUrl = "https://github.com/$GITHUB_REPO/releases/download/$Version/$filename"
        $archivePath = Join-Path $tempDir.FullName $filename
        
        Write-Log "Downloading from: $downloadUrl" -Color $Green
        
        # Download
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        } catch {
            Write-Error "Download failed: $($_.Exception.Message)"
            return $false
        }
        
        # Verify download
        if (-not (Test-Path $archivePath)) {
            Write-Error "Download failed: $archivePath not found"
            return $false
        }
        
        Write-Log "Download complete, extracting..." -Color $Green
        
        # Extract
        try {
            Expand-Archive -Path $archivePath -DestinationPath $tempDir.FullName -Force
        } catch {
            Write-Error "Extraction failed: $($_.Exception.Message)"
            return $false
        }
        
        # Find the binary
        $binaryPath = Get-ChildItem -Path $tempDir.FullName -Name "${BINARY_NAME}.exe" -Recurse | Select-Object -First 1
        if (-not $binaryPath) {
            Write-Error "Binary not found in archive"
            return $false
        }
        
        $sourceBinary = Join-Path $tempDir.FullName $binaryPath
        
        # Create install directory
        if (-not (Test-Path $InstallPath)) {
            New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
        }
        
        # Install binary
        $targetBinary = Join-Path $InstallPath "${BINARY_NAME}.exe"
        
        # Handle existing file
        if (Test-Path $targetBinary) {
            if (-not $Force) {
                $choice = Read-Host "Binary already exists. Overwrite? (y/N)"
                if ($choice -ne 'y' -and $choice -ne 'Y') {
                    Write-Log "Installation cancelled by user"
                    return $false
                }
            }
            
            # Try to stop any running processes
            Get-Process -Name $BINARY_NAME -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 1
        }
        
        try {
            Copy-Item $sourceBinary $targetBinary -Force
        } catch {
            Write-Error "Failed to copy binary: $($_.Exception.Message)"
            Write-Error "Try running as Administrator or use -Force parameter"
            return $false
        }
        
        Write-Log "✓ Installed $BINARY_NAME to $targetBinary" -Color $Green
        return $true
        
    } finally {
        # Cleanup
        Remove-Item $tempDir.FullName -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Update-Path {
    param([string]$InstallPath)
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    
    if ($currentPath -notlike "*$InstallPath*") {
        Write-Warn "$InstallPath is not in your PATH"
        
        try {
            $newPath = "$InstallPath;$currentPath"
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
            $env:PATH = "$InstallPath;$env:PATH"
            Write-Log "✓ Added $InstallPath to your PATH" -Color $Green
            Write-Log "Please restart your terminal to use the updated PATH" -Color $Yellow
        } catch {
            Write-Warn "Failed to update PATH automatically. Please add manually:"
            Write-Host "  1. Open System Properties > Advanced > Environment Variables"
            Write-Host "  2. Add '$InstallPath' to your user PATH variable"
        }
    } else {
        Write-Log "✓ $InstallPath is already in PATH" -Color $Green
    }
}

function Test-Installation {
    param([string]$InstallPath)
    
    $binaryPath = Join-Path $InstallPath "${BINARY_NAME}.exe"
    
    if (Test-Path $binaryPath) {
        Write-Log "Installation verified!" -Color $Green
        
        try {
            $versionOutput = & $binaryPath version 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Log "Version: $versionOutput" -Color $Blue
            }
        } catch {
            Write-Warn "Could not run version command"
        }
        
        Write-Host ""
        Write-Host "🎉 RepoBird CLI installed successfully!" -ForegroundColor $Green
        Write-Host ""
        Write-Host "Get started:"
        Write-Host "  1. Configure your API key: repobird config set api-key YOUR_KEY"
        Write-Host "  2. Run your first task: repobird run task.json"
        Write-Host "  3. Check status: repobird status"
        Write-Host ""
        Write-Host "Documentation: https://docs.repobird.ai"
        Write-Host "Issues: https://github.com/$GITHUB_REPO/issues"
        
        return $true
    } else {
        Write-Error "Installation verification failed"
        return $false
    }
}

function Show-Banner {
    Write-Host @"
    ____                ____  _         _ 
   |  _ \ ___ _ __   ___| __ )(_)_ __ __| |
   | |_) / _ \ '_ \ / _ \  _ \| | '__/ _` |
   |  _ <  __/ |_) | (_) |_) | | | | (_| |
   |_| \_\___| .__/ \___/____/|_|_|  \__,_|
             |_|                          
   
   RepoBird CLI Installer (Windows)
"@ -ForegroundColor $Blue
}

# Main installation flow
function Main {
    Show-Banner
    
    Write-Log "Starting RepoBird CLI installation..." -Color $Green
    
    # Check for existing installation
    $existingBinary = Join-Path $InstallDir "${BINARY_NAME}.exe"
    if (Test-Path $existingBinary) {
        try {
            $currentVersion = & $existingBinary version 2>$null | Select-Object -First 1
            Write-Warn "RepoBird CLI is already installed: $currentVersion"
            Write-Host "This will update your existing installation."
            Write-Host ""
        } catch {
            Write-Warn "RepoBird CLI appears to be installed but version check failed"
        }
    }
    
    # Check if running as admin (optional but recommended)
    if (-not (Test-AdminRights)) {
        Write-Warn "Not running as Administrator. Installation may fail if directory is protected."
        Write-Host "Consider running PowerShell as Administrator for best results."
        Write-Host ""
    }
    
    # Get version
    if ($Version -eq "latest") {
        $Version = Get-LatestVersion
        if (-not $Version) {
            Write-Error "Could not determine latest version"
            exit 1
        }
    }
    
    # Install binary
    $success = Install-Binary -Version $Version -InstallPath $InstallDir
    if (-not $success) {
        exit 1
    }
    
    # Update PATH
    Update-Path -InstallPath $InstallDir
    
    # Verify installation
    $verified = Test-Installation -InstallPath $InstallDir
    if (-not $verified) {
        exit 1
    }
}

# Run the installer
try {
    Main
} catch {
    Write-Error "Installation failed: $($_.Exception.Message)"
    Write-Host ""
    Write-Host "Manual installation options:"
    Write-Host "  1. Download from: https://github.com/$GITHUB_REPO/releases"
    Write-Host "  2. Package managers: choco install repobird, scoop install repobird"
    Write-Host "  3. Build from source: git clone https://github.com/$GITHUB_REPO"
    exit 1
}