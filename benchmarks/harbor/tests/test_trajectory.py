import json
import unittest

from harbor.models.trajectories import Agent, Trajectory

from harness_harbor.trajectory import RUNNING, bash_result, convert


def record(sequence, kind, data):
    return json.dumps(
        {
            "Sequence": sequence,
            "Kind": kind,
            "Data": data,
            "RecordedAt": "2026-09-10T10:00:00Z",
        }
    )


def response(turn, output):
    return {
        "TurnID": turn,
        "Response": {
            "ID": turn,
            "Stop": "complete",
            "Output": output,
            "Usage": {
                "InputTokens": 10,
                "CachedInputTokens": 4,
                "CacheWriteInputTokens": 0,
                "OutputTokens": 3,
                "ReasoningTokens": 1,
            },
        },
    }


def status(state="completed", stdout="test", **fields):
    return {
        "CallID": "call-1",
        "Status": {"WaitingFor": ["op-1"]},
        "Operations": [
            {
                "ID": "op-1",
                "Type": "shell",
                "Status": state,
                "State": {
                    "Result": {"Out": stdout, "Err": "", "ExitCode": 0},
                    **fields,
                },
            }
        ],
    }


class TrajectoryTests(unittest.TestCase):
    def test_plain_text_is_not_base64_decoded(self):
            with self.subTest(text=text):

    def test_running_and_failed_operations(self):

    def test_internal_control_and_delayed_observations(self):
        tool_call = {
            "Type": "tool_call",
            "Data": {
                "CallID": "call-1",
                "Name": "Bash",
                "Arguments": '{"command":"printf test"}',
            },
        }
        lines = [
            record(1, "input", {"Kind": "external", "Payload": '"literal quotes"'}),
            record(2, "input", {"Kind": "control", "Payload": {"Mode": "when_idle"}}),
            record(3, "turn", {"ID": "turn-1"}),
            record(4, "model_response", response("turn-1", [tool_call])),
            record(5, "tool_call_status", status(state="ready")),
            record(6, "turn", {"ID": "turn-2"}),
            record(7, "model_response", response("turn-2", [])),
            record(8, "tool_call_status", status()),
            record(9, "turn", {"ID": "turn-3"}),
            record(10, "model_response", response("turn-3", [])),
        ]
        trajectory = convert(
            lines, Agent(name="unreal-agent", version="test"), "session"
        )
        self.assertEqual(len(trajectory.steps), 4)
        self.assertEqual(trajectory.steps[0].message, '"literal quotes"')
        observations = trajectory.steps[1].observation.results
        self.assertEqual(observations[0].content, RUNNING)
        self.assertEqual(observations[0].extra["available_before_turn"], "turn-2")
        self.assertEqual(observations[1].extra["available_before_turn"], "turn-3")
        self.assertEqual(trajectory.final_metrics.total_prompt_tokens, 30)
        self.assertEqual(trajectory.final_metrics.total_cached_tokens, 12)
        self.assertEqual(trajectory.final_metrics.total_completion_tokens, 9)
        self.assertIsNone(trajectory.final_metrics.total_cost_usd)
        Trajectory.model_validate(trajectory.to_json_dict())

    def test_only_latest_pending_running_result_is_marked_available(self):
        for state in ("ready", "completed"):
            with self.subTest(state=state):
                calls = [
                    {
                        "Type": "tool_call",
                        "Data": {
                            "CallID": call_id,
                            "Name": "Bash",
                            "Arguments": '{"command":"printf test"}',
                        },
                    }
                    for call_id in ("call-1", "call-2")
                ]
                other_status = {**status(state="ready"), "CallID": "call-2"}
                lines = [
                    record(1, "model_response", response("turn-1", calls)),
                    record(2, "tool_call_status", status(state="ready")),
                    record(3, "tool_call_status", other_status),
                    record(4, "input", {"Kind": "external", "Payload": "continue"}),
                    record(5, "tool_call_status", status(state=state)),
                    record(6, "turn", {"ID": "turn-2"}),
                ]
                trajectory = convert(
                    lines, Agent(name="unreal-agent", version="test"), "session"
                )
                observations = trajectory.steps[0].observation.results
                self.assertEqual(len(observations), 3)
                self.assertEqual(observations[0].content, RUNNING)
                self.assertNotIn("available_before_turn", observations[0].extra)
                self.assertEqual(observations[1].source_call_id, "call-2")
                self.assertEqual(observations[1].content, RUNNING)
                self.assertEqual(
                    observations[1].extra["available_before_turn"], "turn-2"
                )
                self.assertEqual(
                    observations[2].content, bash_result(status(state=state))
                )
                self.assertEqual(
                    observations[2].extra["available_before_turn"], "turn-2"
                )
                self.assertEqual(observations[2].extra["sequence"], 5)

    def test_malformed_record_is_an_error_including_partial_final_line(self):
        for lines in (["{"], ['{"Kind":']):
            with self.assertRaisesRegex(ValueError, "line 1"):
                convert(lines, Agent(name="unreal-agent", version="test"), "session")

    def test_validation_error_and_incomplete_run_keep_usage(self):
        lines = [
            record(
                1,
                "model_response",
                response(
                    "turn-1",
                    [
                        {
                            "Type": "tool_call",
                            "Data": {
                                "CallID": "call-1",
                                "Name": "Bash",
                                "Arguments": "{",
                            },
                        }
                    ],
                ),
            ),
            record(
                2,
                "tool_call_status",
                {"CallID": "call-1", "Status": {"Error": "bad JSON"}},
            ),
            json.dumps({"type": "error", "message": "provider disconnected"}),
        ]
        trajectory = convert(
            lines, Agent(name="unreal-agent", version="test"), "session"
        )
        self.assertEqual(trajectory.final_metrics.total_prompt_tokens, 10)
        self.assertEqual(trajectory.extra["runner_errors"], ["provider disconnected"])
        self.assertEqual(
        )


if __name__ == "__main__":
    unittest.main()
