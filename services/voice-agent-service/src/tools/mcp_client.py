from __future__ import annotations

import itertools
import logging
from typing import Any

import aiohttp
from opentelemetry import propagate, trace

logger = logging.getLogger("zorba.mcp")
tracer = trace.get_tracer("voice-agent-service.mcp")


class MCPToolError(Exception):
    pass


class MCPClient:
    def __init__(self, endpoint: str, session: aiohttp.ClientSession | None = None) -> None:
        self._endpoint = endpoint.rstrip("/")
        self._owned_session = session is None
        self._session = session or aiohttp.ClientSession()
        self._ids = itertools.count(1)

    async def close(self) -> None:
        if self._owned_session and not self._session.closed:
            await self._session.close()

    async def call_tool(self, name: str, arguments: dict[str, Any]) -> str:
        with tracer.start_as_current_span(
            "mcp.tools.call",
            attributes={
                "mcp.tool.name": name,
            },
        ) as span:
            result = await self._rpc("tools/call", {"name": name, "arguments": arguments})
            is_error = bool(result.get("isError"))
            span.set_attribute("mcp.tool.is_error", is_error)
            if is_error:
                raise MCPToolError(_content_text(result) or f"{name} failed")
            return _content_text(result)

    async def _rpc(self, method: str, params: dict[str, Any]) -> dict[str, Any]:
        payload = {
            "jsonrpc": "2.0",
            "id": next(self._ids),
            "method": method,
            "params": params,
        }
        headers = {
            "Accept": "application/json, text/event-stream",
            "Content-Type": "application/json",
        }
        propagate.inject(headers)
        try:
            with tracer.start_as_current_span(
                "mcp.http.request",
                attributes={
                    "http.request.method": "POST",
                    "server.address": self._endpoint,
                    "rpc.method": method,
                },
            ) as span:
                async with self._session.post(self._endpoint, json=payload, headers=headers) as resp:
                    if status := getattr(resp, "status", None):
                        span.set_attribute("http.response.status_code", status)
                    data = await resp.json(content_type=None)
        except Exception as exc:
            logger.error("MCP request failed method=%s: %s", method, exc)
            raise MCPToolError("the clinical tool server is unavailable") from exc

        if data.get("error"):
            err = data["error"]
            message = err.get("message") if isinstance(err, dict) else str(err)
            raise MCPToolError(message or "MCP request failed")
        result = data.get("result")
        if not isinstance(result, dict):
            raise MCPToolError("MCP returned an invalid response")
        return result


def _content_text(result: dict[str, Any]) -> str:
    parts: list[str] = []
    for item in result.get("content") or []:
        if isinstance(item, dict) and item.get("type") == "text" and item.get("text"):
            parts.append(str(item["text"]))
    return "\n".join(parts)
