"""
SDK Configuration Loader

Fetches SDK configuration from backend API and applies to ClaudeAgentOptions.
"""

import os
import logging
from typing import Dict, Any, Optional
from urllib import request as _urllib_request
import json

logger = logging.getLogger(__name__)


class SDKConfigLoader:
    """Loads SDK configuration from backend API."""

    def __init__(self, session_id: str, project_name: str):
        self.session_id = session_id
        self.project_name = project_name
        self.backend_url = os.getenv('BACKEND_API_URL', '').rstrip('/')
        self.bot_token = os.getenv('BOT_TOKEN', '').strip()

    async def load_configuration(self) -> Optional[Dict[str, Any]]:
        """
        Load SDK configuration from backend API.

        Returns:
            Configuration dict or None if not found/error
        """
        if not self.backend_url:
            logger.warning("BACKEND_API_URL not set, using default configuration")
            return None

        url = f"{self.backend_url}/api/projects/{self.project_name}/sdk/configuration/session/{self.session_id}"
        logger.info(f"Loading SDK configuration from: {url}")

        try:
            req = _urllib_request.Request(url, method='GET')
            if self.bot_token:
                req.add_header('Authorization', f'Bearer {self.bot_token}')

            with _urllib_request.urlopen(req, timeout=10) as resp:
                data = resp.read().decode('utf-8')
                config = json.loads(data)
                logger.info(f"Loaded SDK configuration: model={config.get('model')}, tools={len(config.get('allowedTools', []))}")
                return config

        except Exception as e:
            logger.warning(f"Failed to load SDK configuration: {e}, using defaults")
            return None

    def apply_to_options(self, options: Any, config: Dict[str, Any]) -> None:
        """
        Apply configuration to ClaudeAgentOptions.

        Args:
            options: ClaudeAgentOptions instance
            config: Configuration dict from API
        """
        if not config:
            return

        # Apply model
        if config.get('model'):
            try:
                options.model = config['model']
                logger.info(f"Applied model: {config['model']}")
            except Exception as e:
                logger.warning(f"Failed to set model: {e}")

        # Apply max tokens
        if config.get('maxTokens'):
            try:
                options.max_tokens = config['maxTokens']
                logger.info(f"Applied max_tokens: {config['maxTokens']}")
            except Exception as e:
                logger.warning(f"Failed to set max_tokens: {e}")

        # Apply temperature
        if config.get('temperature') is not None:
            try:
                options.temperature = config['temperature']
                logger.info(f"Applied temperature: {config['temperature']}")
            except Exception as e:
                logger.warning(f"Failed to set temperature: {e}")

        # Apply permission mode
        if config.get('permissionMode'):
            try:
                options.permission_mode = config['permissionMode']
                logger.info(f"Applied permission_mode: {config['permissionMode']}")
            except Exception as e:
                logger.warning(f"Failed to set permission_mode: {e}")

        # Apply allowed tools
        if config.get('allowedTools'):
            try:
                # Merge with existing tools (don't remove MCP tools)
                existing_mcp_tools = [t for t in options.allowed_tools if t.startswith('mcp__')]
                options.allowed_tools = config['allowedTools'] + existing_mcp_tools
                logger.info(f"Applied allowed_tools: {len(config['allowedTools'])} tools")
            except Exception as e:
                logger.warning(f"Failed to set allowed_tools: {e}")

        # Apply streaming
        if config.get('includePartialMessages') is not None:
            try:
                options.include_partial_messages = config['includePartialMessages']
                logger.info(f"Applied include_partial_messages: {config['includePartialMessages']}")
            except Exception as e:
                logger.warning(f"Failed to set include_partial_messages: {e}")

        # Apply conversation continuation
        if config.get('continueConversation') is not None:
            try:
                # Note: This is typically set based on IS_RESUME, not user config
                # Only apply if explicitly overriding
                pass
            except Exception as e:
                logger.warning(f"Failed to set continue_conversation: {e}")

        # Apply system prompt (merge with workspace context)
        if config.get('systemPrompt'):
            try:
                existing_prompt = options.system_prompt.get('text', '') if options.system_prompt else ''
                custom_prompt = config['systemPrompt']
                merged_prompt = f"{existing_prompt}\n\n## Custom Instructions\n{custom_prompt}"
                options.system_prompt = {"type": "text", "text": merged_prompt}
                logger.info("Applied custom system prompt")
            except Exception as e:
                logger.warning(f"Failed to set system_prompt: {e}")

        # MCP servers are handled separately in adapter.py
        # This loader only modifies ClaudeAgentOptions

    def merge_mcp_servers(self, existing_servers: Dict[str, Any], config: Dict[str, Any]) -> Dict[str, Any]:
        """
        Merge MCP servers from configuration with existing servers.

        Args:
            existing_servers: Existing MCP servers (from .mcp.json)
            config: Configuration dict from API

        Returns:
            Merged MCP servers dict
        """
        if not config or not config.get('mcpServers'):
            return existing_servers

        merged = existing_servers.copy()

        for name, server_config in config['mcpServers'].items():
            if not server_config.get('enabled', True):
                # Skip disabled servers
                continue

            # Convert frontend format to SDK format
            merged[name] = {
                "command": server_config['command'],
                "args": server_config.get('args', []),
            }

            if server_config.get('env'):
                merged[name]["env"] = server_config['env']

            logger.info(f"Added MCP server from config: {name}")

        return merged


async def load_and_apply_sdk_config(options: Any, session_id: str, project_name: str) -> None:
    """
    Load SDK configuration from backend and apply to options.

    Convenience function for use in adapter.py.

    Args:
        options: ClaudeAgentOptions instance
        session_id: Current session ID
        project_name: Current project name
    """
    loader = SDKConfigLoader(session_id, project_name)
    config = await loader.load_configuration()

    if config:
        loader.apply_to_options(options, config)
        logger.info("SDK configuration applied successfully")
    else:
        logger.info("Using default SDK configuration")
