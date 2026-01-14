#!/usr/bin/env python3
"""
Script to generate JSON schemas from prompt files based on their example outputs.
"""
import json
import re
from pathlib import Path
from typing import Any, Dict, List, Union


def infer_schema_from_value(value: Any) -> Dict[str, Any]:
    """Infer JSON schema from a Python value."""
    if isinstance(value, dict):
        properties = {}
        required = []
        for k, v in value.items():
            properties[k] = infer_schema_from_value(v)
            required.append(k)
        return {
            "type": "object",
            "properties": properties,
            "required": required,
            "additionalProperties": False
        }
    elif isinstance(value, list):
        if len(value) == 0:
            return {"type": "array", "items": {}}
        item_schema = infer_schema_from_value(value[0])
        return {
            "type": "array",
            "items": item_schema
        }
    elif isinstance(value, str):
        return {"type": "string"}
    elif isinstance(value, int):
        return {"type": "integer"}
    elif isinstance(value, float):
        return {"type": "number"}
    elif isinstance(value, bool):
        return {"type": "boolean"}
    elif value is None:
        return {"type": "null"}
    else:
        return {"type": "string"}


def extract_json_after_format(text: str) -> Union[Dict, List, None]:
    """Extract JSON example after 'Format:' in the text."""
    # Find "Format:" (case insensitive)
    format_idx = text.lower().find('format:')
    if format_idx == -1:
        return None

    # Get everything after "Format:"
    after_format = text[format_idx + 7:].strip()

    # Find the first { or [
    first_brace = after_format.find('{')
    first_bracket = after_format.find('[')

    if first_brace == -1 and first_bracket == -1:
        return None

    # Start from the first JSON structure
    start_pos = min(filter(lambda x: x != -1, [first_brace, first_bracket]))

    # Extract complete JSON by matching braces/brackets
    brace_count = 0
    bracket_count = 0
    in_string = False
    escape_next = False

    for i in range(start_pos, len(after_format)):
        char = after_format[i]

        if escape_next:
            escape_next = False
            continue

        if char == '\\':
            escape_next = True
            continue

        if char == '"' and not escape_next:
            in_string = not in_string
            continue

        if in_string:
            continue

        if char == '{':
            brace_count += 1
        elif char == '}':
            brace_count -= 1
            if brace_count == 0 and bracket_count == 0:
                json_str = after_format[start_pos:i+1]
                return clean_and_parse(json_str)
        elif char == '[':
            bracket_count += 1
        elif char == ']':
            bracket_count -= 1
            if brace_count == 0 and bracket_count == 0:
                json_str = after_format[start_pos:i+1]
                return clean_and_parse(json_str)

    return None


def remove_ellipsis_tokens(s: str) -> str:
    # --- Ellipsis handling (robust) ---
    # Remove ellipsis tokens (...) or (…) that appear OUTSIDE strings, without deleting commas.
    out = []
    in_string = False
    escape_next = False
    i = 0
    while i < len(s):
        ch = s[i]

        if escape_next:
            out.append(ch)
            escape_next = False
            i += 1
            continue

        if ch == '\\':
            out.append(ch)
            escape_next = True
            i += 1
            continue

        if ch == '"':
            out.append(ch)
            in_string = not in_string
            i += 1
            continue

        if not in_string:
            # Unicode ellipsis
            if ch == '…':
                i += 1
                continue
            # Three-dot ellipsis
            if s.startswith('...', i):
                i += 3
                continue

        out.append(ch)
        i += 1

    return ''.join(out)


def clean_and_parse(json_str: str) -> Union[Dict, List, None]:
    """Replace placeholders to make valid JSON, then parse it."""
    # Replace placeholders inside strings (e.g., "text !<INPUT 4>!" -> "text example")
    def replace_in_string(match):
        content = match.group(1)
        content = re.sub(r'!<INPUT \d+>!', 'example', content)
        content = re.sub(r'<[^>]+>', 'example', content)
        return '"' + content + '"'

    json_str = re.sub(r'"([^"]*!<INPUT \d+>[^"]*)"',
                      replace_in_string, json_str)
    json_str = re.sub(r'"([^"]*<[^>]+>[^"]*)"', replace_in_string, json_str)

    # Replace standalone placeholders with quoted values
    json_str = re.sub(r'!<INPUT \d+>!', '"example"', json_str)
    json_str = re.sub(r'<[^>]+>', '"example"', json_str)

    json_str = remove_ellipsis_tokens(json_str)

    # Clean up artifacts caused by removing ellipsis placeholders
    json_str = re.sub(r',\s*,', ',', json_str)              # ", ," -> ","
    # trailing comma before ] or }
    json_str = re.sub(r',\s*(\]|\})', r'\1', json_str)

    try:
        return json.loads(json_str)
    except json.JSONDecodeError:
        print(json_str)
        return None


def generate_schema_for_prompt(prompt_path: Path) -> bool:
    """Generate a JSON schema file for a prompt file."""
    try:
        with open(prompt_path, 'r', encoding='utf-8') as f:
            prompt_content = f.read()

        example = extract_json_after_format(prompt_content)

        if example is None:
            print(f"Warning: Could not extract example from {prompt_path}")
            return False

        schema = infer_schema_from_value(example)
        schema["$schema"] = "http://json-schema.org/draft-07/schema#"

        # Write schema to schema.json in the same directory
        schema_path = prompt_path.parent / "schema.json"
        with open(schema_path, 'w', encoding='utf-8') as f:
            json.dump(schema, f, indent=2)

        print(f"Generated schema for {prompt_path.name}")
        return True

    except Exception as e:
        print(f"Error processing {prompt_path}: {e}")
        return False


def main():
    """Main function to process all prompt files."""
    base_dir = Path(".")
    prompt_files = list(base_dir.glob("*/prompt.txt"))

    if not prompt_files:
        print("No prompt.txt files found!")
        return

    print(f"Found {len(prompt_files)} prompt files")

    success_count = 0
    for prompt_file in sorted(prompt_files):
        if generate_schema_for_prompt(prompt_file):
            success_count += 1

    print(
        f"\nSuccessfully generated {success_count} schemas out of {len(prompt_files)} prompt files")


if __name__ == "__main__":
    main()
