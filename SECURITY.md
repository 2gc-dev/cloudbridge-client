# Security Guidelines for CloudBridge Client

## 🔒 Security Considerations

This repository is **public** and contains sensitive information that should not be exposed. Please follow these security guidelines.

## 🚨 Sensitive Information

### What NOT to commit:
- **Real JWT tokens** - Use placeholder tokens like `"your-jwt-token-here"`
- **Real server addresses** - Use `example.com` instead of real domains
- **Real IP addresses** - Use `192.168.1.100` or `10.0.0.1` for examples
- **Real ports** - Use standard ports or clearly marked example ports
- **Real secrets** - Use `"your-secret-here"` placeholders
- **Real infrastructure details** - Use generic descriptions

### What IS safe to commit:
- **Code structure and architecture**
- **Configuration examples** (with placeholders)
- **Documentation** (without real endpoints)
- **Test files** (with mock data)
- **Build scripts** (without secrets)

## 📁 File Organization

### Configuration Files:
- **`config.yaml.example`** - Template with placeholders ✅
- **`config.yaml`** - Real config (in .gitignore) ❌
- **`config/prometheus.yml.example`** - Template ✅
- **`config/prometheus.yml`** - Real config (in .gitignore) ❌

### Documentation:
- **`docs/`** - General documentation ✅
- **`docs/analysis/`** - Analysis reports (may contain sensitive data) ❌
- **`docs/security/`** - Security reports (may contain sensitive data) ❌

### Test Files:
- **`testdata/config-test.yaml.example`** - Test template ✅
- **`testdata/config-test.yaml`** - Real test config (in .gitignore) ❌
- **`test/executables/`** - Compiled binaries (in .gitignore) ❌

## 🔧 Setup Instructions

### 1. Copy example configurations:
```bash
cp config.yaml.example config.yaml
cp config/prometheus.yml.example config/prometheus.yml
cp testdata/config-test.yaml.example testdata/config-test.yaml
```

### 2. Update with your values:
```yaml
# config.yaml
server:
  host: "your-relay-server.com"  # Replace with real server
  jwt_token: "your-real-jwt-token"  # Replace with real token

# config/prometheus.yml
scrape_configs:
  - job_name: 'cloudbridge-client'
    static_configs:
      - targets: ['your-server:8081']  # Replace with real target
```

### 3. Verify sensitive files are ignored:
```bash
git status
# Should NOT show: config.yaml, prometheus.yml, etc.
```

## 🛡️ Security Best Practices

### For Developers:
1. **Never commit real tokens or secrets**
2. **Use environment variables** for sensitive data
3. **Use .env files** (already in .gitignore)
4. **Review all commits** before pushing
5. **Use pre-commit hooks** to check for secrets

### For Contributors:
1. **Fork the repository** to contribute
2. **Use example configurations** in your PRs
3. **Don't include real infrastructure details**
4. **Test with mock data** only

### For Deployment:
1. **Use secrets management** (Kubernetes secrets, Docker secrets)
2. **Use environment variables** for configuration
3. **Use secure key storage** for private keys
4. **Use TLS certificates** for all communications

## 🔍 Security Scanning

### Pre-commit checks:
```bash
# Check for secrets in staged files
git diff --cached | grep -i "eyJ\|token\|secret\|password"

# Run security scanner
gosec ./...

# Check for hardcoded IPs/domains
grep -r "2gc\.\|edge\.2gc\|auth\.2gc" . --exclude-dir=.git
```

### Automated scanning:
- **GitHub Security** - Automatic secret scanning
- **CodeQL** - Code analysis for vulnerabilities
- **Dependabot** - Dependency vulnerability alerts

## 📋 Checklist Before Committing

- [ ] No real JWT tokens in code
- [ ] No real server addresses in examples
- [ ] No real IP addresses in documentation
- [ ] No real secrets in configuration
- [ ] All sensitive files in .gitignore
- [ ] Example files use placeholders
- [ ] Documentation uses generic examples
- [ ] Test files use mock data

## 🚨 If You Accidentally Commit Sensitive Data

### Immediate Actions:
1. **Remove the commit** from history:
   ```bash
   git reset --hard HEAD~1
   git push --force-with-lease
   ```

2. **Rotate all exposed secrets**:
   - JWT tokens
   - API keys
   - Private keys
   - Passwords

3. **Notify security team** if applicable

4. **Update documentation** to prevent future issues

## 📞 Security Contacts

- **Security Issues**: Create a private security issue
- **Sensitive Data Exposure**: Contact maintainers privately
- **Vulnerability Reports**: Use responsible disclosure

## 🔗 Related Files

- **`.gitignore`** - Lists all ignored sensitive files
- **`config.yaml.example`** - Safe configuration template
- **`SECURITY.md`** - This file
- **`docs/security/`** - Security documentation (may contain sensitive data)

---

*Last updated: 2025-08-16*
