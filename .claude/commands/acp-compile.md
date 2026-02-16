---
description: Submit a plan file to ACP for execution as an AgenticSession on the cluster.
---

## User Input

```text
$ARGUMENTS
```

## Steps

1. **Locate the plan file**:
   - If `$ARGUMENTS` is a non-empty file path, use that file
   - If `$ARGUMENTS` is empty, find the most recently modified `.md` file in `.claude/plans/`
   - Read the plan file contents — this becomes the `initial_prompt`
   - If no plan file is found, stop and ask the user to provide a path

2. **Get repository info**:
   - Run `git remote get-url origin` to get the repo URL
   - Run `git branch --show-current` to get the current branch

3. **Build the prompt**:
   - Prepend a context header to the plan contents:
     ```
     You are executing a plan that was compiled and submitted to ACP.
     Repository: {repo_url}
     Branch: {branch}

     ---

     {plan_file_contents}
     ```

4. **Create the session**:
   - Call the `acp_create_session` MCP tool with:
     - `initial_prompt`: the assembled prompt from step 3
     - `repos`: `["{repo_url}"]`
     - `display_name`: `"Compiled: {plan_file_basename}"`
     - `interactive`: `false`
     - `timeout`: `1800`
   - If the tool returns `created: false`, print the error message and stop

5. **Report results**:
   - Print the session name and project from the response
   - Print follow-up commands:
     ```
     Check status:  acp_list_sessions(project="...")
     View logs:     acp_get_session_logs(project="...", session="...")
     ```
   - Do NOT wait for the session to complete — return immediately
