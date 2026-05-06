#!/usr/bin/env python3
"""Analyze associative memory nodes for all personas in a simulation instance."""

import json
import sys
from pathlib import Path
from collections import defaultdict
from statistics import median


def percentile(values: list, p: float) -> float:
    if not values:
        return None
    sorted_vals = sorted(values)
    k = (len(sorted_vals) - 1) * p / 100
    lo, hi = int(k), min(int(k) + 1, len(sorted_vals) - 1)
    return round(sorted_vals[lo] + (sorted_vals[hi] - sorted_vals[lo]) * (k - lo), 1)


def field_stats(values: list) -> dict:
    if not values:
        return {"avg": None, "median": None, "p1": None, "p99": None}
    return {
        "avg": round(sum(values) / len(values), 2),
        "median": round(median(values), 2),
        "p1": percentile(values, 1),
        "p99": percentile(values, 99),
    }


def analyze_nodes(nodes: dict) -> dict:
    total = len(nodes)
    if total == 0:
        return {
            "count": 0,
            "expanded_descriptions": 0,
            "poignancy": {"avg": None, "median": None, "p99": None},
            "valence": {"avg": None, "median": None, "p99": None},
            "description_len": {"avg": None, "median": None, "p99": None},
        }

    expanded = sum(
        1 for n in nodes.values()
        if n.get("description") != n.get("original_description")
    )
    poignancy_values = [n["poignancy"] for n in nodes.values() if n.get("poignancy") is not None]
    valence_values = [n["valence"] for n in nodes.values() if n.get("valence") is not None]
    desc_lens = [len(n["description"]) for n in nodes.values() if n.get("description")]

    return {
        "count": total,
        "expanded_descriptions": expanded,
        "poignancy": field_stats(poignancy_values),
        "valence": field_stats(valence_values),
        "description_len": field_stats(desc_lens),
    }


def print_field(pad: str, label: str, s: dict, unit: str = "") -> None:
    if s["avg"] is None:
        return
    u = f" {unit}" if unit else ""
    print(f"{pad}  {label + ':':<26} avg {s['avg']}{u}  |  p1 {s['p1']}{u}  |  median {s['median']}{u}  |  p99 {s['p99']}{u}")


def print_stats(label: str, stats: dict, indent: int = 0) -> None:
    pad = "  " * indent
    print(f"{pad}{label}:")
    print(f"{pad}  count:                  {stats['count']}")
    print(f"{pad}  expanded descriptions:  {stats['expanded_descriptions']}")
    print_field(pad, "poignancy", stats["poignancy"])
    print_field(pad, "valence", stats["valence"])
    print_field(pad, "description length", stats["description_len"], "chars")


def analyze_persona(persona_dir: Path) -> None:
    nodes_path = persona_dir / "bootstrap_memory" / "associative_memory" / "nodes.json"
    if not nodes_path.exists():
        print(f"  [no nodes.json found at {nodes_path}]")
        return

    with nodes_path.open() as f:
        all_nodes: dict = json.load(f)

    by_type: dict[str, dict] = defaultdict(dict)
    for key, node in all_nodes.items():
        by_type[node.get("type", "unknown")][key] = node

    print_stats("overall", analyze_nodes(all_nodes), indent=1)
    for node_type in ("event", "thought", "chat"):
        nodes_of_type = by_type.get(node_type, {})
        print_stats(node_type, analyze_nodes(nodes_of_type), indent=1)

    other_types = set(by_type.keys()) - {"event", "thought", "chat"}
    for node_type in sorted(other_types):
        print_stats(node_type, analyze_nodes(by_type[node_type]), indent=1)


def main() -> None:
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <simulation_instance_dir>")
        sys.exit(1)

    sim_dir = Path(sys.argv[1])
    if not sim_dir.exists():
        print(f"Error: '{sim_dir}' does not exist.")
        sys.exit(1)

    personas_dir = sim_dir / "personas"
    if not personas_dir.exists():
        print(f"Error: no 'personas' directory found in '{sim_dir}'.")
        sys.exit(1)

    personas = sorted(p for p in personas_dir.iterdir() if p.is_dir())
    if not personas:
        print("No personas found.")
        sys.exit(0)

    print(f"Simulation: {sim_dir.name}")
    print(f"Personas:   {len(personas)}\n")

    for persona_dir in personas:
        print(f"[{persona_dir.name}]")
        analyze_persona(persona_dir)
        print()


if __name__ == "__main__":
    main()
