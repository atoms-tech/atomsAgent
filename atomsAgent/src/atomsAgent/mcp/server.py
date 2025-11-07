"""
FastMCP Server for atomsAgent

Provides built-in tools for atomsAgent:
- search_requirements: Search requirements in Supabase
- create_requirement: Create new requirements
- analyze_document: Use Claude to analyze documents
- search_codebase: Search code using ripgrep
"""

from __future__ import annotations

import json
import subprocess
from typing import Any

from fastmcp import Context, FastMCP

from atomsAgent.mcp.supabase_client import get_supabase_client as _get_supabase_client

# Create FastMCP server
mcp = FastMCP("atoms-tools")


def get_supabase_client():
    return _get_supabase_client()


@mcp.tool
async def search_requirements(
    query: str,
    project_id: str | None = None,
    status: str | None = None,
    limit: int = 50
) -> dict[str, Any]:
    """
    Search requirements in the database.

    Args:
        query: Search query to match against title and description
        project_id: Optional project ID to filter by
        status: Optional status to filter by (draft, active, completed, archived)
        limit: Maximum number of results to return (default: 50)

    Returns:
        Dictionary with 'results' (list of requirements) and 'count' (number of results)
    """
    try:
        supabase = get_supabase_client()

        filters: dict[str, str] = {}
        if project_id:
            filters["project_id"] = f"eq.{project_id}"
        if status:
            filters["status"] = f"eq.{status}"
        if query:
            pattern = f"%{query}%"
            filters["or"] = f"(title.ilike.{pattern},description.ilike.{pattern})"

        response = await supabase.select(
            "requirements",
            filters=filters,
            limit=limit,
        )
        results = response.data or []

        return {
            "success": True,
            "results": results,
            "count": len(results),
        }
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "results": [],
            "count": 0,
        }


@mcp.tool
async def create_requirement(
    project_id: str,
    title: str,
    description: str,
    priority: str = "medium",
    status: str = "draft",
    tags: list[str] | None = None,
    metadata: dict[str, Any] | None = None
) -> dict[str, Any]:
    """
    Create a new requirement in the database.
    
    Args:
        project_id: ID of the project this requirement belongs to
        title: Title of the requirement
        description: Detailed description of the requirement
        priority: Priority level (low, medium, high, critical) - default: medium
        status: Status of the requirement (draft, active, completed, archived) - default: draft
        tags: Optional list of tags
        metadata: Optional metadata dictionary
    
    Returns:
        Dictionary with 'success' boolean and 'requirement' object if successful
    """
    try:
        supabase = get_supabase_client()

        requirement_data = {
            "project_id": project_id,
            "title": title,
            "description": description,
            "priority": priority,
            "status": status,
        }

        if tags:
            requirement_data["tags"] = tags
        if metadata:
            requirement_data["metadata"] = metadata

        result = await supabase.insert("requirements", requirement_data)
        requirement = result.data[0] if result.data else None

        return {
            "success": True,
            "requirement": requirement,
        }
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "requirement": None,
        }


@mcp.tool
async def analyze_document(
    document_id: str,
    analysis_type: str = "summary",
    ctx: Context | None = None
) -> dict[str, Any]:
    """
    Analyze a document using Claude AI.
    
    Args:
        document_id: ID of the document to analyze
        analysis_type: Type of analysis (summary, requirements, risks, dependencies)
        ctx: FastMCP context for logging and LLM sampling
    
    Returns:
        Dictionary with analysis results
    """
    try:
        supabase = get_supabase_client()

        doc_result = await supabase.select(
            "documents",
            filters={"id": f"eq.{document_id}"},
            limit=1,
        )

        if not doc_result.data:
            return {
                "success": False,
                "error": f"Document {document_id} not found"
            }

        doc = doc_result.data[0]
        content = doc.get("content", "")

        # Log to client if context available
        if ctx:
            await ctx.info(f"Analyzing document: {doc.get('title', document_id)}")

        # Prepare analysis prompt based on type
        prompts = {
            "summary": f"Provide a concise summary of this document:\n\n{content}",
            "requirements": f"Extract all requirements from this document:\n\n{content}",
            "risks": f"Identify potential risks mentioned in this document:\n\n{content}",
            "dependencies": f"List all dependencies mentioned in this document:\n\n{content}"
        }
        
        prompt = prompts.get(analysis_type, prompts["summary"])
        
        # Use Claude to analyze if context available
        if ctx:
            analysis = await ctx.sample(prompt)
            analysis_text = analysis.text
        else:
            # Fallback: return prompt for manual analysis
            analysis_text = f"Analysis prompt: {prompt}"
        
        return {
            "success": True,
            "document_id": document_id,
            "document_title": doc.get("title"),
            "analysis_type": analysis_type,
            "analysis": analysis_text
        }
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "document_id": document_id
        }


@mcp.tool
def search_codebase(
    query: str,
    file_pattern: str = "*",
    case_sensitive: bool = False,
    max_results: int = 100
) -> dict[str, Any]:
    """
    Search the codebase for code matching the query using ripgrep.
    
    Args:
        query: Search query (regex pattern)
        file_pattern: File pattern to search (e.g., "*.py", "*.ts") - default: all files
        case_sensitive: Whether search should be case-sensitive - default: False
        max_results: Maximum number of results to return - default: 100
    
    Returns:
        Dictionary with search results
    """
    try:
        # Build ripgrep command
        cmd = ["rg", query, "--json"]
        
        if not case_sensitive:
            cmd.append("--ignore-case")
        
        if file_pattern != "*":
            cmd.extend(["--glob", file_pattern])
        
        cmd.extend(["--max-count", str(max_results)])
        
        # Execute ripgrep
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30  # 30 second timeout
        )
        
        # Parse JSON output
        matches = []
        for line in result.stdout.split("\n"):
            if line.strip():
                try:
                    match_data = json.loads(line)
                    if match_data.get("type") == "match":
                        matches.append({
                            "file": match_data.get("data", {}).get("path", {}).get("text"),
                            "line_number": match_data.get("data", {}).get("line_number"),
                            "line": match_data.get("data", {}).get("lines", {}).get("text"),
                        })
                except json.JSONDecodeError:
                    continue
        
        return {
            "success": True,
            "query": query,
            "file_pattern": file_pattern,
            "matches": matches[:max_results],
            "count": len(matches)
        }
    except subprocess.TimeoutExpired:
        return {
            "success": False,
            "error": "Search timed out after 30 seconds",
            "query": query,
            "matches": [],
            "count": 0
        }
    except FileNotFoundError:
        return {
            "success": False,
            "error": "ripgrep (rg) not found. Please install ripgrep.",
            "query": query,
            "matches": [],
            "count": 0
        }
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "query": query,
            "matches": [],
            "count": 0
        }


if __name__ == "__main__":
    # Run the MCP server
    mcp.run()
