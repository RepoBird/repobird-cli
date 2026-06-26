#!/bin/bash
# local-package-publish.sh - Publish to package managers locally
# Updates Homebrew, Scoop, Chocolatey, AUR, etc. without GitHub Actions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

# Parse arguments
VERSION=""
PACKAGE_MANAGERS=()
DRY_RUN=false
OVERRIDE_VERSION=""

show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --homebrew            Update Homebrew tap"
    echo "  --scoop               Update Scoop bucket"
    echo "  --chocolatey          Update Chocolatey package"
    echo "  --aur                 Update AUR package"
    echo "  --apt                 Update APT repository"
    echo "  --yum                 Update YUM repository"
    echo "  --all                 Update all package managers"
    echo "  --dry-run             Show what would be done without making changes"
    echo "  --version VERSION     Override VERSION file (not recommended)"
    echo "  --help, -h            Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Update Homebrew only (uses VERSION file)"
    echo "  $0 --homebrew"
    echo ""
    echo "  # Update all package managers"
    echo "  $0 --all"
    echo ""
    echo "  # Dry run for Homebrew and Scoop"
    echo "  $0 --homebrew --scoop --dry-run"
    echo ""
    echo "  # Override version (not recommended)"
    echo "  $0 --version v1.2.3 --homebrew"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            OVERRIDE_VERSION="$2"
            print_warning "Overriding VERSION file with $2"
            shift 2
            ;;
        --homebrew)
            PACKAGE_MANAGERS+=("homebrew")
            shift
            ;;
        --scoop)
            PACKAGE_MANAGERS+=("scoop")
            shift
            ;;
        --chocolatey)
            PACKAGE_MANAGERS+=("chocolatey")
            shift
            ;;
        --aur)
            PACKAGE_MANAGERS+=("aur")
            shift
            ;;
        --apt)
            PACKAGE_MANAGERS+=("apt")
            shift
            ;;
        --yum)
            PACKAGE_MANAGERS+=("yum")
            shift
            ;;
        --all)
            PACKAGE_MANAGERS=("homebrew" "scoop" "chocolatey" "aur" "apt" "yum")
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Get version from VERSION file or override
if [ -n "$OVERRIDE_VERSION" ]; then
    VERSION="$OVERRIDE_VERSION"
elif [ -f VERSION ]; then
    VERSION=$(cat VERSION)
    print_info "Using version from VERSION file: $VERSION"
else
    print_error "VERSION file not found and no version specified"
    echo "Please create a VERSION file with the version number (e.g., v1.2.3)"
    echo "Or use --version to override"
    exit 1
fi

if [ ${#PACKAGE_MANAGERS[@]} -eq 0 ]; then
    print_error "At least one package manager must be specified"
    show_help
    exit 1
fi

# Ensure version starts with 'v'
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

CLEAN_VERSION=${VERSION#v}

print_step "Package Manager Publisher"
echo "Version: $VERSION ($CLEAN_VERSION)"
echo "Package managers: ${PACKAGE_MANAGERS[*]}"
echo "Dry run: $DRY_RUN"
echo ""

# Get checksums from GitHub release
get_checksum() {
    local file=$1
    local checksums_url="https://github.com/repobird/repobird-cli/releases/download/$VERSION/checksums.txt"
    
    print_info "Fetching checksum for $file"
    
    # Download checksums file
    if command -v curl &> /dev/null; then
        CHECKSUM=$(curl -sL "$checksums_url" | grep "$file" | awk '{print $1}')
    elif command -v wget &> /dev/null; then
        CHECKSUM=$(wget -qO- "$checksums_url" | grep "$file" | awk '{print $1}')
    else
        print_error "Neither curl nor wget found"
        return 1
    fi
    
    if [ -z "$CHECKSUM" ]; then
        print_error "Could not find checksum for $file"
        return 1
    fi
    
    echo "$CHECKSUM"
}

# Update Homebrew
update_homebrew() {
    print_step "Updating Homebrew tap"
    
    local TAP_REPO="homebrew-tap"
    local TAP_PATH="/tmp/repobird-homebrew-tap"
    
    # Clone or update tap repository
    if [ -d "$TAP_PATH" ]; then
        cd "$TAP_PATH"
        git pull
    else
        git clone "https://github.com/repobird/$TAP_REPO.git" "$TAP_PATH"
        cd "$TAP_PATH"
    fi
    
    # Get checksums for macOS binaries
    DARWIN_AMD64_SHA=$(get_checksum "repobird_darwin_amd64.tar.gz")
    DARWIN_ARM64_SHA=$(get_checksum "repobird_darwin_arm64.tar.gz")
    LINUX_AMD64_SHA=$(get_checksum "repobird_linux_amd64.tar.gz")
    LINUX_ARM64_SHA=$(get_checksum "repobird_linux_arm64.tar.gz")
    
    # Update formula
    cat > Formula/repobird.rb << EOF
class Repobird < Formula
  desc "Fast CLI for RepoBird AI agent platform"
  homepage "https://github.com/repobird/repobird-cli"
  version "$CLEAN_VERSION"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_darwin_amd64.tar.gz"
      sha256 "$DARWIN_AMD64_SHA"
    else
      url "https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_darwin_arm64.tar.gz"
      sha256 "$DARWIN_ARM64_SHA"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_linux_amd64.tar.gz"
      sha256 "$LINUX_AMD64_SHA"
    else
      url "https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_linux_arm64.tar.gz"
      sha256 "$LINUX_ARM64_SHA"
    end
  end

  def install
    bin.install "repobird"
    
    # Install shell completions
    bash_completion.install "completions/repobird.bash" => "repobird"
    zsh_completion.install "completions/_repobird"
    fish_completion.install "completions/repobird.fish"
    
    # Install man pages
    man1.install Dir["man/*.1"]
  end

  test do
    system "#{bin}/repobird", "version"
  end
end
EOF
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Would update Formula/repobird.rb with version $VERSION"
        cat Formula/repobird.rb
    else
        git add Formula/repobird.rb
        git commit -m "Update RepoBird to $VERSION"
        git push
        print_success "Homebrew formula updated"
    fi
    
    cd - > /dev/null
}

# Update Scoop
update_scoop() {
    print_step "Updating Scoop bucket"
    
    local BUCKET_REPO="scoop-bucket"
    local BUCKET_PATH="/tmp/repobird-scoop-bucket"
    
    # Clone or update bucket repository
    if [ -d "$BUCKET_PATH" ]; then
        cd "$BUCKET_PATH"
        git pull
    else
        git clone "https://github.com/repobird/$BUCKET_REPO.git" "$BUCKET_PATH" 2>/dev/null || {
            print_warning "Scoop bucket repository not found, creating manifest only"
            mkdir -p "$BUCKET_PATH"
        }
        cd "$BUCKET_PATH"
    fi
    
    # Get checksums for Windows binaries
    WINDOWS_AMD64_SHA=$(get_checksum "repobird_windows_amd64.zip")
    WINDOWS_386_SHA=$(get_checksum "repobird_windows_386.zip")
    
    # Update manifest
    cat > repobird.json << EOF
{
    "version": "$CLEAN_VERSION",
    "description": "Fast CLI for RepoBird AI agent platform",
    "homepage": "https://github.com/repobird/repobird-cli",
    "license": "MIT",
    "architecture": {
        "64bit": {
            "url": "https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_windows_amd64.zip",
            "hash": "$WINDOWS_AMD64_SHA"
        },
        "32bit": {
            "url": "https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_windows_386.zip",
            "hash": "$WINDOWS_386_SHA"
        }
    },
    "bin": "repobird.exe",
    "checkver": {
        "github": "https://github.com/repobird/repobird-cli"
    },
    "autoupdate": {
        "architecture": {
            "64bit": {
                "url": "https://github.com/repobird/repobird-cli/releases/download/v\$version/repobird_windows_amd64.zip"
            },
            "32bit": {
                "url": "https://github.com/repobird/repobird-cli/releases/download/v\$version/repobird_windows_386.zip"
            }
        }
    }
}
EOF
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Would update repobird.json with version $VERSION"
        cat repobird.json
    else
        if [ -d .git ]; then
            git add repobird.json
            git commit -m "Update RepoBird to $VERSION"
            git push
            print_success "Scoop manifest updated"
        else
            print_info "Scoop manifest created at $BUCKET_PATH/repobird.json"
            print_warning "Manual upload to bucket repository required"
        fi
    fi
    
    cd - > /dev/null
}

# Update Chocolatey
update_chocolatey() {
    print_step "Updating Chocolatey package"
    
    if ! command -v choco &> /dev/null; then
        print_warning "Chocolatey not installed, creating package files only"
    fi
    
    local CHOCO_PATH="/tmp/repobird-chocolatey"
    mkdir -p "$CHOCO_PATH/tools"
    cd "$CHOCO_PATH"
    
    # Get checksum
    WINDOWS_AMD64_SHA=$(get_checksum "repobird_windows_amd64.zip")
    
    # Create nuspec file
    cat > repobird.nuspec << EOF
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>repobird</id>
    <version>$CLEAN_VERSION</version>
    <title>RepoBird CLI</title>
    <authors>RepoBird Team</authors>
    <owners>repobird</owners>
    <licenseUrl>https://github.com/repobird/repobird-cli/blob/main/LICENSE</licenseUrl>
    <projectUrl>https://github.com/repobird/repobird-cli</projectUrl>
    <iconUrl>https://repobird.ai/icon.png</iconUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <description>Fast CLI for RepoBird AI agent platform</description>
    <summary>RepoBird CLI enables users to submit AI-powered code generation tasks and track their progress.</summary>
    <releaseNotes>https://github.com/repobird/repobird-cli/releases/tag/$VERSION</releaseNotes>
    <tags>cli ai code-generation developer-tools</tags>
  </metadata>
  <files>
    <file src="tools\\**" target="tools" />
  </files>
</package>
EOF
    
    # Create install script
    cat > tools/chocolateyinstall.ps1 << 'EOF'
$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url64 = 'https://github.com/repobird/repobird-cli/releases/download/VERSION/repobird_windows_amd64.zip'
$checksum64 = 'CHECKSUM'

$packageArgs = @{
  packageName    = $env:ChocolateyPackageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  checksum64     = $checksum64
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
EOF
    
    # Replace placeholders
    sed -i "s|VERSION|$VERSION|g" tools/chocolateyinstall.ps1
    sed -i "s|CHECKSUM|$WINDOWS_AMD64_SHA|g" tools/chocolateyinstall.ps1
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Would create Chocolatey package version $VERSION"
        echo "Files created in $CHOCO_PATH"
    else
        if command -v choco &> /dev/null; then
            choco pack
            print_success "Chocolatey package created: repobird.$CLEAN_VERSION.nupkg"
            print_info "To publish: choco push repobird.$CLEAN_VERSION.nupkg --source https://push.chocolatey.org/"
        else
            print_info "Chocolatey package files created at $CHOCO_PATH"
            print_warning "Install Chocolatey to build and push the package"
        fi
    fi
    
    cd - > /dev/null
}

# Update AUR
update_aur() {
    print_step "Updating AUR package"
    
    local AUR_PATH="/tmp/repobird-aur"
    
    # Clone or update AUR repository
    if [ -d "$AUR_PATH" ]; then
        cd "$AUR_PATH"
        git pull
    else
        # AUR uses SSH, so we need to handle this differently
        print_info "Creating AUR PKGBUILD locally"
        mkdir -p "$AUR_PATH"
        cd "$AUR_PATH"
    fi
    
    # Get checksum
    LINUX_AMD64_SHA=$(get_checksum "repobird_linux_amd64.tar.gz")
    
    # Create PKGBUILD
    cat > PKGBUILD << EOF
# Maintainer: RepoBird Team <team@repobird.ai>
pkgname=repobird
pkgver=$CLEAN_VERSION
pkgrel=1
pkgdesc="Fast CLI for RepoBird AI agent platform"
arch=('x86_64' 'aarch64')
url="https://github.com/repobird/repobird-cli"
license=('MIT')
depends=('glibc')
source_x86_64=("https://github.com/repobird/repobird-cli/releases/download/v\${pkgver}/repobird_linux_amd64.tar.gz")
sha256sums_x86_64=('$LINUX_AMD64_SHA')

package() {
    cd "\$srcdir"
    
    # Install binary
    install -Dm755 repobird "\$pkgdir/usr/bin/repobird"
    
    # Install completions
    install -Dm644 completions/repobird.bash "\$pkgdir/usr/share/bash-completion/completions/repobird"
    install -Dm644 completions/_repobird "\$pkgdir/usr/share/zsh/site-functions/_repobird"
    install -Dm644 completions/repobird.fish "\$pkgdir/usr/share/fish/vendor_completions.d/repobird.fish"
    
    # Install man pages
    for man in man/*.1; do
        install -Dm644 "\$man" "\$pkgdir/usr/share/man/man1/\$(basename \$man)"
    done
    
    # Install license and docs
    install -Dm644 LICENSE "\$pkgdir/usr/share/licenses/\$pkgname/LICENSE"
    install -Dm644 README.md "\$pkgdir/usr/share/doc/\$pkgname/README.md"
}
EOF
    
    # Create .SRCINFO
    cat > .SRCINFO << EOF
pkgbase = repobird
    pkgdesc = Fast CLI for RepoBird AI agent platform
    pkgver = $CLEAN_VERSION
    pkgrel = 1
    url = https://github.com/repobird/repobird-cli
    arch = x86_64
    arch = aarch64
    license = MIT
    depends = glibc
    source_x86_64 = https://github.com/repobird/repobird-cli/releases/download/v$CLEAN_VERSION/repobird_linux_amd64.tar.gz
    sha256sums_x86_64 = $LINUX_AMD64_SHA

pkgname = repobird
EOF
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Would update AUR package to version $VERSION"
        echo "PKGBUILD created at $AUR_PATH"
    else
        print_success "AUR package files created at $AUR_PATH"
        print_info "To publish to AUR:"
        echo "  1. Clone AUR repo: git clone ssh://aur@aur.archlinux.org/repobird.git"
        echo "  2. Copy PKGBUILD and .SRCINFO to the repo"
        echo "  3. Run: makepkg --printsrcinfo > .SRCINFO"
        echo "  4. Commit and push to AUR"
    fi
    
    cd - > /dev/null
}

# Update APT repository
update_apt() {
    print_step "Updating APT repository"
    
    print_info "APT repository update requires server access"
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Would update APT repository with version $VERSION"
    else
        echo "To update APT repository:"
        echo "  1. Download .deb files from GitHub release"
        echo "  2. Upload to APT server"
        echo "  3. Run: reprepro -b /var/www/apt includedeb stable *.deb"
        print_warning "Manual server access required for APT repository"
    fi
}

# Update YUM repository
update_yum() {
    print_step "Updating YUM repository"
    
    print_info "YUM repository update requires server access"
    
    if [ "$DRY_RUN" = true ]; then
        print_info "Would update YUM repository with version $VERSION"
    else
        echo "To update YUM repository:"
        echo "  1. Download .rpm files from GitHub release"
        echo "  2. Upload to YUM server"
        echo "  3. Run: createrepo /var/www/yum/"
        print_warning "Manual server access required for YUM repository"
    fi
}

# Main execution
for pm in "${PACKAGE_MANAGERS[@]}"; do
    case $pm in
        homebrew)
            update_homebrew
            ;;
        scoop)
            update_scoop
            ;;
        chocolatey)
            update_chocolatey
            ;;
        aur)
            update_aur
            ;;
        apt)
            update_apt
            ;;
        yum)
            update_yum
            ;;
        *)
            print_warning "Unknown package manager: $pm"
            ;;
    esac
done

echo ""
print_success "Package manager updates completed!"

if [ "$DRY_RUN" = true ]; then
    print_info "This was a dry run. No changes were made."
fi