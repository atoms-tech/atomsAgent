# FastMCP Service Dockerfile
# Production-ready container for FastMCP service

FROM python:3.11-slim

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy service files
COPY fastmcp_service.py .
COPY fastmcp_wrapper.py .

# Create non-root user
RUN useradd -m -u 1000 fastmcp && \
    chown -R fastmcp:fastmcp /app

# Switch to non-root user
USER fastmcp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run service
CMD ["python", "fastmcp_service.py", "--host", "0.0.0.0", "--port", "8080"]
