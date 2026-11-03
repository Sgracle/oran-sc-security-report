from pathlib import Path
import json
root = Path(__file__).resolve().parents[1]
summary = {}
for variant, expected in [("baseline", (9, 11, 1)), ("fixed", (29, 0, 0))]:
    events = [json.loads(line) for line in (root / f"verification/{variant}-tests.jsonl").read_text().splitlines() if line.startswith("{")]
    completed = {event["Test"]: event["Action"] for event in events if "Test" in event and event["Action"] in ("pass", "fail")}
    leaves = {name: status for name, status in completed.items() if not any(other.startswith(name + "/") for other in completed)}
    passed = sum(status == "pass" for status in leaves.values())
    failed = sum(status == "fail" for status in leaves.values())
    code = int((root / f"verification/{variant}-exit-code.txt").read_text())
    summary[variant] = {"passed": passed, "failed": failed, "exit_code": code, "expected": list(expected), "matches_expected": (passed, failed, code) == expected}
    print(variant, summary[variant])
(root / "verification/summary.json").write_text(json.dumps(summary, indent=2) + "\n")
if not all(item["matches_expected"] for item in summary.values()):
    raise SystemExit("Results differ from the reviewed evidence")
