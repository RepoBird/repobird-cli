# Security Policy

## Secure API Key Storage

RepoBird CLI implements multiple layers of security for API key storage, automatically selecting the most secure method available on your system.

### Storage Methods (in order of preference)

#### 1. System Keyring (Most Secure)
- **macOS**: Keychain
- **Windows**: Credential Manager  
- **Linux**: GNOME Keyring, KWallet, or Secret Service (when desktop environment is available)

This is the default and most secure storage method when available.

#### 2. Encrypted File Storage (Secure Fallback)
- Uses AES-256-GCM encryption
- Key derived from machine-specific identifiers
- File permissions set to 0600 (owner read/write only)
- Location: `~/.repobird/.api_key.enc`

Automatically used on headless servers, containers, or when keyring is unavailable.

#### 3. Environment Variable (CI/CD)
- Set `REPOBIRD_API_KEY` environment variable
- Suitable for CI/CD pipelines and containerized environments
- Takes precedence over stored keys for flexibility

### Security Best Practices

#### For Users

1. **Never share your API key**
   - Treat it like a password
   - Don't commit it to version control
   - Don't include it in scripts or documentation

2. **Use the secure login command**
   ```bash
   repobird login
   ```
   This ensures your key is stored using the most secure method available.

3. **Check your storage method**
   ```bash
   repobird info
   ```
   This shows where and how your API key is stored.

4. **Rotate keys regularly**
   - Generate new API keys periodically
   - Remove old keys after rotation

5. **Use environment variables for CI/CD only**
   - Environment variables are suitable for automated systems
   - For development, use `repobird login` instead

#### For Developers

1. **API keys are masked in logs**
   - Debug output shows only first 4 characters
   - Full keys are never logged

2. **Memory clearing**
   - Sensitive data is cleared from memory after use (best effort)
   - Go's garbage collector may retain copies

3. **File permissions**
   - Config files: 0644 (readable, no secrets)
   - Encrypted key files: 0600 (owner only)

4. **No command-line API keys**
   - API keys should not be passed as command arguments
   - They may appear in shell history or process lists

## Reporting Security Vulnerabilities

If you discover a security vulnerability in RepoBird CLI, please report it responsibly:

- **Contact**: https://repobird.ai/contact
- **Response Time**: We aim to respond within 48 hours
- **Please DO NOT**: Create public GitHub issues for security vulnerabilities

We appreciate your help in keeping RepoBird secure for all users.

### Migration from Insecure Storage

If you have an API key stored in plain text (legacy versions), it will be automatically migrated to secure storage on first use. You can manually trigger migration:

```bash
repobird login
```

## Compliance

RepoBird CLI's security implementation follows industry best practices:
- Encryption at rest for stored credentials
- Platform-specific security integration (Keychain, Credential Manager, Keyring)
- Secure defaults with automatic fallback mechanisms
- Clear security warnings when using less secure methods

### Security Warnings

The CLI will warn you when:
- Using environment variables (less secure than keyring)
- API key is stored in plain text (legacy config)
- Keyring is unavailable (falling back to encrypted file)

Example warnings:
```
⚠️  Using API key from environment variable. For better security, use 'repobird login'
⚠️  API key stored in plain text. Run 'repobird login' to secure it
```

## Frequently Asked Questions

**Q: Is my API key encrypted?**  
A: Yes, when stored via `repobird login`, your key is either stored in the system keyring or encrypted with AES-256-GCM.

**Q: Can I use the same API key on multiple machines?**  
A: Yes, but for better security, consider using different keys for different machines and revoking access when needed.

**Q: What happens if I forget my API key?**  
A: You can generate a new one from the RepoBird web interface at https://repobird.ai. The old key should be revoked.

**Q: Is the environment variable method secure?**  
A: It's suitable for CI/CD where keys are injected securely. For development machines, use the keyring storage instead.

**Q: How do I remove my API key completely?**  
A: Run `repobird logout` to remove it from all storage locations.