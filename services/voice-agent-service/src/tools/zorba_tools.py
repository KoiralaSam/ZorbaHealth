from __future__ import annotations

import logging

from livekit.agents import RunContext, function_tool

from tools.mcp_client import MCPClient, MCPToolError
from userdata import SessionUserData

logger = logging.getLogger("zorba.tools")


def _client(context: RunContext[SessionUserData]) -> MCPClient:
    client = context.userdata.mcp_client
    if client is None:
        raise MCPToolError("MCP client is not configured for this session")
    return client


@function_tool
async def translate(
    context: RunContext[SessionUserData],
    text: str,
    target_lang: str,
    source_lang: str = "",
) -> str:
    """Translate text into another language.

    Args:
        text: The text to translate.
        target_lang: Target ISO 639-1 language code.
        source_lang: Optional source ISO 639-1 language code.
    """
    ud = context.userdata
    try:
        return await _client(context).call_tool(
            "translate",
            {
                "text": text,
                "targetLang": target_lang,
                "sourceLang": source_lang,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("translate failed session=%s: %s", ud.session_id, exc)
        return "I was unable to translate that right now."


@function_tool
async def get_location(context: RunContext[SessionUserData]) -> str:
    """Get the caller's current location."""
    ud = context.userdata
    try:
        return await _client(context).call_tool(
            "get_location",
            {
                "sessionID": ud.session_id,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("get_location failed session=%s: %s", ud.session_id, exc)
        return "I was unable to determine your location right now."


@function_tool
async def find_nearest_hospital(
    context: RunContext[SessionUserData],
    lat: float,
    lng: float,
    place_type: str = "hospital",
) -> str:
    """Find the nearest hospital, urgent care clinic, or pharmacy.

    Args:
        lat: Latitude from get_location.
        lng: Longitude from get_location.
        place_type: One of hospital, urgent_care, or pharmacy.
    """
    ud = context.userdata
    try:
        return await _client(context).call_tool(
            "find_nearest_hospital",
            {
                "lat": lat,
                "lng": lng,
                "placeType": place_type,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("find_nearest_hospital failed session=%s: %s", ud.session_id, exc)
        return "I was unable to find nearby facilities right now. Please call 9-1-1 if this is an emergency."


@function_tool
async def search_health_records(
    context: RunContext[SessionUserData],
    query: str,
    top_k: int = 5,
) -> str:
    """Search verified patient health records.

    Only call this after identity verification has provided a patient token.

    Args:
        query: Health-record search query.
        top_k: Maximum number of matching chunks to return.
    """
    ud = context.userdata
    if not ud.is_verified:
        return "Identity verification is required before I can access personal health records."
    try:
        return await _client(context).call_tool(
            "search_health_records",
            {
                "query": query,
                "topK": top_k,
                "_auth": ud.active_token,
            },
        )
    except MCPToolError as exc:
        logger.warning("search_health_records failed session=%s: %s", ud.session_id, exc)
        return "I was unable to search your health records right now."


ALL_TOOLS = [
    translate,
    get_location,
    find_nearest_hospital,
    search_health_records,
]
