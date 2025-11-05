"""Interactive chat command for atoms-agent CLI."""

from __future__ import annotations

import typer
from rich.console import Console
from rich.markdown import Markdown
from rich.panel import Panel

from atomsAgent.services.chat_client import ChatClient

chat_app = typer.Typer(help="Interactive chat commands")
console = Console()


@chat_app.command("interactive")
def chat_interactive(
    model: str = typer.Option(
        "claude-4-5-haiku-20251001",
        "--model",
        "-m",
        help="Model to use for chat",
    ),
    system_prompt: str = typer.Option(
        "You are a helpful assistant.",
        "--system-prompt",
        "-s",
        help="System prompt for the conversation",
    ),
    temperature: float = typer.Option(
        0.7,
        "--temperature",
        "-t",
        help="Sampling temperature (0-2)",
        min=0.0,
        max=2.0,
    ),
    stream: bool = typer.Option(
        True,
        "--stream/--no-stream",
        help="Enable streaming responses",
    ),
    max_tokens: int | None = typer.Option(
        None,
        "--max-tokens",
        help="Maximum tokens to generate",
    ),
) -> None:
    """Start an interactive chat session.

    Example:
        atoms-agent chat interactive --model claude-4-5-haiku-20251001
    """
    console.print(Panel.fit(
        f"[bold green]🤖 Atoms Agent Chat[/bold green]\n"
        f"Model: [cyan]{model}[/cyan]\n"
        f"Temperature: [cyan]{temperature}[/cyan]\n"
        f"Streaming: [cyan]{'enabled' if stream else 'disabled'}[/cyan]",
        border_style="green",
    ))
    console.print("[dim]Type 'exit', 'quit', or press Ctrl+C to end session[/dim]\n")

    client = ChatClient()
    messages: list[dict[str, str]] = [{"role": "system", "content": system_prompt}]

    try:
        while True:
            # Get user input
            try:
                user_input = console.input("[bold blue]You:[/bold blue] ")
            except (EOFError, KeyboardInterrupt):
                console.print("\n[yellow]Goodbye![/yellow]")
                break

            if not user_input.strip():
                continue

            if user_input.lower() in ["exit", "quit", "q"]:
                console.print("[yellow]Goodbye![/yellow]")
                break

            # Add user message
            messages.append({"role": "user", "content": user_input})

            # Get assistant response
            console.print("[bold green]Assistant:[/bold green] ", end="")

            try:
                if stream:
                    response = ""
                    for chunk in client.stream_chat(
                        messages,
                        model,
                        temperature=temperature,
                        max_tokens=max_tokens,
                    ):
                        console.print(chunk, end="")
                        response += chunk
                    console.print()  # New line after streaming
                else:
                    result = client.chat(
                        messages,
                        model,
                        temperature=temperature,
                        max_tokens=max_tokens,
                    )
                    response = result["choices"][0]["message"]["content"]
                    console.print(Markdown(response))

                # Add assistant response to history
                messages.append({"role": "assistant", "content": response})

            except Exception as e:
                console.print(f"\n[red]Error: {e}[/red]")
                # Remove the failed user message
                messages.pop()

    except KeyboardInterrupt:
        console.print("\n[yellow]Goodbye![/yellow]")
    finally:
        client.close()


@chat_app.command("once")
def chat_once(
    prompt: str = typer.Argument(..., help="Prompt to send"),
    model: str = typer.Option(
        "claude-4-5-haiku-20251001",
        "--model",
        "-m",
        help="Model to use",
    ),
    system_prompt: str | None = typer.Option(
        None,
        "--system-prompt",
        "-s",
        help="System prompt",
    ),
    temperature: float = typer.Option(
        0.7,
        "--temperature",
        "-t",
        help="Sampling temperature (0-2)",
    ),
    stream: bool = typer.Option(
        True,
        "--stream/--no-stream",
        help="Enable streaming",
    ),
) -> None:
    """Send a single chat message and get a response.

    Example:
        atoms-agent chat once "Explain quantum computing"
    """
    client = ChatClient()

    messages: list[dict[str, str]] = []
    if system_prompt:
        messages.append({"role": "system", "content": system_prompt})
    messages.append({"role": "user", "content": prompt})

    try:
        if stream:
            for chunk in client.stream_chat(messages, model, temperature=temperature):
                console.print(chunk, end="")
            console.print()
        else:
            result = client.chat(messages, model, temperature=temperature)
            response = result["choices"][0]["message"]["content"]
            console.print(Markdown(response))
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        raise typer.Exit(1) from e
    finally:
        client.close()
