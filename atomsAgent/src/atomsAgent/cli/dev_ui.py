"""Launch Gradio dev UI for atoms-agent."""

from __future__ import annotations

import typer
from rich.console import Console
from rich.panel import Panel

dev_ui_app = typer.Typer(help="Development UI commands")
console = Console()


@dev_ui_app.command("launch")
def launch_dev_ui(
    port: int = typer.Option(7860, "--port", "-p", help="Port to run on"),
    host: str = typer.Option("127.0.0.1", "--host", help="Host to bind to"),
    share: bool = typer.Option(False, "--share", help="Create public shareable link"),
    debug: bool = typer.Option(False, "--debug", help="Enable debug mode"),
) -> None:
    """Launch Gradio development UI.

    Example:
        atoms-agent dev-ui launch --port 7860 --share
    """
    try:
        import gradio as gr  # noqa: F401
    except ImportError:
        console.print(
            "[red]Error: Gradio not installed.[/red]\n"
            "[yellow]Install dev dependencies:[/yellow]\n"
            "  uv pip install -e '.[dev]'\n"
            "  or\n"
            "  pip install 'atoms-agent[dev]'"
        )
        raise typer.Exit(1) from None

    console.print(Panel.fit(
        "[bold green]🚀 Starting Atoms Agent Dev UI[/bold green]\n\n"
        f"Host: [cyan]{host}[/cyan]\n"
        f"Port: [cyan]{port}[/cyan]\n"
        f"Share: [cyan]{'Yes' if share else 'No'}[/cyan]",
        border_style="green",
    ))

    try:
        from atomsAgent.services.chat_client import ChatClient
        from atomsAgent.ui.app import create_app

        # Create client and app
        client = ChatClient()
        app = create_app(client)

        # Launch
        console.print(f"\n[bold blue]🌐 UI running at http://{host}:{port}[/bold blue]")

        if share:
            console.print("[bold yellow]📡 Creating shareable link...[/bold yellow]")

        console.print("[dim]Press Ctrl+C to stop[/dim]\n")

        app.launch(
            server_name=host,
            server_port=port,
            share=share,
            show_error=debug,
            quiet=not debug,
        )

    except KeyboardInterrupt:
        console.print("\n[yellow]Shutting down...[/yellow]")
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        if debug:
            raise
        raise typer.Exit(1) from e
