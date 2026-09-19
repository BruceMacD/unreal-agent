"""Run a real Harbor Docker trial against the deterministic in-container model."""

import argparse
import json
import subprocess
from pathlib import Path
from tempfile import TemporaryDirectory

from harbor.models.trajectories import Trajectory


def smoke(bundle: Path) -> None:
    with TemporaryDirectory(prefix="harness-harbor-smoke-") as directory:
        jobs = Path(directory)
        subprocess.run(
            [
                "harbor",
                "run",
                "-p",
                str(Path(__file__).parent / "task"),
                "-a",
                "harness_harbor.agent:UnrealAgent",
                "-m",
                "openai/smoke",
                "--ak",
                f"bundle={bundle.resolve()}",
                "--ae",
                "OPENAI_API_KEY=harbor-smoke-test-key",
                "--ae",
                "OPENAI_BASE_URL=http://127.0.0.1:8765/v1",
                "-e",
                "docker",
                "-n",
                "1",
                "-k",
                "1",
                "-o",
                str(jobs),
                "--job-name",
                "smoke",
            ],
            check=True,
            timeout=360,
        )
        trials = list(jobs.glob("smoke/*/result.json"))
        assert len(trials) == 1, trials
        trial = trials[0].parent
        result = json.loads(trials[0].read_text())
        if result.get("exception_info"):
            stderr = trial / "agent/runner.stderr"
            detail = stderr.read_text() if stderr.exists() else "No runner stderr"
            raise AssertionError(
                f"{result['exception_info']['exception_message']}\n{detail}"
            )
        assert result["verifier_result"]["rewards"]["reward"] == 1
        trajectory = Trajectory.model_validate_json(
            (trial / "agent/trajectory.json").read_text()
        )
        requests = [
            json.loads(line)
            for line in (trial / "verifier/model-requests.jsonl")
            .read_text()
            .splitlines()
        ]
        assert requests and all(request.get("stream") is True for request in requests)
        observed = set()
        for request in requests:
            for item in request["input"]:
                if item.get("type") != "function_call_output":
                    continue
                output = item["output"]
        assert exported == observed, (exported, observed)
        assert len([s for s in trajectory.steps if s.source == "user"]) == 1
        metrics = result["agent_result"]
        assert metrics["n_input_tokens"] == 10 * len(requests), metrics
        assert metrics["n_cache_tokens"] == 4 * len(requests), metrics
        assert metrics["n_output_tokens"] == 3 * len(requests), metrics
        assert metrics["cost_usd"] is None, metrics
        assert (
            trajectory.agent.extra["binary_sha256"]
            == json.loads((bundle / "manifest.json").read_text())["sha256"]
        )
        sessions = list((trial / "agent/sessions").glob("*.session.jsonl"))
        assert sessions, "No persisted session was collected"
        for session in sessions:
            assert "harbor-smoke-test-key" not in session.read_text()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bundle", type=Path, required=True)
    smoke(parser.parse_args().bundle)
