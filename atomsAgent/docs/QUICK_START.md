# Atoms Agent Dev UI - Quick Start

## Installation

```bash
cd atomsAgent
uv pip install -e '.[dev]'
```

## Launch UI

```bash
# Local
atoms-agent dev-ui launch

# Share with team
atoms-agent dev-ui launch --share
```

## CLI Chat

```bash
# Interactive
atoms-agent chat interactive

# Single message
atoms-agent chat once "Your prompt here"
```

## Test

```bash
# Test completion
atoms-agent test completion --prompt "Hello\!"

# Test streaming
atoms-agent test streaming --prompt "Count to 10"

# List models
atoms-agent test models
```

## Common Options

```bash
--model claude-3-5-sonnet-20241022
--temperature 0.7
--max-tokens 2048
--system-prompt "Custom prompt"
```

## Help

```bash
atoms-agent --help
atoms-agent COMMAND --help
```
