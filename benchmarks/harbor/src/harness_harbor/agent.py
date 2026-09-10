import json
import shlex
from pathlib import Path, PurePosixPath
from tempfile import NamedTemporaryFile
from typing import Any, override
from uuid import uuid4

import certifi
from harbor.agents.installed.base import BaseInstalledAgent, with_prompt_template
from harbor.agents.model_connection import ModelConnectionSpec
from harbor.environments.base import BaseEnvironment
from harbor.models.agent.context import AgentContext
from harbor.models.trajectories import Agent

from harness_harbor.bundle import Bundle
from harness_harbor.trajectory import convert


    SUPPORTS_ATIF = True
    MODEL_CONNECTION = ModelConnectionSpec()

    def __init__(
        self,
        *args: Any,
        bundle: str,
        thinking_level: str = "high",
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)
        if thinking_level not in {"low", "medium", "high", "xhigh", "max"}:
            raise ValueError("Unsupported thinking_level")
        if not self.model_name:
            raise ValueError("Use a provider/model name, e.g. openai/gpt-5.4")
        self._provider, separator, self._model = self.model_name.partition("/")
        if (
            not separator
            or not self._model
            or self._provider not in {"openai", "openrouter", "fireworks_ai"}
        ):
            raise ValueError(
                "Model must use the openai/, openrouter/ or fireworks_ai/ prefix"
            )
        self._bundle = Bundle.load(bundle)
        self._thinking_level = thinking_level
        self._runner_session = str(uuid4())

    @staticmethod
    @override
    def name() -> str:

    @override
    def version(self) -> str:
        return self._bundle.revision

    @override
    async def install(self, environment: BaseEnvironment) -> None:
        await self.exec_as_root(environment, command=f"mkdir -p {self._remote}")
        # Upload a verified snapshot so replacing the local file cannot mix binaries.
        with NamedTemporaryFile() as snapshot:
            snapshot.write(self._bundle.read_binary())
            snapshot.flush()
            await environment.upload_file(
                Path(snapshot.name), str(self._remote / "runner")
            )
        await environment.upload_file(
            Path(certifi.where()), str(self._remote / "ca.pem")
        )
        await self.exec_as_root(
            environment,
            command=(
                f"chmod 755 {self._remote}/runner && "
                f"chmod 644 {self._remote}/ca.pem && "
                f"{self._remote}/runner -h"
            ),
        )

    @with_prompt_template
    @override
    async def run(
        self, instruction: str, environment: BaseEnvironment, context: AgentContext
    ) -> None:
        connection = self.model_connection
        if not connection.api_key:
            raise ValueError(f"No API key configured for {self._provider}")
        logs = self.environment_logs_dir
        request = {
            "prompt": instruction,
            "model": self._model,
            "thinking_level": self._thinking_level,
            "session_id": self._runner_session,
        }
        await self.exec_as_agent(
            environment, command=f"mkdir -p {shlex.quote(str(logs))}"
        )
        await self._upload_config_text(
            environment,
            content=json.dumps(request),
            filename="request.json",
            remote_path=str(logs / "request.json"),
        )
        env = {
            "UNREAL_HARNESS_LLM_PROVIDER": (
                "fireworks" if self._provider == "fireworks_ai" else self._provider
            ),
            "UNREAL_HARNESS_LLM_API_KEY": connection.api_key,
        }
        if connection.configured_base_url:
            env["UNREAL_HARNESS_LLM_BASE_URL"] = connection.configured_base_url
        command = (
            "if [ ! -s /etc/ssl/certs/ca-certificates.crt ]; then "
            f"export SSL_CERT_FILE={self._remote}/ca.pem; fi; "
            'export SHELL="$(command -v bash)"; '
            + shlex.join(
                [
                    str(self._remote / "runner"),
                    "-workspace",
                    ".",
                    "-session-directory",
                    str(logs / "sessions"),
                    "-log-directory",
                    str(logs / "logs"),
                ]
            )
            + f" <{shlex.quote(str(logs / 'request.json'))}"
            + f" >{shlex.quote(str(logs / 'runner.jsonl'))}"
            + f" 2>{shlex.quote(str(logs / 'runner.stderr'))}"
        )
        await self.exec_as_agent(environment, command=command, env=env)

    @override
    def populate_context_post_run(self, context: AgentContext) -> None:
        identity = Agent(
            name=self.name(),
            version=self.version(),
            model_name=self.model_name,
            extra={
                "binary_sha256": self._bundle.sha256,
                "arch": self._bundle.arch,
                "thinking_level": self._thinking_level,
            },
        )
        (self.logs_dir / "trajectory.json").write_text(
            json.dumps(trajectory.to_json_dict(), ensure_ascii=False, indent=2) + "\n"
        )
        metrics = trajectory.final_metrics
        context.n_input_tokens = metrics.total_prompt_tokens
        context.n_cache_tokens = metrics.total_cached_tokens
        context.n_output_tokens = metrics.total_completion_tokens
        context.cost_usd = None
        context.metadata = {
            **(context.metadata or {}),
            **identity.extra,
            **trajectory.extra,
            "revision": self.version(),
        }
