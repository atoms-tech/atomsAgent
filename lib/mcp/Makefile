# Makefile for FastMCP Service

.PHONY: help install test run dev docker-build docker-run docker-stop clean lint format

# Default target
help:
	@echo "FastMCP Service - Available commands:"
	@echo ""
	@echo "  make install       - Install dependencies"
	@echo "  make test          - Run tests"
	@echo "  make run           - Run service in production mode"
	@echo "  make dev           - Run service in development mode (with auto-reload)"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run service in Docker"
	@echo "  make docker-stop   - Stop Docker container"
	@echo "  make clean         - Clean up generated files"
	@echo "  make lint          - Run code linting"
	@echo "  make format        - Format code"
	@echo ""

# Install dependencies
install:
	@echo "Installing dependencies..."
	pip install -r requirements.txt
	pip install pytest pytest-asyncio black flake8 mypy
	@echo "✓ Dependencies installed"

# Run tests
test:
	@echo "Running tests..."
	pytest test_fastmcp_service.py -v
	@echo "✓ Tests completed"

# Run in production mode
run:
	@echo "Starting FastMCP Service (production)..."
	python3 fastmcp_service.py --host 0.0.0.0 --port 8080

# Run in development mode
dev:
	@echo "Starting FastMCP Service (development)..."
	python3 fastmcp_service.py --host 127.0.0.1 --port 8080 --reload --debug

# Run with Gunicorn
run-gunicorn:
	@echo "Starting FastMCP Service with Gunicorn..."
	gunicorn fastmcp_service:app \
		--workers 4 \
		--worker-class uvicorn.workers.UvicornWorker \
		--bind 0.0.0.0:8080 \
		--log-level info

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t fastmcp-service:latest .
	@echo "✓ Docker image built"

# Run in Docker
docker-run:
	@echo "Starting FastMCP Service in Docker..."
	docker-compose up -d
	@echo "✓ Service started in Docker"
	@echo "Access at: http://localhost:8080"

# Stop Docker container
docker-stop:
	@echo "Stopping FastMCP Service..."
	docker-compose down
	@echo "✓ Service stopped"

# View Docker logs
docker-logs:
	docker-compose logs -f

# Clean up
clean:
	@echo "Cleaning up..."
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type f -name "*.log" -delete
	rm -rf .pytest_cache
	rm -rf .mypy_cache
	@echo "✓ Cleanup complete"

# Lint code
lint:
	@echo "Running linters..."
	flake8 fastmcp_service.py --max-line-length=120
	mypy fastmcp_service.py --ignore-missing-imports
	@echo "✓ Linting complete"

# Format code
format:
	@echo "Formatting code..."
	black fastmcp_service.py test_fastmcp_service.py example_usage.py --line-length=120
	@echo "✓ Formatting complete"

# Check health
health:
	@echo "Checking service health..."
	curl -f http://localhost:8080/health || (echo "✗ Service is not responding" && exit 1)
	@echo ""
	@echo "✓ Service is healthy"

# Install systemd service
install-systemd:
	@echo "Installing systemd service..."
	sudo cp fastmcp-service.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable fastmcp-service
	@echo "✓ Systemd service installed"
	@echo "Start with: sudo systemctl start fastmcp-service"

# Start systemd service
start-systemd:
	sudo systemctl start fastmcp-service
	@echo "✓ Service started"

# Stop systemd service
stop-systemd:
	sudo systemctl stop fastmcp-service
	@echo "✓ Service stopped"

# View systemd logs
logs-systemd:
	sudo journalctl -u fastmcp-service -f

# Run example
example:
	@echo "Running example usage..."
	python3 example_usage.py
