"""Settings panel tab for Gradio UI."""

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


def create_settings_tab(client: ChatClient) -> None:
    """Create settings configuration tab.

    Args:
        client: ChatClient instance for API calls
    """
    gr.Markdown("## Model Management")

    with gr.Row():
        with gr.Column():
            gr.Markdown("### Available Models")

            models_table = gr.Dataframe(
                headers=["Model ID", "Provider", "Context Length"],
                value=[],
                interactive=False,
                wrap=True,
            )

            refresh_models_btn = gr.Button("🔄 Refresh Models", variant="secondary")

        with gr.Column():
            gr.Markdown("### Model Information")

            _ = gr.JSON(
                label="Selected Model Details",
                value={},
            )

    gr.Markdown("---")
    gr.Markdown("## System Prompts")

    with gr.Row():
        with gr.Column():
            _ = gr.Textbox(
                label="Prompt Name",
                placeholder="e.g., 'Code Assistant', 'Creative Writer'",
            )

            prompt_content = gr.Textbox(
                label="Prompt Content",
                lines=6,
                placeholder="Enter system prompt...",
                value="You are a helpful AI assistant.",
            )

            with gr.Row():
                _ = gr.Button("💾 Save Prompt", variant="primary")  # Placeholder for future implementation
                load_prompt_btn = gr.Button("📂 Load Prompt", variant="secondary")

        with gr.Column():
            gr.Markdown("### Saved Prompts")

            saved_prompts_list = gr.Dropdown(
                label="Select Prompt",
                choices=[
                    "Default Assistant",
                    "Code Expert",
                    "Creative Writer",
                    "Technical Explainer",
                ],
                value="Default Assistant",
            )

            prompt_preview = gr.Textbox(
                label="Preview",
                lines=6,
                interactive=False,
                value="You are a helpful AI assistant.",
            )

    gr.Markdown("---")
    gr.Markdown("## API Configuration")

    with gr.Row():
        api_url = gr.Textbox(
            label="API Base URL",
            value=client.base_url,
            placeholder="http://localhost:3284",
        )

        api_key = gr.Textbox(
            label="API Key",
            value="***" if client.api_key else "",
            type="password",
            placeholder="Optional API key",
        )

    test_connection_btn = gr.Button("🔌 Test Connection", variant="secondary")
    connection_status = gr.Textbox(label="Status", interactive=False, value="Not tested")

    # Event handlers
    def refresh_models_fn() -> list[list[str]]:
        """Refresh models list."""
        try:
            models = client.list_models()
            return [
                [
                    m.id,
                    m.provider or "N/A",
                    str(m.context_length) if m.context_length else "N/A",
                ]
                for m in models
            ]
        except Exception as e:
            return [["Error", str(e), "N/A"]]

    def test_connection_fn(url: str, key: str) -> str:
        """Test API connection."""
        try:
            test_client = ChatClient(base_url=url, api_key=key if key != "***" else None)
            models = test_client.list_models()
            test_client.close()
            return f"✅ Connected successfully! Found {len(models)} models."
        except Exception as e:
            return f"❌ Connection failed: {e!s}"

    def load_prompt_fn(prompt_name: str) -> str:
        """Load a saved prompt."""
        prompts = {
            "Default Assistant": "You are a helpful AI assistant.",
            "Code Expert": "You are an expert programmer. Provide clear, concise code examples with explanations.",
            "Creative Writer": "You are a creative writer. Write engaging, imaginative content.",
            "Technical Explainer": "You are a technical expert. Explain complex topics in simple terms.",
        }
        return prompts.get(prompt_name, "You are a helpful AI assistant.")

    # Wire up events
    refresh_models_btn.click(refresh_models_fn, outputs=[models_table])

    test_connection_btn.click(
        test_connection_fn,
        inputs=[api_url, api_key],
        outputs=[connection_status],
    )

    saved_prompts_list.change(
        load_prompt_fn,
        inputs=[saved_prompts_list],
        outputs=[prompt_preview],
    )

    load_prompt_btn.click(
        load_prompt_fn,
        inputs=[saved_prompts_list],
        outputs=[prompt_content],
    )

    # Load models on tab load
    models_table.value = refresh_models_fn()
