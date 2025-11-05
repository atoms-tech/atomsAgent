"""MCP server management tab for Gradio UI."""

from __future__ import annotations

try:
    import gradio as gr
except ImportError as e:
    raise ImportError(
        "Gradio is required for the dev UI. "
        "Install dev dependencies with: "
        "uv pip install -e '.[dev]' or pip install 'atoms-agent[dev]'"
    ) from e


def create_mcp_tab() -> None:
    """Create MCP server management tab."""
    gr.Markdown("## MCP Server Management")

    gr.Markdown(
        """
        Manage Model Context Protocol (MCP) servers for extended capabilities.
        
        **Note:** MCP management requires proper authentication and organization context.
        Use the CLI commands for full MCP functionality:
        - `atoms-agent mcp list --org <ORG_ID>`
        - `atoms-agent mcp create <NAME> --endpoint <URL>`
        - `atoms-agent mcp test <CONFIG_ID>`
        """
    )

    with gr.Row():
        with gr.Column(scale=2):
            gr.Markdown("### Configured Servers")

            servers_table = gr.Dataframe(
                headers=["Name", "Endpoint", "Status", "Auth Type"],
                value=[
                    ["Example Server", "https://mcp.example.com", "Not Connected", "none"],
                ],
                interactive=False,
                wrap=True,
            )

            with gr.Row():
                refresh_servers_btn = gr.Button("🔄 Refresh", variant="secondary")
                _ = gr.Button("🧪 Test All", variant="secondary")  # Placeholder for future implementation

        with gr.Column(scale=1):
            gr.Markdown("### Add New Server")

            server_name = gr.Textbox(
                label="Server Name",
                placeholder="My MCP Server",
            )

            server_endpoint = gr.Textbox(
                label="Endpoint URL",
                placeholder="https://mcp.example.com",
            )

            server_auth = gr.Dropdown(
                choices=["none", "bearer", "oauth"],
                label="Auth Type",
                value="none",
            )

            auth_token = gr.Textbox(
                label="Auth Token (if bearer)",
                type="password",
                placeholder="Optional",
            )

            add_server_btn = gr.Button("+ Add Server", variant="primary")

    gr.Markdown("---")
    gr.Markdown("### Test Server Connection")

    with gr.Row():
        test_server_name = gr.Textbox(
            label="Server Name",
            placeholder="Enter server name to test",
        )

        test_btn = gr.Button("🔌 Test Connection", variant="secondary")

    test_result = gr.JSON(
        label="Test Result",
        value={},
    )

    gr.Markdown("---")
    gr.Markdown("### Server Tools & Resources")

    with gr.Row():
        with gr.Column():
            gr.Markdown("#### Available Tools")
            _ = gr.JSON(
                label="Tools",
                value={"message": "Select a server and test connection to view tools"},
            )

        with gr.Column():
            gr.Markdown("#### Available Resources")
            _ = gr.JSON(
                label="Resources",
                value={"message": "Select a server and test connection to view resources"},
            )

    # Event handlers
    def refresh_servers_fn() -> list[list[str]]:
        """Refresh servers list."""
        # This would call the MCP service in a real implementation
        return [
            ["Example Server", "https://mcp.example.com", "Not Connected", "none"],
            ["Local Server", "http://localhost:8080", "Not Connected", "none"],
        ]

    def test_connection_fn(server_name: str) -> dict:
        """Test server connection."""
        if not server_name:
            return {"error": "Please enter a server name"}

        # This would call the MCP service in a real implementation
        return {
            "status": "success",
            "message": f"Connection to '{server_name}' would be tested here",
            "note": "Use CLI command: atoms-agent mcp test <CONFIG_ID>",
        }

    def add_server_fn(name: str, endpoint: str, auth: str, token: str) -> tuple[list[list[str]], str]:
        """Add new server."""
        if not name or not endpoint:
            return refresh_servers_fn(), "❌ Name and endpoint are required"

        # This would call the MCP service in a real implementation
        return refresh_servers_fn(), f"✅ Server '{name}' would be added here. Use CLI: atoms-agent mcp create"

    # Wire up events
    refresh_servers_btn.click(refresh_servers_fn, outputs=[servers_table])

    test_btn.click(
        test_connection_fn,
        inputs=[test_server_name],
        outputs=[test_result],
    )

    add_server_btn.click(
        add_server_fn,
        inputs=[server_name, server_endpoint, server_auth, auth_token],
        outputs=[servers_table, test_result],
    )
