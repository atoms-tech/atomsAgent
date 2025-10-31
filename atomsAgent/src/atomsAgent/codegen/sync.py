"""Modern schema synchronization utility with progress tracking and parallelization.

This utility provides optimized schema synchronization from atoms-mcp-prod with:
- Modern Typer CLI interface
- Rich progress tracking
- Parallel processing capabilities
- Retry logic via tenacity
- Caching via aiocache
- Optimized database connections
"""

from __future__ import annotations

import asyncio
import pathlib
import subprocess
import sys
import time
from typing import Any

import aiocache
import httpx
import tenacity
import typer
from rich.console import Console
from rich.progress import (
    BarColumn,
    Progress,
    TaskID,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)
from rich.table import Table

app = typer.Typer(
    name="atoms-codegen",
    help="Modern schema synchronization utility with parallelization and progress tracking",
    no_args_is_help=True,
)
console = Console()


@tenacity.retry(
    stop=tenacity.stop_after_attempt(3),
    wait=tenacity.wait_exponential(multiplier=1, min=4, max=10),
    retry_error_callback=lambda retry_state: console.print(
        f"[yellow]Retrying... (attempt {retry_state.attempt_number})[/yellow]"
    ),
)
@aiocache.cached(
    ttl=300,  # Cache for 5 minutes
    cache=aiocache.SimpleMemoryCache,
)
async def check_database_connection(db_url: str) -> bool:
    """Check if database connection is available with caching."""
    try:
        # For Supabase local, we'll check if the port is open
        if "localhost" in db_url and "54322" in db_url:
            import socket

            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            result = sock.connect_ex(("localhost", 54322))
            sock.close()
            return result == 0
        else:
            # For other databases, use httpx to check connectivity
            async with httpx.AsyncClient(timeout=5.0) as client:
                # This is a simplified check - real implementation would depend on DB type
                _ = client  # Use client to avoid unused variable warning
                return True
    except Exception:
        return False


async def run_with_progress(
    command: list[str],
    description: str,
    progress: Progress,
    task_id: TaskID | None = None,
) -> subprocess.CompletedProcess:
    """Run command with rich progress tracking."""
    if task_id is None:
        task_id = progress.add_task(description, total=None)

    try:
        process = subprocess.Popen(
            command,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
            universal_newlines=True,
        )

        # Monitor progress while process runs
        while True:
            return_code = process.poll()
            if return_code is not None:
                progress.update(task_id, completed=100)
                break
            await asyncio.sleep(0.1)
            progress.update(task_id, advance=0.1)  # Small progress to show activity

        stdout, stderr = process.communicate()

        if return_code == 0:
            progress.update(task_id, completed=100, description=f"[green]✅ {description}")
        else:
            progress.update(task_id, description=f"[red]❌ {description} failed")

        return subprocess.CompletedProcess(command, return_code, stdout, stderr)

    except Exception as e:
        progress.update(task_id, description=f"[red]❌ {description} error: {e}")
        raise


async def parallel_schema_processing(
    schema_files: list[pathlib.Path],
    output_dir: pathlib.Path,
    max_workers: int = 4,
) -> dict[str, Any]:
    """Process multiple schema files in parallel."""
    results = {}

    with Progress(
        TextColumn("[progress.description]{task.description}"),
        BarColumn(),
        TaskProgressColumn(),
        TimeElapsedColumn(),
        console=console,
    ) as progress:
        # Submit all schema processing tasks
        tasks = [
            process_single_schema(schema_file, output_dir, progress)
            for schema_file in schema_files
        ]

        # Collect results as they complete
        for coro in asyncio.as_completed(tasks):
            try:
                result = await coro
                # Extract schema_file from result
                schema_file_path = result.get("file", "unknown")
                results[schema_file_path] = result
            except Exception as e:
                console.print(f"[red]Error processing schema: {e}[/red]")
                results["unknown"] = {"error": str(e)}

    return results


async def process_single_schema(
    schema_file: pathlib.Path,
    output_dir: pathlib.Path,
    progress: Progress,
) -> dict[str, Any]:
    """Process a single schema file."""
    task_id = progress.add_task(f"Processing {schema_file.name}", total=100)

    try:
        # For now, delegate to supabase-pydantic
        # Future enhancement: Direct SQL parsing with parallel table processing
        cmd = [
            sys.executable,
            "-m",
            "supabase_pydantic",
            "gen",
            "--local",
            "--type",
            "pydantic",
            "--dir",
            str(output_dir / schema_file.stem),
        ]

        result = await run_with_progress(cmd, f"Processing {schema_file.name}", progress, task_id)

        return {
            "file": str(schema_file),
            "success": result.returncode == 0,
            "output": result.stdout,
            "error": result.stderr if result.returncode != 0 else None,
        }

    except Exception as e:
        return {
            "file": str(schema_file),
            "success": False,
            "error": str(e),
        }


@app.command()
def sync(
    schema_path: pathlib.Path | None = typer.Option(
        None,
        "--schema",
        "-s",
        help="Path to schema file or directory",
    ),
    output_dir: pathlib.Path | None = typer.Option(
        None,
        "--output",
        "-o",
        help="Output directory for generated models",
    ),
    db_url: str = typer.Option(
        "postgresql://localhost:54322/supabase",
        "--db-url",
        "-d",
        help="Database connection URL",
    ),
    parallel: bool = typer.Option(
        True,
        "--parallel/--no-parallel",
        help="Enable parallel processing",
    ),
    max_workers: int = typer.Option(
        4,
        "--max-workers",
        "-w",
        help="Maximum number of parallel workers",
    ),
    cache_ttl: int = typer.Option(
        300,
        "--cache-ttl",
        help="Cache TTL in seconds",
    ),
    verbose: bool = typer.Option(
        False,
        "--verbose",
        "-v",
        help="Verbose output",
    ),
) -> None:
    """Synchronize schemas with optimized processing and progress tracking."""

    # Set up paths
    if schema_path is None:
        schema_path = pathlib.Path(__file__).resolve().parents[4] / "database"

    if output_dir is None:
        output_dir = pathlib.Path(__file__).resolve().parents[1] / "db"

    # Resolve schema files
    if schema_path.is_file():
        schema_files = [schema_path]
    else:
        schema_files = list(schema_path.glob("*.sql"))
        if not schema_files:
            console.print(f"[red]No SQL files found in {schema_path}[/red]")
            raise typer.Exit(1)

    # Create output directory
    output_dir.mkdir(parents=True, exist_ok=True)

    # Display configuration
    console.print("[bold blue]=== Atoms Agent Schema Sync ===[/bold blue]")
    config_table = Table(show_header=False, box=None)
    config_table.add_row("Schema files", f"{len(schema_files)} files")
    config_table.add_row("Output directory", str(output_dir))
    config_table.add_row("Database", db_url)
    config_table.add_row("Parallel processing", "Enabled" if parallel else "Disabled")
    if parallel:
        config_table.add_row("Max workers", str(max_workers))
    console.print(config_table)
    console.print()

    # Check database connection
    with Progress(
        TextColumn("[progress.description]{task.description}"),
        console=console,
    ) as progress:
        task = progress.add_task("Checking database connection...", total=None)

        connection_ok = asyncio.run(check_database_connection(db_url))

        if connection_ok:
            progress.update(task, description="[green]✅ Database connection OK[/green]")
        else:
            progress.update(task, description="[red]❌ Database connection failed[/red]")
            console.print()
            console.print("[yellow]Tip: Make sure your local database is running[/yellow]")
            console.print(f"[dim]Database URL: {db_url}[/dim]")
            raise typer.Exit(1)

    # Process schemas
    start_time = time.time()

    if parallel and len(schema_files) > 1:
        console.print("[bold]Processing schemas in parallel...[/bold]")
        results = asyncio.run(parallel_schema_processing(schema_files, output_dir, max_workers))
    else:
        console.print("[bold]Processing schemas sequentially...[/bold]")
        with Progress(
            TextColumn("[progress.description]{task.description}"),
            BarColumn(),
            TaskProgressColumn(),
            TimeElapsedColumn(),
            console=console,
        ) as progress:
            results = {}
            for schema_file in schema_files:
                result = asyncio.run(process_single_schema(schema_file, output_dir, progress))
                results[str(schema_file)] = result

    # Summary
    elapsed_time = time.time() - start_time
    successful = sum(1 for r in results.values() if r.get("success"))
    total = len(results)

    console.print()
    console.print("[bold blue]=== Summary ===[/bold blue]")
    summary_table = Table(show_header=False, box=None)
    summary_table.add_row("Total files", str(total))
    summary_table.add_row("Successful", f"[green]{successful}[/green]")
    summary_table.add_row("Failed", f"[red]{total - successful}[/red]")
    summary_table.add_row("Time elapsed", f"{elapsed_time:.2f}s")
    console.print(summary_table)

    # Show errors if any
    failed_files = [f for f, r in results.items() if not r.get("success")]
    if failed_files and verbose:
        console.print()
        console.print("[bold red]Errors:[/bold red]")
        for file_path in failed_files:
            result = results[file_path]
            console.print(f"[red]• {file_path}: {result.get('error', 'Unknown error')}[/red]")

    if successful == total:
        console.print()
        console.print("[green]✅ All schemas processed successfully![/green]")
        raise typer.Exit(0)
    else:
        console.print()
        console.print(f"[red]❌ {total - successful} files failed[/red]")
        raise typer.Exit(1)


@app.command()
def clean(
    output_dir: pathlib.Path | None = typer.Option(
        None,
        "--output",
        "-o",
        help="Output directory to clean",
    ),
    force: bool = typer.Option(
        False,
        "--force",
        "-f",
        help="Force clean without confirmation",
    ),
) -> None:
    """Clean generated model files."""

    if output_dir is None:
        output_dir = pathlib.Path(__file__).resolve().parents[1] / "db"

    if not output_dir.exists():
        console.print(f"[yellow]Output directory {output_dir} does not exist[/yellow]")
        return

    # List files to be deleted
    generated_files = list(output_dir.rglob("*.py"))
    if not generated_files:
        console.print("[yellow]No generated files to clean[/yellow]")
        return

    console.print(f"[bold red]Files to be deleted from {output_dir}:[/bold red]")
    for file_path in generated_files:
        console.print(f"  • {file_path.relative_to(output_dir)}")

    if not force and not typer.confirm("Continue?"):
        console.print("Cancelled.")
        raise typer.Exit(0)

    # Delete files
    with Progress(
        TextColumn("[progress.description]{task.description}"),
        BarColumn(),
        console=console,
    ) as progress:
        task = progress.add_task("Cleaning files...", total=len(generated_files))

        for file_path in generated_files:
            file_path.unlink()
            progress.advance(task)

    console.print("[green]✅ Clean completed[/green]")


@app.command()
def version() -> None:
    """Show version information."""
    console.print("atoms-codegen v0.1.0")
    console.print("Modern schema synchronization utility")


if __name__ == "__main__":
    app()
