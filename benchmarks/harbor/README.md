# Harbor evaluation adapter


## Setup and build

From the repository root:

```sh
make -C benchmarks/harbor sync
make -C benchmarks/harbor build REVISION=HEAD
```

Builds committed source into `bin/harbor/<short-commit>/` with a revision and

## Run

Use an absolute bundle path:

```sh
export OPENAI_API_KEY=...
uv run --project benchmarks/harbor --locked harbor run \
  -p /absolute/path/to/task \
  -a harness_harbor.agent:UnrealAgent \
  -m openai/gpt-5.4 \
  --ak bundle="$PWD/bin/harbor/<short-commit>" \
  --ak thinking_level=high
```

`thinking_level`: `low`, `medium`, `high` (default), `xhigh`, or `max`.
Also supports `openrouter/<model>` with `OPENROUTER_API_KEY` and
`fireworks_ai/<model>` with `FIREWORKS_AI_API_KEY`. Use Harbor's `--ae` option for
environment overrides.


For Terminal-Bench 4.0 on Modal, configure Modal credentials and run:

```sh
uv run --project benchmarks/harbor --locked --extra modal harbor run \
  -d terminal-bench/terminal-bench@4.0.0 \
  -e modal -a harness_harbor.agent:UnrealAgent \
  -m openai/gpt-6-astra \
  --ak bundle="$PWD/bin/harbor/<short-commit>" --ak thinking_level=max \
  -k 5 -n 40 --job-name tb4-unreal-agent
```

Inspect results with `harbor view jobs/<job-name>`.


- Logs and sessions are under `agent/`; `agent/trajectory.json` contains ATIF
- Observations are grouped by tool call. Their `extra` sequence, timestamp, and

## Validation

```sh
make -C benchmarks/harbor test check
make -C benchmarks/harbor smoke BUNDLE=../../bin/harbor/<short-commit>
```


Reference: [Harbor agents](https://www.harborframework.com/docs/agents).
