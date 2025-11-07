"""
Test Artifact Detection and Extraction
"""

import sys
sys.path.insert(0, 'src')

from atomsAgent.services.artifacts import (
    detect_artifacts,
    extract_artifacts,
    format_artifact_for_frontend,
    ArtifactType
)


def test_react_artifact():
    """Test React artifact extraction"""
    print("\n1️⃣  Testing React Artifact...")
    
    text = '''
Here's a React component for you:

<artifact type="react" title="Counter Component" language="tsx">
import React, { useState } from 'react';

export default function Counter() {
    const [count, setCount] = useState(0);
    
    return (
        <div>
            <h1>Count: {count}</h1>
            <button onClick={() => setCount(count + 1)}>Increment</button>
        </div>
    );
}
</artifact>

This component demonstrates state management.
'''
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 1, f"Expected 1 artifact, got {len(artifacts)}"
    assert artifacts[0]["type"] == "react"
    assert artifacts[0]["title"] == "Counter Component"
    assert "useState" in artifacts[0]["content"]
    assert "<artifact" not in cleaned
    
    formatted = format_artifact_for_frontend(artifacts[0])
    assert formatted["renderMode"] == "preview"
    assert formatted["editable"] == True
    
    print("   ✅ React artifact extracted successfully")
    print(f"      - Title: {artifacts[0]['title']}")
    print(f"      - Type: {artifacts[0]['type']}")
    print(f"      - Language: {artifacts[0]['language']}")
    print(f"      - Content length: {len(artifacts[0]['content'])} chars")


def test_html_artifact():
    """Test HTML artifact extraction"""
    print("\n2️⃣  Testing HTML Artifact...")
    
    text = '''
<artifact type="html" title="Landing Page">
<\!DOCTYPE html>
<html>
<head>
    <title>My Page</title>
</head>
<body>
    <h1>Welcome\!</h1>
</body>
</html>
</artifact>
'''
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 1
    assert artifacts[0]["type"] == "html"
    assert artifacts[0]["title"] == "Landing Page"
    assert "<\!DOCTYPE html>" in artifacts[0]["content"]
    
    print("   ✅ HTML artifact extracted successfully")


def test_mermaid_artifact():
    """Test Mermaid diagram extraction"""
    print("\n3️⃣  Testing Mermaid Artifact...")
    
    text = '''
<artifact type="mermaid" title="System Architecture">
graph TD
    A[Client] --> B[API Gateway]
    B --> C[Service 1]
    B --> D[Service 2]
</artifact>
'''
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 1
    assert artifacts[0]["type"] == "mermaid"
    assert artifacts[0]["title"] == "System Architecture"
    assert "graph TD" in artifacts[0]["content"]
    
    formatted = format_artifact_for_frontend(artifacts[0])
    assert formatted["renderMode"] == "diagram"
    
    print("   ✅ Mermaid artifact extracted successfully")


def test_svg_artifact():
    """Test SVG artifact extraction"""
    print("\n4️⃣  Testing SVG Artifact...")
    
    text = '''
<artifact type="svg" title="Circle Icon">
<svg width="100" height="100">
    <circle cx="50" cy="50" r="40" fill="blue" />
</svg>
</artifact>
'''
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 1
    assert artifacts[0]["type"] == "svg"
    assert artifacts[0]["title"] == "Circle Icon"
    assert "<circle" in artifacts[0]["content"]
    
    print("   ✅ SVG artifact extracted successfully")


def test_multiple_artifacts():
    """Test multiple artifacts in one response"""
    print("\n5️⃣  Testing Multiple Artifacts...")
    
    text = '''
Here are two components:

<artifact type="react" title="Button">
export default function Button() {
    return <button>Click me</button>;
}
</artifact>

And here's a diagram:

<artifact type="mermaid" title="Flow">
graph LR
    A --> B
</artifact>

That's all\!
'''
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 2
    assert artifacts[0]["type"] == "react"
    assert artifacts[1]["type"] == "mermaid"
    assert "Here are two components:" in cleaned
    assert "That's all\!" in cleaned
    assert "<artifact" not in cleaned
    
    print("   ✅ Multiple artifacts extracted successfully")
    print(f"      - Found {len(artifacts)} artifacts")
    print(f"      - Cleaned text length: {len(cleaned)} chars")


def test_code_block_artifact():
    """Test code block marked as artifact"""
    print("\n6️⃣  Testing Code Block Artifact...")
    
    text = '''
```python // artifact: Data Processing Script
def process_data(data):
    return [x * 2 for x in data]
```
'''
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 1
    assert artifacts[0]["type"] == "code"
    assert artifacts[0]["language"] == "python"
    assert "process_data" in artifacts[0]["content"]
    
    print("   ✅ Code block artifact extracted successfully")


def test_no_artifacts():
    """Test text with no artifacts"""
    print("\n7️⃣  Testing No Artifacts...")
    
    text = "This is just regular text with no artifacts."
    
    cleaned, artifacts = extract_artifacts(text)
    
    assert len(artifacts) == 0
    assert cleaned == text
    
    print("   ✅ No artifacts detected (as expected)")


def main():
    """Run all tests"""
    print("\n🧪 Testing Artifact Detection and Extraction\n")
    print("=" * 60)
    
    try:
        test_react_artifact()
        test_html_artifact()
        test_mermaid_artifact()
        test_svg_artifact()
        test_multiple_artifacts()
        test_code_block_artifact()
        test_no_artifacts()
        
        print("\n" + "=" * 60)
        print("\n✅ All Artifact Tests Passed\!\n")
        
        print("Summary:")
        print("  - React artifacts: ✅")
        print("  - HTML artifacts: ✅")
        print("  - Mermaid diagrams: ✅")
        print("  - SVG graphics: ✅")
        print("  - Multiple artifacts: ✅")
        print("  - Code block artifacts: ✅")
        print("  - No artifacts: ✅")
        
    except AssertionError as e:
        print(f"\n❌ Test failed: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
