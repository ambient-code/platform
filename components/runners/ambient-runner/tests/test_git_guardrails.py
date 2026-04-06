"""Unit tests for git_guardrails module."""

import pytest

from ambient_runner.platform.git_guardrails import (
    GitGuardrailViolation,
    check_command,
    format_violations,
    has_blocking_violation,
    redact_tokens_in_command,
)


class TestCheckCommand:
    """Tests for check_command function."""

    # -- Blocked patterns --------------------------------------------------

    def test_delete_remote_ref_via_gh_api(self):
        """Deleting a remote ref via gh api should be blocked."""
        cmd = 'gh api -X DELETE "repos/user/repo/git/refs/heads/my-branch"'
        violations = check_command(cmd)
        assert any(v.rule == "delete_remote_ref" and v.severity == "block" for v in violations)

    def test_delete_remote_ref_via_curl(self):
        """Deleting a remote ref via curl should be blocked."""
        cmd = 'curl -X DELETE https://api.github.com/repos/user/repo/git/refs/heads/branch'
        violations = check_command(cmd)
        assert any(v.rule == "delete_remote_ref" and v.severity == "block" for v in violations)

    def test_api_force_update_ref(self):
        """Force-updating a ref via API should be blocked."""
        cmd = (
            'gh api -X PATCH repos/user/repo/git/refs/heads/branch '
            '-f sha=abc123 -f "force":true'
        )
        violations = check_command(cmd)
        assert any(
            v.rule == "api_force_update_ref" and v.severity == "block" for v in violations
        )

    def test_api_create_commit(self):
        """Creating commits directly via API should be blocked."""
        cmd = 'gh api -X POST repos/user/repo/git/commits -f message="direct commit"'
        violations = check_command(cmd)
        assert any(
            v.rule == "api_create_commit_on_ref" and v.severity == "block"
            for v in violations
        )

    def test_api_create_tree(self):
        """Creating trees directly via API should be blocked."""
        cmd = 'curl -X POST https://api.github.com/repos/user/repo/git/trees'
        violations = check_command(cmd)
        assert any(
            v.rule == "api_create_commit_on_ref" and v.severity == "block"
            for v in violations
        )

    def test_api_create_blob(self):
        """Creating blobs directly via API should be blocked."""
        cmd = 'gh api -X POST repos/user/repo/git/blobs -f content="data"'
        violations = check_command(cmd)
        assert any(
            v.rule == "api_create_commit_on_ref" and v.severity == "block"
            for v in violations
        )

    def test_force_push(self):
        """git push --force should be blocked."""
        cmd = "git push origin my-branch --force"
        violations = check_command(cmd)
        assert any(v.rule == "force_push" and v.severity == "block" for v in violations)

    def test_force_push_short_flag(self):
        """git push -f should be blocked."""
        cmd = "git push origin my-branch -f"
        violations = check_command(cmd)
        assert any(
            v.rule == "force_push_short" and v.severity == "block" for v in violations
        )

    def test_force_with_lease_not_blocked_as_force(self):
        """git push --force-with-lease should NOT trigger the force_push rule."""
        cmd = "git push origin my-branch --force-with-lease"
        violations = check_command(cmd)
        assert not any(v.rule == "force_push" for v in violations)

    def test_push_to_main(self):
        """Pushing to main should be blocked."""
        cmd = "git push origin main"
        violations = check_command(cmd)
        assert any(v.rule == "push_to_main" and v.severity == "block" for v in violations)

    def test_push_to_master(self):
        """Pushing to master should be blocked."""
        cmd = "git push origin master"
        violations = check_command(cmd)
        assert any(v.rule == "push_to_main" and v.severity == "block" for v in violations)

    def test_push_to_feature_branch_allowed(self):
        """Pushing to a feature branch should not trigger push_to_main."""
        cmd = "git push origin feature/my-branch"
        violations = check_command(cmd)
        assert not any(v.rule == "push_to_main" for v in violations)

    def test_reset_hard(self):
        """git reset --hard should be blocked."""
        cmd = "git reset --hard origin/main"
        violations = check_command(cmd)
        assert any(v.rule == "reset_hard" and v.severity == "block" for v in violations)

    def test_clean_force(self):
        """git clean -fd should be blocked."""
        cmd = "git clean -fd"
        violations = check_command(cmd)
        assert any(v.rule == "clean_force" and v.severity == "block" for v in violations)

    def test_clean_force_extended(self):
        """git clean -fdx should be blocked."""
        cmd = "git clean -fdx"
        violations = check_command(cmd)
        assert any(v.rule == "clean_force" and v.severity == "block" for v in violations)

    def test_checkout_discard_all(self):
        """git checkout -- . should be blocked."""
        cmd = "git checkout -- ."
        violations = check_command(cmd)
        assert any(
            v.rule == "checkout_discard" and v.severity == "block" for v in violations
        )

    def test_branch_delete_remote(self):
        """git push --delete should be blocked."""
        cmd = "git push origin --delete my-branch"
        violations = check_command(cmd)
        assert any(
            v.rule == "branch_delete_remote" and v.severity == "block" for v in violations
        )

    def test_branch_delete_colon_syntax(self):
        """git push origin :branch should be blocked."""
        cmd = "git push origin :my-branch"
        violations = check_command(cmd)
        assert any(
            v.rule == "branch_delete_remote_colon" and v.severity == "block"
            for v in violations
        )

    # -- Warning patterns --------------------------------------------------

    def test_rebase_warns(self):
        """git rebase should generate a warning."""
        cmd = "git rebase main"
        violations = check_command(cmd)
        assert any(v.rule == "rebase" and v.severity == "warn" for v in violations)

    def test_force_with_lease_warns(self):
        """git push --force-with-lease should generate a warning."""
        cmd = "git push origin my-branch --force-with-lease"
        violations = check_command(cmd)
        assert any(
            v.rule == "force_with_lease" and v.severity == "warn" for v in violations
        )

    def test_amend_commit_warns(self):
        """git commit --amend should generate a warning."""
        cmd = 'git commit --amend -m "updated message"'
        violations = check_command(cmd)
        assert any(v.rule == "amend_commit" and v.severity == "warn" for v in violations)

    # -- Safe commands (no violations) -------------------------------------

    def test_safe_push(self):
        """Normal push to a feature branch should have no blocking violations."""
        cmd = "git push -u origin feature/my-work"
        assert not has_blocking_violation(cmd)

    def test_safe_commit(self):
        """Normal commit should have no violations."""
        cmd = 'git commit -m "add feature"'
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_safe_add(self):
        """git add should have no violations."""
        cmd = "git add ."
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_safe_status(self):
        """git status should have no violations."""
        cmd = "git status"
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_safe_diff(self):
        """git diff should have no violations."""
        cmd = "git diff HEAD~1"
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_safe_log(self):
        """git log should have no violations."""
        cmd = "git log --oneline -10"
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_safe_gh_pr_create(self):
        """gh pr create should have no violations."""
        cmd = 'gh pr create --title "my PR" --body "description"'
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_non_git_command(self):
        """Non-git commands should have no violations."""
        cmd = "ls -la /workspace/repos"
        violations = check_command(cmd)
        assert len(violations) == 0

    def test_empty_command(self):
        """Empty command should return no violations."""
        violations = check_command("")
        assert len(violations) == 0

    def test_none_command(self):
        """None command should return no violations."""
        violations = check_command(None)
        assert len(violations) == 0

    def test_whitespace_command(self):
        """Whitespace-only command should return no violations."""
        violations = check_command("   ")
        assert len(violations) == 0


class TestHasBlockingViolation:
    """Tests for has_blocking_violation function."""

    def test_blocking_command_returns_true(self):
        assert has_blocking_violation("git push --force origin main") is True

    def test_warning_only_returns_false(self):
        assert has_blocking_violation("git rebase main") is False

    def test_safe_command_returns_false(self):
        assert has_blocking_violation("git status") is False


class TestFormatViolations:
    """Tests for format_violations function."""

    def test_empty_list(self):
        assert format_violations([]) == ""

    def test_single_violation(self):
        violations = [
            GitGuardrailViolation(
                rule="force_push",
                severity="block",
                command="git push --force",
                explanation="Force pushing is dangerous",
            )
        ]
        result = format_violations(violations)
        assert "BLOCKED" in result
        assert "force_push" in result
        assert "Force pushing is dangerous" in result

    def test_mixed_severities(self):
        violations = [
            GitGuardrailViolation(
                rule="force_push",
                severity="block",
                command="test",
                explanation="blocked reason",
            ),
            GitGuardrailViolation(
                rule="rebase",
                severity="warn",
                command="test",
                explanation="warning reason",
            ),
        ]
        result = format_violations(violations)
        assert "BLOCKED" in result
        assert "WARNING" in result


class TestRedactTokensInCommand:
    """Tests for redact_tokens_in_command function."""

    def test_redact_github_pat(self):
        cmd = "curl -H 'Authorization: token ghp_Qo5uXxYzAbCdEfGhIjKlMnOpQrStUvWxYz12' https://api.github.com"
        result = redact_tokens_in_command(cmd)
        assert "ghp_" not in result
        assert "[REDACTED]" in result

    def test_redact_github_fine_grained_pat(self):
        cmd = "git remote set-url origin https://github_pat_abc123def456_abcdefghijklmnopqrstuvwxyz1234@github.com/user/repo"
        result = redact_tokens_in_command(cmd)
        assert "github_pat_" not in result
        assert "[REDACTED]" in result

    def test_redact_gitlab_token(self):
        cmd = "git clone https://oauth2:glpat-xxxxxxxxxxxxxxxxxxxx@gitlab.com/user/repo"
        result = redact_tokens_in_command(cmd)
        assert "glpat-" not in result
        assert "[REDACTED]" in result

    def test_redact_url_credentials(self):
        cmd = "git clone https://user:secret_token@github.com/user/repo"
        result = redact_tokens_in_command(cmd)
        assert "secret_token" not in result

    def test_no_tokens_unchanged(self):
        cmd = "git push -u origin my-branch"
        result = redact_tokens_in_command(cmd)
        assert result == cmd

    def test_empty_string(self):
        result = redact_tokens_in_command("")
        assert result == ""


class TestMultipleViolations:
    """Test commands that trigger multiple violations."""

    def test_force_push_to_main(self):
        """Force push to main should trigger both force_push and push_to_main."""
        cmd = "git push --force origin main"
        violations = check_command(cmd)
        rules = {v.rule for v in violations}
        assert "force_push" in rules
        assert "push_to_main" in rules

    def test_destructive_local_sequence(self):
        """Commands chained with && should each be checked."""
        # check_command operates on the full string, so patterns match
        cmd = "git reset --hard HEAD && git clean -fd"
        violations = check_command(cmd)
        rules = {v.rule for v in violations}
        assert "reset_hard" in rules
        assert "clean_force" in rules
