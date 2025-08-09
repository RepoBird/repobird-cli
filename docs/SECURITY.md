# Security - RepoBird CLI

## API Key Storage

The RepoBird CLI uses a multi-layered approach to securely store your API key, automatically choosing the most secure method available on your system.

### Storage Methods (in order of preference)

#### 1. Environment Variable (Most Secure for CI/CD)
- **Variable**: `REPOBIRD_API_KEY`
- **Best for**: CI/CD pipelines, containers, automated scripts
- **Security**: ✅ Never written to disk
- **Example**:
  ```bash
  export REPOBIRD_API_KEY="your-api-key"
  rb status
  ```

#### 2. System Keyring (Desktop Systems)
- **macOS**: Keychain Access
- **Windows**: Credential Manager  
- **Linux**: GNOME Keyring/KWallet (only on desktop environments)
- **Security**: ✅ OS-level encryption and access control
- **Automatic**: Selected automatically when available

#### 3. Encrypted File (Universal Fallback)
- **Location**: `~/.repobird/.api_key.enc`
- **Encryption**: AES-256-GCM
- **Key Derivation**: Machine-specific (hostname + username + machine ID)
- **Permissions**: 0600 (owner read/write only)
- **Security**: ✅ Encrypted at rest
- **Best for**: Linux servers, headless systems, containers

### Linux-Specific Behavior

On Linux systems, the CLI is conservative about using desktop keyrings:
- **Desktop Environment**: Uses keyring if GNOME/KDE is detected
- **Servers/Containers**: Defaults to encrypted file storage
- **SSH Sessions**: Always uses encrypted file storage
- **Docker/Kubernetes**: Always uses encrypted file storage

This ensures the CLI works reliably on all Linux systems without requiring GUI dependencies.

### Commands

```bash
# Set API key (automatically selects best storage method)
rb config set api-key YOUR_KEY

# Check current storage method
rb config get

# View detailed storage information
rb config get storage

# Delete API key from all storage locations
rb config delete api-key
```

### Security Best Practices

1. **Never commit API keys to version control**
2. **Use environment variables in CI/CD pipelines**
3. **Rotate API keys regularly**
4. **Use different keys for development and production**
5. **Monitor API key usage in the RepoBird dashboard**

### Migration from Plain Text

If you have an existing API key stored in plain text (`~/.repobird/config.yaml`), it will be automatically migrated to secure storage on first use. The plain text version is then removed.

### Troubleshooting

#### API Key Not Found
```bash
# Check if key is set
rb config get api-key

# Verify storage location
rb config get storage
```

#### Permission Denied
```bash
# Fix permissions on config directory
chmod 700 ~/.repobird
chmod 600 ~/.repobird/.api_key.enc
```

#### Encrypted File Issues
If the encrypted file becomes corrupted:
```bash
# Delete and re-set the key
rb config delete api-key
rb config set api-key YOUR_KEY
```

### Technical Details

#### Encryption Specifications
- **Algorithm**: AES-256-GCM (Authenticated Encryption)
- **Key Size**: 256 bits
- **Key Derivation**: SHA-256 hash of machine-specific identifiers
- **Nonce**: Random 12 bytes per encryption
- **Authentication**: GCM mode provides built-in authentication

#### Machine-Specific Key Components
The encryption key is derived from:
- Hostname
- Username
- Home directory path
- Machine ID (Linux: `/etc/machine-id`)
- Application salt

This ensures the encrypted file can only be decrypted on the same machine by the same user.

### Compliance

The RepoBird CLI's secure storage implementation follows industry best practices:
- ✅ Encryption at rest
- ✅ Principle of least privilege (0600 file permissions)
- ✅ No plain text storage
- ✅ Automatic secure migration
- ✅ Support for hardware security modules (via OS keyrings)

### Reporting Security Issues

If you discover a security vulnerability, please report it to:
- Email: security@repobird.ai
- Do not create public GitHub issues for security vulnerabilities