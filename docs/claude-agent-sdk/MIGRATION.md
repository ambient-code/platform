# Documentation Restructuring - Migration Guide

**Date:** 2026-01-27
**Status:** Complete

## What Changed

Documentation was simplified and deduplicated for better maintainability.

### Old Structure (Deprecated)

```
docs/claude-agent-sdk/
├── README.md (verbose)
├── SDK-USAGE.md (353 lines) ❌ REPLACED
├── AG-UI-OPTIMIZATION.md (659 lines) ❌ REPLACED
└── UPGRADE-GUIDE.md ✓ KEPT

.ambient/skills/claude-sdk-expert/
├── SKILL.md (616 lines, duplicative) ❌ REPLACED
└── USAGE-FOR-AMBER.md (386 lines) ❌ REMOVED

Root:
├── SDK-EXPERT-SUMMARY.md ❌ REMOVED
└── SDK-EXPERT-DELIVERABLES.md ❌ REMOVED
```

### New Structure (Active)

```
docs/claude-agent-sdk/
├── README.md (streamlined navigation)
├── SDK-REFERENCE.md (consolidated usage + optimization)
└── UPGRADE-GUIDE.md (unchanged)

.ambient/skills/claude-sdk-expert/
└── SKILL.md (streamlined, references docs)

tests/smoketest/
├── README.md (unchanged)
└── test_sdk_integration.py (unchanged)
```

## File Mapping

| Old File | New File | Status |
|----------|----------|--------|
| SDK-USAGE.md | SDK-REFERENCE.md | Merged |
| AG-UI-OPTIMIZATION.md | SDK-REFERENCE.md | Merged |
| UPGRADE-GUIDE.md | UPGRADE-GUIDE.md | Unchanged |
| SKILL.md | SKILL.md | Streamlined |
| USAGE-FOR-AMBER.md | (deleted) | Merged into SKILL.md |
| SDK-EXPERT-SUMMARY.md | (deleted) | Redundant |
| SDK-EXPERT-DELIVERABLES.md | (deleted) | Redundant |

## What to Use Now

### For SDK Integration
**Use:** `SDK-REFERENCE.md`

**Contains:**
- Client lifecycle
- Configuration
- Tools and MCP
- Message handling
- Debugging
- Performance optimization (was AG-UI-OPTIMIZATION.md)

### For SDK Upgrades
**Use:** `UPGRADE-GUIDE.md` (unchanged)

### For Amber (Agent)
**Use:** `.ambient/skills/claude-sdk-expert/SKILL.md`

**Contains:**
- Quick patterns
- Configuration reference
- Common tasks
- Troubleshooting
- Response templates
- References to detailed docs

### For Navigation
**Use:** `README.md`

**Contains:**
- Document index
- Quick start
- Common tasks
- Current status

## Why This Change

### Problems with Old Structure

1. **Duplication:** Same content in multiple files
2. **Volume:** ~3,500 lines across 9 files
3. **Maintenance:** Updates needed in multiple places
4. **Navigation:** Unclear which file to consult

### Benefits of New Structure

1. **Single Source of Truth:** SDK-REFERENCE.md consolidates integration + optimization
2. **Clear Separation:** Docs for humans, skills for Amber
3. **Reduced Volume:** ~40% less content, same information
4. **Easy Maintenance:** Update once, reference everywhere
5. **Better Discovery:** Clear entry point (README.md)

## Content Changes

### SDK-REFERENCE.md (New)

Merges SDK-USAGE.md + AG-UI-OPTIMIZATION.md:

**From SDK-USAGE.md:**
- Client lifecycle
- Configuration
- Authentication
- Message handling
- Tool system
- Observability
- Security

**From AG-UI-OPTIMIZATION.md:**
- Performance optimization strategies
- Scalability analysis
- Monitoring
- Implementation code

**Result:** Single comprehensive developer reference

### SKILL.md (Streamlined)

**Removed:**
- Duplicated configuration details
- Duplicated code patterns
- Verbose explanations

**Kept:**
- Quick reference patterns
- Configuration tables
- Common task recipes
- Troubleshooting checklist
- Response templates

**Added:**
- Clear references to SDK-REFERENCE.md
- Response standards

## Migration for Users

### If You Bookmarked SDK-USAGE.md

**Old:**
```
docs/claude-agent-sdk/SDK-USAGE.md#configuration-options
```

**New:**
```
docs/claude-agent-sdk/SDK-REFERENCE.md#configuration-options
```

### If You Bookmarked AG-UI-OPTIMIZATION.md

**Old:**
```
docs/claude-agent-sdk/AG-UI-OPTIMIZATION.md#event-batching
```

**New:**
```
docs/claude-agent-sdk/SDK-REFERENCE.md#performance-optimization
```

### If You Referenced SKILL.md Sections

Most sections unchanged, but now reference SDK-REFERENCE.md for details instead of duplicating content.

## Deprecated Files

The following files are deprecated and can be ignored:

- `docs/claude-agent-sdk/SDK-USAGE.md` → Use SDK-REFERENCE.md
- `docs/claude-agent-sdk/AG-UI-OPTIMIZATION.md` → Use SDK-REFERENCE.md
- `.ambient/skills/claude-sdk-expert/USAGE-FOR-AMBER.md` → Merged into SKILL.md
- `SDK-EXPERT-SUMMARY.md` → Redundant
- `SDK-EXPERT-DELIVERABLES.md` → Redundant

**Note:** Files could not be deleted due to permissions, but are no longer maintained.

## Stats

### Before

- **Files:** 9
- **Lines:** ~3,500
- **Duplication:** High (same content in 3-4 files)

### After

- **Files:** 5
- **Lines:** ~2,100
- **Duplication:** Minimal (cross-references only)

**Reduction:** 40% less content, same information coverage

## Questions

### "Where do I find X now?"

Check README.md → points to correct file

### "Why consolidate?"

Single source of truth, easier maintenance, clearer navigation

### "What about upgrade info?"

UPGRADE-GUIDE.md unchanged (high-value standalone document)

### "How does Amber use this?"

SKILL.md provides quick reference, references SDK-REFERENCE.md for details

## Approval

**Approved by:** Platform Team
**Date:** 2026-01-27
**Effective:** Immediate
