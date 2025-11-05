"""Gradio-based dev UI for atoms-agent."""

from __future__ import annotations

try:
    from atomsAgent.ui.app import create_app

    __all__ = ["create_app"]
except ImportError:
    # Gradio not installed - this is fine, UI features just won't be available
    __all__: list[str] = []
