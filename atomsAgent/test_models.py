#!/usr/bin/env python3

import os
import sys

sys.path.insert(0, "/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/atomsAgent/src")

# Set environment variables
os.environ["ATOMS_SECRETS_PATH"] = (
    "/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/atomsAgent/config/secrets.yml"
)
os.environ["ATOMS_CONFIG_PATH"] = (
    "/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/atomsAgent/config/config.yml"
)

from google.cloud import aiplatform  # type: ignore[import-untyped]

from atomsAgent.config import settings

# Initialize Vertex AI
project_id = settings.vertex_project_id
location = settings.vertex_location

print(f"Project ID: {project_id}")
print(f"Location: {location}")

try:
    # Initialize Vertex AI
    aiplatform.init(project=project_id, location=location)

    # List models
    print("Available models:")
    models = aiplatform.Model.list()
    claude_models = [m for m in models if "claude" in m.display_name.lower()]
    for model in claude_models:
        print(f"  - {model.display_name} ({model.name})")

    if not claude_models:
        print("  No Claude models found. Listing first 10 models:")
        for model in models[:10]:
            print(f"  - {model.display_name} ({model.name})")

    # Also try listing Anthropic models specifically
    print("\nAnthropic models:")
    try:
        from google.cloud import aiplatform_v1 as aiplatform_v1  # type: ignore[import-untyped]

        client_options = {"api_endpoint": f"{location}-aiplatform.googleapis.com"}
        client = aiplatform_v1.ModelServiceClient(client_options=client_options)

        parent = f"projects/{project_id}/locations/{location}/publishers/anthropic"
        response = client.list_models(parent=parent)

        for model in response:
            print(f"  - {model.display_name} ({model.name})")

    except Exception as e:
        print(f"Error listing Anthropic models: {e}")

except Exception as e:
    print(f"Error: {e}")
