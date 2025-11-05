"""Chat interface tab for Gradio UI."""

from __future__ import annotations

import time
from collections.abc import Generator
from typing import TYPE_CHECKING, Any

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


def create_chat_tab(client: ChatClient) -> None:
    """Create the chat interface tab.

    Args:
        client: ChatClient instance for API calls
    """
    # Get available models
    try:
        models = client.list_models()
        model_choices = [m.id for m in models]
        default_model = model_choices[0] if model_choices else "claude-4-5-haiku-20251001"
    except Exception:
        model_choices = ["claude-4-5-haiku-20251001", "claude-3-5-haiku-20241022"]
        default_model = "claude-4-5-haiku-20251001"

    with gr.Row():
        with gr.Column(scale=3):
            chatbot = gr.Chatbot(
                label="Conversation",
                height=500,
                type="messages",
                show_copy_button=True,
                avatar_images=(None, "🤖"),
            )

            with gr.Row():
                msg_input = gr.Textbox(
                    label="Message",
                    placeholder="Type your message here...",
                    lines=2,
                    scale=4,
                    show_label=False,
                )
                send_btn = gr.Button("Send", variant="primary", scale=1, size="lg")

            with gr.Row():
                clear_btn = gr.Button("🗑️ Clear", size="sm", variant="secondary")
                retry_btn = gr.Button("🔄 Retry", size="sm", variant="secondary")

        with gr.Column(scale=1):
            gr.Markdown("### Model Settings")

            model_dropdown = gr.Dropdown(
                choices=model_choices,
                value=default_model,
                label="Model",
                interactive=True,
            )

            temperature_slider = gr.Slider(
                minimum=0,
                maximum=2,
                value=0.7,
                step=0.1,
                label="Temperature",
                info="Higher = more creative, Lower = more focused",
            )

            max_tokens_slider = gr.Slider(
                minimum=100,
                maximum=4096,
                value=2048,
                step=100,
                label="Max Tokens",
            )

            top_p_slider = gr.Slider(
                minimum=0,
                maximum=1,
                value=1.0,
                step=0.05,
                label="Top P",
                info="Nucleus sampling threshold",
            )

            gr.Markdown("### System Prompt")

            system_prompt_box = gr.Textbox(
                label="System Prompt",
                value="You are a helpful AI assistant.",
                lines=4,
                placeholder="Enter system prompt...",
            )

            gr.Markdown("### Stats")

            stats_display = gr.JSON(
                label="Last Response Stats",
                value={},
            )

    def chat_fn(
        message: str,
        history: list[dict[str, Any]],
        model: str,
        temp: float,
        max_tok: int,
        top_p: float,
        sys_prompt: str,
    ) -> Generator[tuple[list[dict[str, Any]], str, dict[str, Any]], None, None]:
        """Handle chat message with streaming.

        Returns:
            Tuple of (updated_history, empty_input, stats)
        """
        if not message.strip():
            yield history, "", {}
            return

        # Build messages list
        messages: list[dict[str, str]] = [{"role": "system", "content": sys_prompt}]

        # Add history
        for msg in history:
            messages.append({"role": msg["role"], "content": msg["content"]})

        # Add current message
        messages.append({"role": "user", "content": message})

        # Add user message to history
        history.append({"role": "user", "content": message})

        # Stream response
        start_time = time.time()
        response = ""
        chunk_count = 0

        try:
            for chunk in client.stream_chat(
                messages,
                model,
                temperature=temp,
                max_tokens=max_tok,
                top_p=top_p,
            ):
                response += chunk
                chunk_count += 1

                # Update history with streaming response
                if len(history) > 0 and history[-1]["role"] == "assistant":
                    history[-1]["content"] = response
                else:
                    history.append({"role": "assistant", "content": response})

                yield history, "", {}

            elapsed = time.time() - start_time

            # Final stats
            stats = {
                "latency_ms": round(elapsed * 1000, 2),
                "chunks": chunk_count,
                "chars": len(response),
                "chars_per_sec": round(len(response) / elapsed, 1) if elapsed > 0 else 0,
            }

            yield history, "", stats

        except Exception as e:
            error_msg = f"Error: {e!s}"
            history.append({"role": "assistant", "content": error_msg})
            yield history, "", {"error": str(e)}

    def clear_fn() -> tuple[list, str, dict]:
        """Clear chat history."""
        return [], "", {}

    def retry_fn(
        history: list[dict[str, Any]],
        model: str,
        temp: float,
        max_tok: int,
        top_p: float,
        sys_prompt: str,
    ) -> Generator[tuple[list[dict[str, Any]], str, dict[str, Any]], None, None]:
        """Retry last message."""
        if not history or history[-1]["role"] != "assistant":
            yield history, "", {}
            return

        # Remove last assistant message
        history = history[:-1]

        if not history or history[-1]["role"] != "user":
            yield history, "", {}
            return

        # Get last user message
        last_message = history[-1]["content"]

        # Remove it from history
        history = history[:-1]

        # Re-send
        yield from chat_fn(last_message, history, model, temp, max_tok, top_p, sys_prompt)

    # Event handlers
    send_btn.click(
        chat_fn,
        inputs=[
            msg_input,
            chatbot,
            model_dropdown,
            temperature_slider,
            max_tokens_slider,
            top_p_slider,
            system_prompt_box,
        ],
        outputs=[chatbot, msg_input, stats_display],
    )

    msg_input.submit(
        chat_fn,
        inputs=[
            msg_input,
            chatbot,
            model_dropdown,
            temperature_slider,
            max_tokens_slider,
            top_p_slider,
            system_prompt_box,
        ],
        outputs=[chatbot, msg_input, stats_display],
    )

    clear_btn.click(clear_fn, outputs=[chatbot, msg_input, stats_display])

    retry_btn.click(
        retry_fn,
        inputs=[
            chatbot,
            model_dropdown,
            temperature_slider,
            max_tokens_slider,
            top_p_slider,
            system_prompt_box,
        ],
        outputs=[chatbot, msg_input, stats_display],
    )
