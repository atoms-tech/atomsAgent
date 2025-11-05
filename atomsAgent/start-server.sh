#\!/bin/bash

# atomsAgent Server Startup Script
# This script starts the atomsAgent API server with proper configuration

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Change to the atomsAgent directory
cd "$SCRIPT_DIR"

# Activate virtual environment
source .venv/bin/activate

# Set the config path explicitly
export ATOMS_CONFIG_PATH="$SCRIPT_DIR/config/config.yml"

# Start the server
echo "Starting atomsAgent server on port 3284..."
echo "Config path: $ATOMS_CONFIG_PATH"

uvicorn atomsAgent.main:app --reload --host 0.0.0.0 --port 3284
