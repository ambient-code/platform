"""Workflow management endpoint (SDK-provided).

Re-exports the router from the top-level endpoints package.
When the SDK is extracted to a standalone package, this will be self-contained.
"""

import importlib
_mod = importlib.import_module("endpoints.workflow")
router = _mod.router

__all__ = ["router"]
