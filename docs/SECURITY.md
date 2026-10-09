# Security Policy

## Secure API Key Storage

RepoBird CLI supports environment credentials, native system keyrings, and a permission-protected file fallback. The fallback is not secret-backed encryption; its security boundary is filesystem access.

### Storage Methods

#### 1. System Keyring (Most Secure)
- **macOS**: Keychain
- **Windows**: Credential Manager  
- **Linux**: GNOME Keyring, KWallet, or Secret Service (when desktop environment is available)

This is the default primary storage method when available. The CLI also writes an XDG fallback so authentication survives keyring availability changes. **That fallback has the limitations below even after a successful keyring save**; keyring use does not eliminate the file copy.

#### 2. Permission-Protected File Fallback
- Retains the existing AES-256-GCM format with random nonces for compatibility.
- Its SHA-256 key derives from hostname, username, home directory, machine ID (when available), and a static application salt. **These attributes are not secret.**
- File permissions are set to **0600 on every save**, including overwrites of existing files. New configuration directories use 0700.
- Location: `$XDG_CONFIG_HOME/repobird/.api_key.enc`, or `~/.config/repobird/.api_key.enc` by default.

Anyone who can read the fallback and obtain these machine attributes can reconstruct its key and recover the API key. Treat the file and its backups as credentials. This is obfuscation, not encryption that protects against a file-reading attacker, the same user, or root. A stronger KDF cannot create secrecy from public inputs; a random key stored alongside the ciphertext would not protect against an attacker who reads both.

Unix owner-only permissions protect against other unprivileged users only while filesystem permissions are enforced. On Windows, 0600 is not a replacement for access-control lists; restrict the configuration directory's ACL to your account. Do not share configuration directories or include credential files in broadly readable backups.

Existing fallback files remain readable without re-login and keep the same format on the next write. Reads do not tighten existing permissions; the next save restores 0600. Legacy encrypted keys in `~/.repobird/.api_key.enc` are still read and migrated.

Automatically used on headless servers, containers, or when keyring is unavailable. Use externally managed secret injection via `REPOBIRD_API_KEY` when local file storage is unsuitable.

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
   This validates your key and saves it using the available storage methods, including the fallback copy described above.

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

If you have an API key stored in plain text (legacy versions), it will be automatically migrated to the available storage methods on first use. On keyless systems, this moves the key to the permission-protected fallback, not secret-backed encryption. You can manually trigger migration:

```bash
repobird login
```

## Compliance

The storage design provides:
- Platform-specific security integration (Keychain, Credential Manager, Keyring)
- Owner-only Unix permissions for fallback files on each write
- Backward-compatible reads and migration of existing credentials
- Explicit warnings about the keyless fallback's limits

### Security Warnings

The CLI will warn you when:
- Using environment variables (less secure than keyring)
- API key is stored in plain text (legacy config)
- The keyless file fallback is used (its machine-derived key is not secret)

Example warnings:
```
⚠️  Using API key from environment variable. For better security, use 'repobird login'
⚠️  API key stored in plain text. Run 'repobird login' to secure it
```

## Frequently Asked Questions

**Q: Is my API key encrypted?**  
A: The fallback uses AES-256-GCM, but its key is reproducible from non-secret machine attributes. File permissions, not cryptographic secrecy, protect that copy. The system keyring uses native platform protection; the CLI also maintains a fallback file copy.

**Q: Can I use the same API key on multiple machines?**  
A: Yes, but for better security, consider using different keys for different machines and revoking access when needed.

**Q: What happens if I forget my API key?**  
A: You can generate a new one from the RepoBird web interface at https://repobird.ai. The old key should be revoked.

**Q: Is the environment variable method secure?**  
A: It's suitable for CI/CD where keys are injected securely. For development machines, use the keyring storage instead.

**Q: How do I remove my API key completely?**  
A: Run `repobird logout` to remove it from all storage locations.
