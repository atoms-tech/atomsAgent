"""Main Gradio application for atoms-agent dev UI."""

from __future__ import annotations

from typing import TYPE_CHECKING

try:
    import gradio as gr
except ImportError as e:
    raise ImportError(
        "Gradio is required for the dev UI. "
        "Install dev dependencies with: "
        "uv pip install -e '.[dev]' or pip install 'atoms-agent[dev]'"
    ) from e

if TYPE_CHECKING:
    from atomsAgent.services.chat_client import ChatClient
else:
    from atomsAgent.services.chat_client import ChatClient

from atomsAgent.ui.chat_tab import create_chat_tab
from atomsAgent.ui.mcp_tab import create_mcp_tab
from atomsAgent.ui.settings_tab import create_settings_tab


def create_app(client: ChatClient | None = None) -> gr.Blocks:
    """Create the main Gradio application.

    Args:
        client: Optional ChatClient instance (creates new one if not provided)

    Returns:
        Gradio Blocks application
    """
    if client is None:
        client = ChatClient()

    with gr.Blocks(
        title="Atoms Agent Dev UI",
        theme=gr.themes.Soft(
            primary_hue="blue",
            secondary_hue="slate",
        ),
    ) as app:
        gr.Markdown(
            """
            # 🤖 Atoms Agent Dev UI
            
            Standalone development interface for testing chat completions, models, and MCP servers.
            """
        )

        with gr.Tabs():
            with gr.Tab("💬 Chat", id="chat"):
                create_chat_tab(client)

            with gr.Tab("⚙️ Settings", id="settings"):
                create_settings_tab(client)

            with gr.Tab("🔌 MCP Servers", id="mcp"):
                create_mcp_tab()

        gr.Markdown(
            """
            ---
            **Atoms Agent** | [Documentation](https://github.com/atoms) | Built with [Gradio](https://gradio.app)
            """
        )

    return app
