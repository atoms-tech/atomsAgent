"""
Setup script for AgentAPI Python SDK
"""

from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="agentapi-sdk",
    version="0.10.0",
    author="AgentAPI Team",
    author_email="team@agentapi.dev",
    description="Python SDK for AgentAPI - Interact with coding agents via HTTP API",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/coder/agentapi",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: Internet :: WWW/HTTP :: HTTP Servers",
    ],
    python_requires=">=3.8",
    install_requires=[
        "requests>=2.25.0",
    ],
    extras_require={
        "dev": [
            "pytest>=6.0",
            "pytest-cov>=2.0",
            "black>=21.0",
            "flake8>=3.8",
            "mypy>=0.900",
        ],
    },
    keywords="api client sdk agent coding claude droid ccrouter",
    project_urls={
        "Bug Reports": "https://github.com/coder/agentapi/issues",
        "Source": "https://github.com/coder/agentapi",
        "Documentation": "https://github.com/coder/agentapi#readme",
    },
)
