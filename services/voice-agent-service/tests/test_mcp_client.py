import pytest

from tools.mcp_client import MCPClient, MCPToolError


class _Response:
    def __init__(self, payload: dict) -> None:
        self._payload = payload

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        return None

    async def json(self, content_type=None) -> dict:
        return self._payload


class _Session:
    closed = False

    def __init__(self, payload: dict) -> None:
        self.payload = payload
        self.requests = []

    def post(self, url: str, json: dict, headers: dict) -> _Response:
        self.requests.append({"url": url, "json": json, "headers": headers})
        return _Response(self.payload)


@pytest.mark.asyncio
async def test_call_tool_posts_mcp_json_rpc() -> None:
    session = _Session(
        {
            "jsonrpc": "2.0",
            "id": 1,
            "result": {
                "content": [{"type": "text", "text": "translated"}],
            },
        }
    )
    client = MCPClient("http://mcp-server:8092", session=session)

    result = await client.call_tool("translate", {"text": "hello", "_auth": "token"})

    assert result == "translated"
    assert session.requests[0]["url"] == "http://mcp-server:8092"
    assert session.requests[0]["json"]["method"] == "tools/call"
    assert session.requests[0]["json"]["params"]["name"] == "translate"
    assert session.requests[0]["json"]["params"]["arguments"]["_auth"] == "token"


@pytest.mark.asyncio
async def test_call_tool_raises_on_tool_error() -> None:
    session = _Session(
        {
            "jsonrpc": "2.0",
            "id": 1,
            "result": {
                "isError": True,
                "content": [{"type": "text", "text": "forbidden"}],
            },
        }
    )
    client = MCPClient("http://mcp-server:8092", session=session)

    with pytest.raises(MCPToolError, match="forbidden"):
        await client.call_tool("search_health_records", {"_auth": "token"})
