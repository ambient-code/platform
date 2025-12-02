# Repomix Architecture View Guide

**Purpose:** Guide for using the repomix architecture view for Claude Code context loading.

## Executive Decision: Single View Approach

After comprehensive analysis of 7 different repomix configurations (see `repomix-analysis/repomix-analysis-report.md`), we've adopted a **single-view approach** using only `03-architecture-only.xml`.

**Why one view?**
- **Grade 8.8/10** - Highest quality score of all tested configurations
- **187K tokens** - Optimal for context windows, leaves room for conversation
- **Comprehensive coverage** - 132 critical files across all 7 components
- **Simpler mental model** - No decision fatigue about which view to use
- **Smaller repo** - 1M vs 19M for all 7 views

See the [analysis heatmap](../repomix-analysis/repomix-heatmap.png) for visual comparison.

## Available View

The `repomix-analysis/` directory contains one pre-generated codebase view:

| File | Size | Tokens | Grade | Use For |
|------|------|--------|-------|---------|
| `03-architecture-only.xml` | 737KB | 187K | 8.8/10 | All development tasks, architecture understanding, planning |

**What's included:**
- ✓ CLAUDE.md (project instructions)
- ✓ All component READMEs (11 files)
- ✓ Type definitions (17 files)
- ✓ Design guidelines
- ✓ Route definitions
- ✓ Infrastructure manifests
- ✓ CRD definitions
- ✓ 132 critical files across all components

**What's excluded:**
- Test files (reduces noise)
- Generated code
- Dependencies (node_modules, vendor)
- Build artifacts

## Usage Examples

### General Development

```
"Claude, load the repomix architecture view (repomix-analysis/03-architecture-only.xml)
and help me understand how multi-repo support works in AgenticSessions."
```

### Architecture Understanding

```
"Claude, using the architecture view, explain how the operator watches for
AgenticSession creation and spawns jobs."
```

### Planning New Features

```
"Claude, reference the architecture view and help me plan where to add
support for custom agent configurations in the frontend."
```

### Combining with Context Files

For even better results, combine the architecture view with context files:

```
"Claude, load the architecture view and the backend-development context file,
then help me add a new endpoint for project settings."
```

```
"Claude, load the architecture view and security-standards context file,
then review this PR for authentication issues."
```

## Quick Reference Table

| Task Type | Command Pattern | Context Files |
|-----------|----------------|---------------|
| Backend API work | Load architecture view | backend-development.md |
| Frontend UI work | Load architecture view | frontend-development.md |
| Security review | Load architecture view | security-standards.md |
| Architecture planning | Load architecture view | - |
| Pattern implementation | Load architecture view | patterns/*.md |

## Regenerating the View

The architecture view is a snapshot in time. Regenerate monthly or after major changes:

```bash
# Regenerate the architecture-only view
repomix --output repomix-analysis/03-architecture-only.xml --style xml

# Uses exclusion patterns from .repomixignore
```

**When to regenerate:**
- After major architectural changes
- Monthly (scheduled maintenance)
- Before major refactoring efforts
- When codebase structure changes significantly

**Tip:** Add to monthly maintenance calendar alongside dependency updates.

## Why Not Multiple Views?

The original analysis tested 7 different configurations:

1. **01-full-context.xml** (550K tokens) - Too large, poor token efficiency
2. **02-production-optimized.xml** (1.1M tokens) - Excessive, unusable
3. **03-architecture-only.xml** (187K tokens) - ✅ **Perfect balance**
4. **04-backend-focused.xml** (103K tokens) - Too narrow, grade 6.6
5. **05-frontend-focused.xml** (196K tokens) - Too narrow, grade 6.4
6. **06-ultra-compressed.xml** (2.6M tokens) - Catastrophically large
7. **07-metadata-rich.xml** (216K tokens) - Redundant with #03

**The verdict:** #03 provides the best balance of:
- Token efficiency (fits in context window)
- Architecture visibility (complete picture)
- Code navigation (132 critical files)
- Context completeness (all components)

See `repomix-analysis/repomix-analysis-report.md` for full analysis details and the heatmap visualization.

## Advanced: Generate-on-Demand

For specialized needs, you can generate custom views on-demand:

```bash
# Backend-only view (if you need it)
repomix --include "components/backend/**" --output backend-custom.xml --style xml

# Frontend-only view
repomix --include "components/frontend/**" --output frontend-custom.xml --style xml

# Security-focused view
repomix --include "**/handlers/**,**/middleware/**,CLAUDE.md" --output security-custom.xml --style xml
```

But in practice, the architecture-only view works for 95% of tasks.
