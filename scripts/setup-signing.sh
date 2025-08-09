#!/bin/bash
set -e

echo "Setting up package signing for RepoBird CLI..."

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check for GPG
if ! command_exists gpg; then
    echo "❌ GPG is required for package signing. Please install gpg:"
    echo "  macOS: brew install gnupg"
    echo "  Ubuntu/Debian: sudo apt install gnupg"
    echo "  RHEL/Fedora: sudo dnf install gnupg2"
    exit 1
fi

# Check for cosign (for container signing)
if ! command_exists cosign; then
    echo "⚠️  Cosign not found. Installing cosign for container signing..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if command_exists brew; then
            brew install sigstore/tap/cosign
        else
            echo "Please install cosign manually from https://github.com/sigstore/cosign/releases"
        fi
    else
        # Linux installation
        COSIGN_VERSION=$(curl -s https://api.github.com/repos/sigstore/cosign/releases/latest | grep '"tag_name":' | cut -d'"' -f4)
        curl -O -L "https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64"
        sudo mv cosign-linux-amd64 /usr/local/bin/cosign
        sudo chmod +x /usr/local/bin/cosign
    fi
fi

echo "✓ Signing tools installed"

# Generate GPG key if it doesn't exist
if ! gpg --list-secret-keys --keyid-format LONG | grep -q "sec"; then
    echo "📄 No GPG key found. Creating one for package signing..."
    
    # Interactive GPG key generation
    cat << EOF | gpg --batch --generate-key
%no-protection
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: RepoBird CLI Release
Name-Email: releases@repobird.ai
Expire-Date: 2y
%commit
EOF
    
    echo "✓ GPG key generated"
else
    echo "✓ GPG key already exists"
fi

# Get the GPG key ID
GPG_KEY_ID=$(gpg --list-secret-keys --keyid-format LONG | grep sec | head -1 | sed 's|.*/||' | sed 's| .*||')
echo "📋 GPG Key ID: $GPG_KEY_ID"

# Export public key
echo "📤 Exporting public key..."
gpg --armor --export $GPG_KEY_ID > repobird-signing-key.asc
echo "✓ Public key exported to repobird-signing-key.asc"

# Create environment variables template
cat << EOF > .env.signing.template
# Package Signing Environment Variables
# Copy this to .env and fill in the values

# GPG Signing
GPG_FINGERPRINT=$GPG_KEY_ID
GPG_PRIVATE_KEY_FILE=~/.gnupg/secring.gpg

# GitHub Tokens (for package repositories)
HOMEBREW_TAP_GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
SCOOP_GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx

# Package Manager API Keys
CHOCOLATEY_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxx
SNAPCRAFT_STORE_CREDENTIALS=base64encoded_credentials

# AUR SSH Key (base64 encoded)
AUR_KEY=base64encoded_ssh_private_key

# Container Signing (Cosign)
COSIGN_PRIVATE_KEY=path/to/cosign.key
COSIGN_PASSWORD=your_cosign_password

EOF

echo "📝 Environment template created: .env.signing.template"

# Create signing script
cat << 'EOF' > scripts/sign-packages.sh
#!/bin/bash
set -e

# Load environment variables
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

echo "🔐 Signing packages..."

# Sign release artifacts
if [ -f dist/checksums.txt ]; then
    echo "  - Signing checksums"
    gpg --batch --local-user "$GPG_FINGERPRINT" --armor --detach-sign dist/checksums.txt
fi

# Sign all binary archives
for file in dist/*.tar.gz dist/*.zip; do
    if [ -f "$file" ]; then
        echo "  - Signing $(basename $file)"
        gpg --batch --local-user "$GPG_FINGERPRINT" --armor --detach-sign "$file"
    fi
done

echo "✓ Package signing complete"
EOF

chmod +x scripts/sign-packages.sh

# Create verification script
cat << 'EOF' > scripts/verify-packages.sh
#!/bin/bash
set -e

echo "🔍 Verifying package signatures..."

# Import public key
if [ -f repobird-signing-key.asc ]; then
    gpg --import repobird-signing-key.asc 2>/dev/null || true
fi

# Verify checksums signature
if [ -f dist/checksums.txt.asc ]; then
    echo "  - Verifying checksums signature"
    gpg --verify dist/checksums.txt.asc dist/checksums.txt
fi

# Verify binary signatures
for sig_file in dist/*.asc; do
    if [ -f "$sig_file" ] && [[ "$sig_file" != *"checksums"* ]]; then
        original_file="${sig_file%.asc}"
        if [ -f "$original_file" ]; then
            echo "  - Verifying $(basename $original_file)"
            gpg --verify "$sig_file" "$original_file"
        fi
    fi
done

echo "✓ All signatures verified"
EOF

chmod +x scripts/verify-packages.sh

# Create APT repository signing setup
cat << 'EOF' > scripts/setup-apt-repo.sh
#!/bin/bash
set -e

# This script sets up an APT repository with proper signing
# Run this on your repository hosting server

REPO_DIR="/var/www/apt"
KEYRING_DIR="$REPO_DIR/keyring"

echo "🏗️  Setting up APT repository..."

# Create repository structure
sudo mkdir -p $REPO_DIR/{dists/stable/main/binary-{amd64,arm64},pool/main}
sudo mkdir -p $KEYRING_DIR

# Copy GPG public key to keyring
sudo cp repobird-signing-key.asc $KEYRING_DIR/

# Create APT configuration
cat << APTCONF | sudo tee $REPO_DIR/conf/distributions
Origin: RepoBird
Label: RepoBird CLI Repository
Codename: stable
Architectures: amd64 arm64
Components: main
Description: RepoBird CLI official repository
SignWith: $GPG_FINGERPRINT
APTCONF

echo "✓ APT repository configured"
echo "📋 Add this to your sources.list:"
echo "deb [signed-by=/usr/share/keyrings/repobird-archive-keyring.gpg] https://apt.repobird.ai stable main"
EOF

chmod +x scripts/setup-apt-repo.sh

echo ""
echo "🎉 Package signing setup complete!"
echo ""
echo "Next steps:"
echo "1. Copy .env.signing.template to .env and fill in your tokens/keys"
echo "2. Test signing: ./scripts/sign-packages.sh"
echo "3. Test verification: ./scripts/verify-packages.sh"
echo "4. For APT repository: ./scripts/setup-apt-repo.sh"
echo ""
echo "⚠️  Important: Keep your private keys secure and never commit them to version control!"