import json
from collections.abc import Iterable
from typing import Any

from harbor.models.trajectories import (
    Agent,
    FinalMetrics,
    Metrics,
    Observation,
    ObservationResult,
    Step,
    ToolCall,
    Trajectory,
)

RUNNING = (
    "Tool call is still running. Its result arrives in a later turn: "
    "continue with independent work, or end your turn to wait for it."
)
TERMINAL = {"completed", "failed", "canceled"}


def bash_result(data: dict[str, Any]) -> str:
    status = data["Status"]
    operations = {op["ID"]: op for op in data.get("Operations", [])}
    waiting = status.get("WaitingFor") or []
    if status.get("Error"):


    steps: list[Step] = []
    calls: dict[str, tuple[Step, str]] = {}
    pending_observations: list[ObservationResult] = []
    errors: list[str] = []
    totals = dict(prompt=0, completion=0, cached=0, reasoning=0, cache_write=0)
    previous_sequence = 0
    for line_number, line in enumerate(lines, 1):
        if not line.strip():
            continue
        try:
            item = json.loads(line)
        except json.JSONDecodeError as exc:
            raise ValueError(f"Invalid runner JSONL at line {line_number}") from exc
        if item.get("type") == "error":
            errors.append(item["message"])
            continue
        sequence = item["Sequence"]
        if sequence <= previous_sequence:
            raise ValueError("Runner records must have increasing sequence numbers")
        previous_sequence = sequence
        kind, data = item["Kind"], item["Data"]
        timestamp = item["RecordedAt"]
        extra = {"sequence": sequence}
        if kind == "turn":
            for result in pending_observations:
                result.extra["available_before_turn"] = data["ID"]
            pending_observations.clear()
        elif kind == "input":
            if data["Kind"] != "external":
                continue
            if not isinstance(data["Payload"], str):
                raise ValueError("External runner input must be text")
            steps.append(
                Step(
                    step_id=len(steps) + 1,
                    source="user",
                    message=data["Payload"],
                    timestamp=timestamp,
                    extra=extra,
                )
            )
        elif kind == "model_response":
            response = data["Response"]
            texts, reasoning, tool_calls = [], [], []
            for output in response.get("Output", []):
                value = output["Data"]
                match output["Type"]:
                    case "message":
                        texts.append(value["Text"])
                    case "reasoning":
                        reasoning.extend(value.get("Summary", []))
                    case "tool_call":
                        raw = value["Arguments"]
                        try:
                            arguments = json.loads(raw)
                        except json.JSONDecodeError:
                            arguments = {"raw_arguments": raw}
                        if not isinstance(arguments, dict):
                            arguments = {"raw_arguments": raw}
                        tool_calls.append(
                            ToolCall(
                                tool_call_id=value["CallID"],
                                function_name=value["Name"],
                                arguments=arguments,
                            )
                        )
                    case _:
                        raise ValueError(f"Unsupported model output: {output['Type']}")
            usage = response["Usage"]
            counts = {
                "prompt": usage["InputTokens"],
                "completion": usage["OutputTokens"],
                "cached": usage["CachedInputTokens"],
                "reasoning": usage["ReasoningTokens"],
                "cache_write": usage["CacheWriteInputTokens"],
            }
            for key, count in counts.items():
                totals[key] += count
            step = Step(
                step_id=len(steps) + 1,
                source="agent",
                timestamp=timestamp,
                message="\n\n".join(texts),
                reasoning_content="\n\n".join(reasoning) or None,
                model_name=agent.model_name,
                tool_calls=tool_calls or None,
                metrics=Metrics(
                    prompt_tokens=counts["prompt"],
                    completion_tokens=counts["completion"],
                    cached_tokens=counts["cached"],
                    extra={
                        "reasoning_tokens": counts["reasoning"],
                        "cache_write_tokens": counts["cache_write"],
                    },
                ),
                llm_call_count=1,
                extra={
                    **extra,
                    "turn_id": data["TurnID"],
                    "stop": response["Stop"],
                    "failure": response.get("Failure"),
                },
            )
            steps.append(step)
            for call in tool_calls:
                if call.tool_call_id in calls:
                    raise ValueError(f"Duplicate tool call: {call.tool_call_id}")
                calls[call.tool_call_id] = step, call.function_name
        elif kind == "tool_call_status":
            step, name = calls[data["CallID"]]
            if name == "Bash":
                content = bash_result(data)
            elif data["Status"].get("Error") and not data.get("Operations"):
                content = data["Status"]["Error"]
            else:
                raise ValueError(f"Unsupported tool result: {name}")
            result = ObservationResult(
                source_call_id=data["CallID"],
                content=content,
                extra={**extra, "timestamp": timestamp},
            )
            if step.observation is None:
                step.observation = Observation(results=[])
            step.observation.results.append(result)
            pending_observations.append(result)
        else:
            raise ValueError(f"Unsupported runner record: {kind}")
    return Trajectory(
        schema_version="ATIF-v1.7",
        session_id=session_id,
        agent=agent,
        steps=steps,
        notes=(
            "Observations are attached to their originating tool calls. "
            "They may arrive asynchronously after later agent steps; use sequence, "
            "timestamp and available_before_turn metadata for ordering. "
            "This is a session audit, not a complete provider-request replay. "
            "Costs are unknown: runner usage contains tokens, not billed amounts."
        ),
        final_metrics=FinalMetrics(
            total_prompt_tokens=totals["prompt"],
            total_completion_tokens=totals["completion"],
            total_cached_tokens=totals["cached"],
            total_steps=len(steps),
            extra={
                "reasoning_tokens": totals["reasoning"],
                "cache_write_tokens": totals["cache_write"],
            },
        ),
        extra={"runner_errors": errors},
    )
