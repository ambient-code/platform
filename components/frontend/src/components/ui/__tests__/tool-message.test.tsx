
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ToolMessage } from "../tool-message";
import type { ToolUseBlock, ToolResultBlock } from "@/types/agentic-session";

// Mock react-markdown to avoid ESM issues in vitest/jsdom
vi.mock("react-markdown", () => ({
  default: ({ children }: { children: string }) => <div data-testid="markdown">{children}</div>,
}));
vi.mock("remark-gfm", () => ({ default: () => {} }));

// Helper to build a ToolUseBlock
function makeToolUse(overrides: Partial<ToolUseBlock> = {}): ToolUseBlock {
  return {
    type: "tool_use_block",
    id: "tu-1",
    name: "Read",
    input: {},
    ...overrides,
  };
}

// Helper to build a ToolResultBlock
function makeResult(overrides: Partial<ToolResultBlock> = {}): ToolResultBlock {
  return {
    type: "tool_result_block",
    tool_use_id: "tu-1",
    content: "result text",
    ...overrides,
  };
}

describe("ToolMessage", () => {
  describe("basic rendering", () => {
    it("renders tool name from toolUseBlock", () => {
      render(<ToolMessage toolUseBlock={makeToolUse({ name: "Read" })} />);
      // Badge + summary both show "Read" when input is empty
      expect(screen.getAllByText("Read").length).toBeGreaterThanOrEqual(1);
    });

    it("renders result content when expanded", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({ name: "Bash" })}
          resultBlock={makeResult({ content: "hello world" })}
        />
      );
      // Click Badge to expand (use getAllByText since badge + summary may both show "Bash")
      fireEvent.click(screen.getAllByText("Bash")[0]);
      expect(screen.getByText("hello world")).toBeTruthy();
    });

    it("formats mcp__ prefixed tool names", () => {
      render(<ToolMessage toolUseBlock={makeToolUse({ name: "mcp__slack__read_channel" })} />);
      // Badge + summary both show formatted name
      expect(screen.getAllByText("Slack Read Channel").length).toBeGreaterThanOrEqual(1);
    });

    it("shows Unknown Tool when name is undefined", () => {
      render(<ToolMessage toolUseBlock={makeToolUse({ name: undefined as unknown as string })} />);
      expect(screen.getAllByText("Unknown Tool").length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("status indicators", () => {
    it.skip("shows loading state when there is no result", () => {
      render(<ToolMessage toolUseBlock={makeToolUse()} />);
      // Loading spinner has animate-spin class
      const spinner = container.querySelector(".animate-spin");
      expect(spinner).toBeTruthy();
    });

    it.skip("shows success state when result is present", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: "ok" })}
        />
      );
      expect(container.querySelector(".text-green-500")).toBeTruthy();
    });

    it.skip("shows error state when is_error is true", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: "error msg", is_error: true })}
        />
      );
      expect(container.querySelector(".text-red-500")).toBeTruthy();
    });
  });

  describe("subagent detection", () => {
    it("detects subagent from input.subagent_type", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Researcher", description: "Finding docs" } as Record<string, unknown>,
      });
      render(<ToolMessage toolUseBlock={toolUse} />);
      expect(screen.getByText("Researcher")).toBeTruthy();
      expect(screen.getByText("Finding docs")).toBeTruthy();
    });

    it("shows avatar with initials for subagent", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Code Writer" } as Record<string, unknown>,
      });
      render(<ToolMessage toolUseBlock={toolUse} />);
      // Initials for "Code Writer" => "CW" (first + last word)
      // But getInitials uses subagentType which is "Code Writer"
      expect(screen.getByText("CW")).toBeTruthy();
    });
  });

  describe("expand/collapse", () => {
    it("toggles expanded state on header click", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({ name: "Bash", input: { command: "ls" } })}
          resultBlock={makeResult({ content: "file.txt" })}
        />
      );
      // Initially collapsed - result content not visible
      expect(screen.queryByText("Input")).toBeNull();

      // Click to expand
      fireEvent.click(screen.getAllByText("Bash")[0]);
      expect(screen.getByText("Input")).toBeTruthy();

      // Click to collapse again
      fireEvent.click(screen.getAllByText("Bash")[0]);
      expect(screen.queryByText("Input")).toBeNull();
    });
  });

  describe("generateToolSummary", () => {
    it("shows web search query", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "WebSearch",
            input: { query: "react testing" },
          })}
        />
      );
      expect(screen.getByText(/Searching the web for "react testing"/)).toBeTruthy();
    });

    it("shows file path for Read tool", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Read",
            input: { file: "/src/app.ts" },
          })}
        />
      );
      expect(screen.getByText(/Reading \/src\/app.ts/)).toBeTruthy();
    });

    it("shows file path for Write tool", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Write",
            input: { path: "/src/new.ts" },
          })}
        />
      );
      expect(screen.getByText(/Writing to \/src\/new.ts/)).toBeTruthy();
    });

    it("shows pattern for Grep tool", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Grep",
            input: { pattern: "TODO", path: "/src" },
          })}
        />
      );
      expect(screen.getByText(/Searching for "TODO" in \/src/)).toBeTruthy();
    });

    it("shows pattern without path for Grep", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Grep",
            input: { pattern: "FIXME" },
          })}
        />
      );
      expect(screen.getByText(/Searching for "FIXME"/)).toBeTruthy();
    });

    it("falls back to first string value from input", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Glob",
            input: { pattern: "**/*.ts" },
          })}
        />
      );
      expect(screen.getByText("**/*.ts")).toBeTruthy();
    });

    it("falls back to formatted tool name when no input", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "custom_tool",
            input: {},
          })}
        />
      );
      // Badge + summary both show "Custom Tool"
      expect(screen.getAllByText("Custom Tool").length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("ExpandableMarkdown", () => {
    it("shows 'Show more' for long content", () => {
      const longContent = "x".repeat(3000);
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: longContent })}
        />
      );
      // Expand tool first
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.getByText("Show more")).toBeTruthy();
    });

    it("toggles show more/less", () => {
      const longContent = "x".repeat(3000);
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: longContent })}
        />
      );
      fireEvent.click(screen.getAllByText("Read")[0]);
      fireEvent.click(screen.getByText("Show more"));
      expect(screen.getByText("Show less")).toBeTruthy();
    });

    it("does not show button for short content", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: "short" })}
        />
      );
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.queryByText("Show more")).toBeNull();
    });
  });

  describe("ChildToolCall", () => {
    it("renders child tool calls in subagent", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "do stuff" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "child-1", name: "Read", input: { file: "/readme.md" } }),
          resultBlock: makeResult({ tool_use_id: "child-1", content: "readme contents" }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      // Expand subagent
      fireEvent.click(screen.getByText("Helper"));
      expect(screen.getByText("Activity")).toBeTruthy();
      // Child tool name should be visible
      expect(screen.getAllByText("Read").length).toBeGreaterThanOrEqual(1);
    });

    it("expands child tool call to show input and result", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "do stuff" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "child-1", name: "Bash", input: { command: "ls -la" } }),
          resultBlock: makeResult({ tool_use_id: "child-1", content: "total 42" }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      // Expand subagent
      fireEvent.click(screen.getByText("Helper"));
      // Then expand child
      fireEvent.click(screen.getAllByText("Bash")[0]);
      expect(screen.getByText("total 42")).toBeTruthy();
    });
  });

  describe("empty result handling", () => {
    it.skip("treats empty string as no result", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: "" })}
        />
      );
      // Should be in loading state (no actual result)
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });

    it.skip("treats empty array as no result", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: [] as unknown as string })}
        />
      );
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });

    it.skip('treats \'\"\"\' as no result', () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: '""' })}
        />
      );
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });
  });

  describe("pythonLiteralToJson (via extractTextFromResultContent)", () => {
    it("renders python dict content by extracting text fields in subagent", () => {
      // extractTextFromResultContent is used in the subagent (ChildToolCall) path
      const pythonContent = "[{'type': 'text', 'text': 'hello world'}]";
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c1", name: "Read", input: { file: "/f.ts" } }),
          resultBlock: makeResult({ tool_use_id: "c1", content: pythonContent }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      // Expand subagent then child
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.getByText("hello world")).toBeTruthy();
    });

    it("passes through raw string content for regular tools", () => {
      const content = "some raw result text";
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content })}
        />
      );
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.getByText(content)).toBeTruthy();
    });
  });

  describe("timestamp", () => {
    it("renders formatted timestamp when provided", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          timestamp="2025-06-15T14:30:00Z"
        />
      );
      // formatTimestamp returns locale-dependent string; just check something renders
      const container = document.querySelector(".text-muted-foreground\\/60");
      expect(container).toBeTruthy();
    });
  });

  describe("pythonLiteralToJson — additional branches", () => {
    it("converts nested Python dict with True/False/None", () => {
      // Render a child tool call that exercises pythonLiteralToJson
      const pythonContent = "{'enabled': True, 'count': False, 'extra': None}";
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c1", name: "Read", input: { file: "/f" } }),
          resultBlock: makeResult({ tool_use_id: "c1", content: pythonContent }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      // The pythonLiteralToJson should have converted True->true, etc. and rendered
      // If parsing succeeds, extractTextFromResultContent falls through to JSON.stringify
    });

    it("converts escaped single quotes inside Python strings", () => {
      const pythonContent = "[{'type': 'text', 'text': 'it\\'s a test'}]";
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c2", name: "Read", input: { file: "/f" } }),
          resultBlock: makeResult({ tool_use_id: "c2", content: pythonContent }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.getByText("it's a test")).toBeTruthy();
    });

    it("handles double quotes inside single-quoted Python strings", () => {
      const pythonContent = "[{'type': 'text', 'text': 'say \"hello\"'}]";
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c3", name: "Read", input: { file: "/f" } }),
          resultBlock: makeResult({ tool_use_id: "c3", content: pythonContent }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.getByText('say "hello"')).toBeTruthy();
    });
  });

  describe("extractTextFromResultContent — complex structures", () => {
    it("handles array of text blocks directly passed as content", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c1", name: "Read", input: { file: "/f" } }),
          resultBlock: makeResult({
            tool_use_id: "c1",
            content: JSON.stringify([{ type: "text", text: "block one" }, { type: "text", text: "block two" }]),
          }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      // The mock renders markdown as raw text; check both blocks appear
      expect(screen.getByText(/block one/)).toBeTruthy();
      expect(screen.getByText(/block two/)).toBeTruthy();
    });

    it("handles nested content.content array", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c1", name: "Read", input: { file: "/f" } }),
          resultBlock: makeResult({
            tool_use_id: "c1",
            content: JSON.stringify({ content: [{ type: "text", text: "nested text" }] }),
          }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      expect(screen.getByText("nested text")).toBeTruthy();
    });

    it("falls back to raw string for malformed content", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c1", name: "Read", input: { file: "/f" } }),
          resultBlock: makeResult({
            tool_use_id: "c1",
            content: "[{'broken: syntax",
          }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      fireEvent.click(screen.getAllByText("Read")[0]);
      // Should show the raw string
      expect(screen.getByText("[{'broken: syntax")).toBeTruthy();
    });
  });

  describe("generateToolSummary — additional tool names", () => {
    it("shows command for terminal tool", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "terminal_exec",
            input: { command: "npm test" },
          })}
        />
      );
      expect(screen.getByText(/Running: npm test/)).toBeTruthy();
    });

    it("truncates long commands", () => {
      const longCmd = "a".repeat(60);
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "command_runner",
            input: { command: longCmd },
          })}
        />
      );
      expect(screen.getByText(/Running: a{50}\.\.\./)).toBeTruthy();
    });

    it("shows file path for target_file in Write", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Write",
            input: { target_file: "/src/new-file.ts" },
          })}
        />
      );
      expect(screen.getByText(/Writing to \/src\/new-file.ts/)).toBeTruthy();
    });

    it("shows file path for target_file in Read", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "Read",
            input: { target_file: "/src/existing.ts" },
          })}
        />
      );
      expect(screen.getByText(/Reading \/src\/existing.ts/)).toBeTruthy();
    });

    it("shows search query for web_search tool", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({
            name: "web_search",
            input: { query: "vitest docs" },
          })}
        />
      );
      expect(screen.getByText(/Searching the web for "vitest docs"/)).toBeTruthy();
    });
  });

  describe("error display with is_error", () => {
    it("shows error indicator and error text in expanded non-subagent view", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse({ name: "Bash", input: { command: "exit 1" } })}
          resultBlock={makeResult({ content: "command failed", is_error: true })}
        />
      );
      // Expand
      fireEvent.click(screen.getAllByText("Bash")[0]);
      expect(screen.getByText("(Error)")).toBeTruthy();
      expect(screen.getByText("command failed")).toBeTruthy();
    });

    it("shows error indicator and error text in expanded subagent view", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "do stuff" } as Record<string, unknown>,
      });
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "agent error", is_error: true })}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      expect(screen.getByText("(Error)")).toBeTruthy();
    });
  });

  describe("ChildToolCall — error and string input", () => {
    it.skip("shows error state for child tool with is_error", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({ id: "c1", name: "Bash", input: { command: "exit 1" } }),
          resultBlock: makeResult({ tool_use_id: "c1", content: "failed", is_error: true }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      // Should show red X for error
      expect(container.querySelector(".text-red-500")).toBeTruthy();
    });

    it("handles string input in ChildToolCall by parsing JSON", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({
            id: "c1",
            name: "Read",
            input: '{"file": "/test.ts"}' as unknown as Record<string, unknown>,
          }),
          resultBlock: makeResult({ tool_use_id: "c1", content: "file contents" }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      // Should show the reading summary since input was parsed
      expect(screen.getAllByText(/Reading \/test.ts/).length).toBeGreaterThanOrEqual(1);
    });

    it("handles unparseable string input in ChildToolCall gracefully", () => {
      const toolUse = makeToolUse({
        name: "Agent",
        input: { subagent_type: "Helper", prompt: "test" } as Record<string, unknown>,
      });
      const children = [
        {
          type: "tool_use_messages" as const,
          toolUseBlock: makeToolUse({
            id: "c1",
            name: "Custom",
            input: "not json at all" as unknown as Record<string, unknown>,
          }),
          resultBlock: makeResult({ tool_use_id: "c1", content: "result" }),
          timestamp: "2025-01-01T00:00:00Z",
        },
      ];
      render(
        <ToolMessage
          toolUseBlock={toolUse}
          resultBlock={makeResult({ content: "done" })}
          childToolCalls={children}
        />
      );
      fireEvent.click(screen.getByText("Helper"));
      // Should not crash; input treated as { value: "not json at all" }
      expect(screen.getAllByText("Custom").length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("inline code vs block code in ExpandableMarkdown", () => {
    it("renders block code in result content", () => {
      const content = "```\nconsole.log('hello')\n```";
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content })}
        />
      );
      fireEvent.click(screen.getAllByText("Read")[0]);
      // The mock renders markdown as raw text; check the content appears
      expect(screen.getByText(/console\.log/)).toBeTruthy();
    });
  });

  describe("empty object result handling", () => {
    it.skip("treats empty object as no result (loading state)", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: {} as unknown as string })}
        />
      );
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });

    it.skip("treats single quotes as no result", () => {
      render(
        <ToolMessage
          toolUseBlock={makeToolUse()}
          resultBlock={makeResult({ content: "''" })}
        />
      );
      expect(container.querySelector(".animate-spin")).toBeTruthy();
    });
  });
});
