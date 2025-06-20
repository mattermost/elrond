# Dependency Management Strategy

This repository uses a **hybrid approach** combining automated security updates with manual version management.

## 🔒 **Security Updates** (Automated)
- **Tool**: Dependabot 
- **Frequency**: Daily
- **Scope**: Security vulnerabilities only
- **Action Required**: Review and approve security PRs promptly

## 📦 **Version Updates** (Manual)
- **Tool**: Makefile commands
- **Frequency**: As needed or monthly
- **Scope**: All dependencies (major, minor, patch)

### Available Commands:
```bash
# Check which dependencies are outdated
make check-modules

# Update all dependencies to latest versions
make update-modules
```

## 🤝 **How They Work Together**

### 1. **Security-First Priority**
- Dependabot handles urgent security fixes immediately
- Manual updates handle planned version bumps

### 2. **No Conflicts**  
- Security updates are typically small, focused changes
- Manual updates happen when convenient for the team

### 3. **Best Practices**
- **Weekly**: Review any dependabot security PRs
- **Monthly**: Automated reminder issues will be created (see below)
- **As Needed**: Run `make update-modules` for bulk updates
- **Before Releases**: Ensure dependencies are current

## 🤖 **Automated Monthly Updates** 

A GitHub Actions workflow runs on the 1st of every month to:

- ✅ Check for outdated dependencies using your existing `make check-modules`
- ✅ **Automatically run `make update-modules`** if updates are available
- ✅ **Create a Pull Request** with the updated dependencies
- ✅ Include detailed analysis of changes and testing checklist
- ✅ Request review from `@mattermost/cloud-sre` team

### What the Automation Does
1. **Dependency Analysis**: Checks both Go modules and Makefile tool versions
2. **Automatic Updates**: Runs `make update-modules` when needed
3. **Smart PR Creation**: Only creates PRs when actual changes are made
4. **Detailed Reporting**: Includes before/after comparisons and full diff
5. **Review Assignment**: Automatically assigns SRE team for review

### Manual Trigger
You can also run this check anytime by going to:
**Actions** → **Monthly Dependency Check** → **Run workflow**

## 🔧 **Optional: Enable Version Updates Too**

If you want dependabot to also handle version updates, uncomment the section in `.github/dependabot.yml`. This creates a **dual approach**:

- **Dependabot**: Handles minor/patch updates automatically
- **Manual**: Handle major version updates that need careful review

## 🚀 **Benefits of This Approach**

✅ **Security**: Critical fixes applied quickly via dependabot  
✅ **Automation**: Monthly updates happen automatically with PR creation  
✅ **Control**: Team reviews all updates before they're merged  
✅ **Efficiency**: No more forgetting monthly dependency updates  
✅ **Transparency**: Full before/after analysis in every PR  
✅ **Integration**: Uses existing Makefile commands for consistency 