# Developer Tools

This directory contains documentation for optional developer productivity tools that live in this repository but are **not part of the core Ambient Code Platform**.

## 🤖 Amber - GitHub Automation Bot

**Amber** is a GitHub Actions-based automation tool that handles issues and creates pull requests automatically.

### What is Amber?

- ⚠️ **Not part of core platform** - Platform runs without Amber
- 🎯 **Repository-specific tool** - Automates development tasks in this repo
- 🔧 **GitHub Actions based** - Triggered by issue labels
- 📍 **Optional setup** - Requires GitHub secrets configuration

### What Amber Does

**Automated Workflows:**
- 🤖 **Auto-Fix** - Linting, formatting, trivial fixes
- 🔧 **Refactoring** - Break large files, extract patterns
- 🧪 **Test Coverage** - Add missing tests

### Quick Links

- **[5-Minute Quickstart](../amber-quickstart.md)** - Get Amber running
- **[Full Automation Guide](../amber-automation.md)** - Complete documentation
- **[Setup Instructions](../../AMBER_SETUP.md)** - Initial configuration

### Usage

1. Create GitHub issue using Amber template
2. Add appropriate label (`amber:auto-fix`, `amber:refactor`, `amber:test-coverage`)
3. Amber automatically creates PR with changes
4. Review and merge PR

**Create Issues:**
- [🤖 Auto-Fix Issue](../../issues/new?template=amber-auto-fix.yml)
- [🔧 Refactoring Issue](../../issues/new?template=amber-refactor.yml)
- [🧪 Test Coverage Issue](../../issues/new?template=amber-test-coverage.yml)

## 🔮 Future Tools

As the project grows, this directory will contain additional developer tools:

- Code generation utilities
- Migration scripts
- Development helpers
- Analysis tools

## 🤝 Contributing Tools

Have an idea for a developer productivity tool?

1. Open a GitHub Discussion describing the tool
2. Get feedback from maintainers
3. Implement and document in this directory
4. Submit PR

---

**Remember:** Tools in this directory are development aids for this repository. They are NOT deployed as part of the Ambient Code Platform runtime.
