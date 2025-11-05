"""Test commands for chat completions and streaming."""

from __future__ import annotations

import time

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from atomsAgent.services.chat_client import ChatClient

test_app = typer.Typer(help="Test chat functionality")
console = Console()


@test_app.command("completion")
def completion_command(
    prompt: str = typer.Option(..., "--prompt", "-p", help="Test prompt"),
    model: str = typer.Option(
        "claude-4-5-haiku-20251001",
        "--model",
        "-m",
        help="Model to test",
    ),
    temperature: float = typer.Option(0.7, "--temperature", "-t", help="Temperature"),
) -> None:
    """Test non-streaming chat completion.

    Example:
        atoms-agent test completion --prompt "Hello, world!"
    """
    console.print(Panel.fit(
        f"[bold]Testing Non-Streaming Completion[/bold]\n"
        f"Model: [cyan]{model}[/cyan]\n"
        f"Prompt: [yellow]{prompt}[/yellow]",
        border_style="blue",
    ))

    client = ChatClient()

    try:
        start_time = time.time()

        with console.status("[bold green]Sending request..."):
            result = client.chat(
                [{"role": "user", "content": prompt}],
                model,
                temperature=temperature,
            )

        elapsed = time.time() - start_time

        # Display response
        console.print("\n[bold green]Response:[/bold green]")
        console.print(result["choices"][0]["message"]["content"])

        # Display stats
        usage = result.get("usage", {})
        table = Table(title="Statistics", show_header=False)
        table.add_column("Metric", style="cyan")
        table.add_column("Value", style="green")

        table.add_row("Prompt Tokens", str(usage.get("prompt_tokens", "N/A")))
        table.add_row("Completion Tokens", str(usage.get("completion_tokens", "N/A")))
        table.add_row("Total Tokens", str(usage.get("total_tokens", "N/A")))
        table.add_row("Latency", f"{elapsed:.2f}s")
        table.add_row("Finish Reason", result["choices"][0].get("finish_reason", "N/A"))

        console.print(table)

    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        raise typer.Exit(1) from e
    finally:
        client.close()


@test_app.command("streaming")
def streaming_command(
    prompt: str = typer.Option(..., "--prompt", "-p", help="Test prompt"),
    model: str = typer.Option(
        "claude-4-5-haiku-20251001",
        "--model",
        "-m",
        help="Model to test",
    ),
    temperature: float = typer.Option(0.7, "--temperature", "-t", help="Temperature"),
) -> None:
    """Test streaming chat completion.

    Example:
        atoms-agent test streaming --prompt "Count from 1 to 10"
    """
    console.print(Panel.fit(
        f"[bold]Testing Streaming Completion[/bold]\n"
        f"Model: [cyan]{model}[/cyan]\n"
        f"Prompt: [yellow]{prompt}[/yellow]",
        border_style="blue",
    ))

    client = ChatClient()

    try:
        console.print("\n[bold green]Streaming response:[/bold green] ", end="")

        start_time = time.time()
        chunk_count = 0
        total_chars = 0

        for chunk in client.stream_chat(
            [{"role": "user", "content": prompt}],
            model,
            temperature=temperature,
        ):
            console.print(chunk, end="")
            chunk_count += 1
            total_chars += len(chunk)

        elapsed = time.time() - start_time
        console.print()

        # Display stats
        table = Table(title="Streaming Statistics", show_header=False)
        table.add_column("Metric", style="cyan")
        table.add_column("Value", style="green")

        table.add_row("Total Chunks", str(chunk_count))
        table.add_row("Total Characters", str(total_chars))
        table.add_row("Latency", f"{elapsed:.2f}s")
        if elapsed > 0:
            table.add_row("Chars/Second", f"{total_chars / elapsed:.1f}")

        console.print(table)

    except Exception as e:
        console.print(f"\n[red]Error: {e}[/red]")
        raise typer.Exit(1) from e
    finally:
        client.close()


@test_app.command("models")
def models_command() -> None:
    """Test model listing endpoint.

    Example:
        atoms-agent test models
    """
    console.print("[bold]Testing Model Listing[/bold]\n")

    client = ChatClient()

    try:
        with console.status("[bold green]Fetching models..."):
            models = client.list_models()

        if not models:
            console.print("[yellow]No models found[/yellow]")
            return

        table = Table(title=f"Available Models ({len(models)})")
        table.add_column("Model ID", style="cyan")
        table.add_column("Provider", style="green")
        table.add_column("Context Length", style="yellow")

        for model in models:
            table.add_row(
                model.id,
                model.provider or "N/A",
                str(model.context_length) if model.context_length else "N/A",
            )

        console.print(table)

    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        raise typer.Exit(1) from e
    finally:
        client.close()
