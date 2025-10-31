"""Utilities for generating Pydantic models from the Supabase schema."""

from __future__ import annotations

import pathlib
import subprocess
import sys


def main(argv: list[str] | None = None) -> int:
    """
    Generate Pydantic models from Supabase schema.

    This script requires a running local Supabase/PostgreSQL database to work.
    The database should be accessible at localhost:54322 (default Supabase port).

    Prerequisites:
    1. supabase-pydantic package installed
    2. Running local database with the schema from database/schema.sql
    3. Proper database connection configuration

    Usage:
        python -m atomsAgent.codegen.supabase
    """
    try:
        # Check if supabase-pydantic CLI is available
        result = subprocess.run(
            [sys.executable, "-m", "supabase_pydantic", "--help"],
            capture_output=True,
            text=True,
            check=False,
        )
        if result.returncode != 0:
            print(
                "supabase-pydantic CLI is not available. Install with `pip install supabase-pydantic`.",
                file=sys.stderr,
            )
            return 1
    except Exception as e:
        print(f"Error checking supabase-pydantic: {e}", file=sys.stderr)
        return 1

    # Paths for schema and output
    schema_path = pathlib.Path(__file__).resolve().parents[4] / "database" / "schema.sql"
    output_dir = pathlib.Path(__file__).resolve().parents[1] / "db"

    if not schema_path.exists():
        print(f"Schema file not found at {schema_path}", file=sys.stderr)
        return 1

    # Create output directory if it doesn't exist
    output_dir.mkdir(parents=True, exist_ok=True)

    print("=== Supabase Pydantic Model Generator ===")
    print(f"Schema file: {schema_path}")
    print(f"Output directory: {output_dir}")
    print()

    print("This utility requires a running local Supabase/PostgreSQL database.")
    print("Default connection: localhost:54322")
    print()
    print("To run this generator:")
    print("1. Start your local Supabase instance")
    print("2. Ensure the database schema is loaded")
    print("3. Update database connection settings if needed")
    print("4. Run this script again")
    print()
    print("Alternative: Use database URL with --db-url option:")
    print(
        f"  {sys.executable} -m supabase_pydantic gen --db-url postgresql://user:pass@host:port/db --type pydantic --dir {output_dir}"
    )
    print()

    # Try to generate models if database is available
    cmd = [
        sys.executable,
        "-m",
        "supabase_pydantic",
        "gen",
        "--local",  # Use local database
        "--type",
        "pydantic",  # Generate Pydantic models
        "--dir",
        str(output_dir),  # Output directory
    ]

    print("Attempting to connect to local database...")
    print(f"Command: {' '.join(cmd)}")
    print()

    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=False)

        if result.returncode == 0:
            print(f"✅ Successfully generated Supabase Pydantic models in {output_dir}")
            return 0
        else:
            print("❌ Database connection failed or generation error occurred:")
            print(result.stderr)
            print()
            print("Common solutions:")
            print("1. Start your local Supabase instance:")
            print("   supabase start")
            print()
            print("2. Check if database is running on correct port:")
            print("   lsof -i :54322")
            print()
            print("3. Use manual database URL if needed:")
            print(
                f"   {sys.executable} -m supabase_pydantic gen --db-url 'postgresql://...' --type pydantic --dir {output_dir}"
            )
            print()
            return 1
    except Exception as e:
        print(f"Error running supabase-pydantic CLI: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
