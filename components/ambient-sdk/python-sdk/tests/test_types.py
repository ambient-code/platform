import pytest
from datetime import datetime, timezone

from ambient_platform import (
    Session,
    SessionBuilder,
    SessionList,
    SessionPatch,
    Agent,
    AgentBuilder,
    Task,
    Skill,
    User,
    UserBuilder,
    Workflow,
    WorkflowBuilder,
    WorkflowSkill,
    WorkflowSkillBuilder,
    WorkflowTask,
    WorkflowTaskBuilder,
    Project,
    ProjectBuilder,
    ProjectPatch,
    ProjectSettings,
    ProjectSettingsBuilder,
    Permission,
    PermissionBuilder,
    PermissionList,
    PermissionPatch,
    RepositoryRef,
    RepositoryRefBuilder,
    RepositoryRefList,
    RepositoryRefPatch,
    ProjectKey,
    ProjectKeyBuilder,
    ProjectKeyList,
    ProjectKeyPatch,
)
from ambient_platform._base import ListOptions, APIError, ObjectReference, _parse_datetime


class TestSessionBuilder:
    def test_valid_session(self):
        data = (
            Session.builder()
            .name("test-session")
            .prompt("analyze this")
            .repo_url("https://github.com/foo/bar")
            .workflow_id("wf-123")
            .assigned_user_id("user-1")
            .build()
        )
        assert data["name"] == "test-session"
        assert data["prompt"] == "analyze this"
        assert data["repo_url"] == "https://github.com/foo/bar"
        assert data["workflow_id"] == "wf-123"

    def test_missing_name_raises(self):
        with pytest.raises(ValueError, match="name is required"):
            Session.builder().prompt("test").build()


class TestSessionBuilderWP4:
    def test_all_writable_fields(self):
        data = (
            Session.builder()
            .name("wp4-session")
            .prompt("test prompt")
            .interactive(True)
            .llm_model("claude-4-opus")
            .llm_temperature(0.7)
            .llm_max_tokens(4096)
            .repos('[{"url":"https://github.com/org/repo"}]')
            .labels("env=dev,team=platform")
            .annotations("note=wp4-test")
            .project_id("proj-1")
            .parent_session_id("parent-123")
            .bot_account_name("bot-1")
            .resource_overrides('{"cpu":"2","memory":"4Gi"}')
            .environment_variables('{"DEBUG":"true"}')
            .timeout(3600)
            .build()
        )
        assert data["interactive"] is True
        assert data["llm_temperature"] == 0.7
        assert data["llm_max_tokens"] == 4096
        assert data["llm_model"] == "claude-4-opus"
        assert data["timeout"] == 3600
        assert data["project_id"] == "proj-1"
        assert data["bot_account_name"] == "bot-1"

    def test_readonly_fields_not_on_builder(self):
        builder = Session.builder()
        for readonly_field in [
            "phase",
            "kube_cr_name",
            "kube_cr_uid",
            "kube_namespace",
            "completion_time",
            "start_time",
            "sdk_restart_count",
            "sdk_session_id",
            "conditions",
            "reconciled_repos",
            "reconciled_workflow",
        ]:
            assert not hasattr(builder, readonly_field), (
                f"Builder should NOT have readOnly method: {readonly_field}"
            )

    def test_writable_fields_present_on_builder(self):
        builder = Session.builder()
        for writable_field in [
            "name",
            "prompt",
            "interactive",
            "llm_model",
            "llm_temperature",
            "llm_max_tokens",
            "repos",
            "labels",
            "annotations",
            "project_id",
            "parent_session_id",
            "bot_account_name",
            "resource_overrides",
            "environment_variables",
            "timeout",
            "workflow_id",
            "repo_url",
            "assigned_user_id",
        ]:
            assert hasattr(builder, writable_field), (
                f"Builder should have writable method: {writable_field}"
            )


class TestSessionFromDictWP4:
    def test_float_field(self):
        s = Session.from_dict({"name": "t", "llm_temperature": 0.85})
        assert s.llm_temperature == 0.85

    def test_bool_field(self):
        s = Session.from_dict({"name": "t", "interactive": True})
        assert s.interactive is True

    def test_bool_default(self):
        s = Session.from_dict({"name": "t"})
        assert s.interactive is False

    def test_float_default(self):
        s = Session.from_dict({"name": "t"})
        assert s.llm_temperature == 0.0

    def test_int_field(self):
        s = Session.from_dict({"name": "t", "llm_max_tokens": 8192})
        assert s.llm_max_tokens == 8192

    def test_readonly_fields_deserialized(self):
        s = Session.from_dict({
            "name": "t",
            "phase": "running",
            "kube_cr_name": "cr-123",
            "kube_cr_uid": "uid-456",
            "kube_namespace": "ambient-code",
            "conditions": "Ready",
            "reconciled_repos": '["repo1"]',
            "reconciled_workflow": "wf-done",
            "sdk_restart_count": 3,
            "sdk_session_id": "sdk-xyz",
            "start_time": "2026-01-15T10:00:00Z",
            "completion_time": "2026-01-15T11:00:00Z",
        })
        assert s.phase == "running"
        assert s.kube_cr_name == "cr-123"
        assert s.kube_namespace == "ambient-code"
        assert s.sdk_restart_count == 3
        assert s.start_time is not None
        assert s.completion_time is not None

    def test_full_wp4_session(self):
        s = Session.from_dict({
            "id": "sess-wp4",
            "kind": "Session",
            "name": "full-session",
            "prompt": "analyze code",
            "interactive": True,
            "llm_model": "claude-4-opus",
            "llm_temperature": 0.7,
            "llm_max_tokens": 4096,
            "timeout": 3600,
            "project_id": "proj-1",
            "phase": "completed",
            "labels": "env=dev",
            "repos": '[{"url":"repo"}]',
            "bot_account_name": "bot-1",
            "parent_session_id": "parent-sess",
        })
        assert s.interactive is True
        assert s.llm_temperature == 0.7
        assert s.phase == "completed"
        assert s.bot_account_name == "bot-1"


class TestSessionPatchWP4:
    def test_patch_new_types(self):
        patch = (
            SessionPatch()
            .interactive(True)
            .llm_temperature(0.9)
            .llm_max_tokens(8192)
            .timeout(7200)
        )
        data = patch.to_dict()
        assert data["interactive"] is True
        assert data["llm_temperature"] == 0.9
        assert data["llm_max_tokens"] == 8192
        assert data["timeout"] == 7200

    def test_patch_readonly_fields_not_present(self):
        patch = SessionPatch()
        for readonly_field in [
            "phase",
            "kube_cr_name",
            "kube_cr_uid",
            "kube_namespace",
            "completion_time",
            "start_time",
            "sdk_restart_count",
            "sdk_session_id",
            "conditions",
            "reconciled_repos",
            "reconciled_workflow",
        ]:
            assert not hasattr(patch, readonly_field), (
                f"Patch should NOT have readOnly method: {readonly_field}"
            )


class TestSessionStatusPatchWP5:
    def test_all_fields(self):
        from ambient_platform import SessionStatusPatch

        patch = (
            SessionStatusPatch()
            .phase("Running")
            .sdk_session_id("sdk-123")
            .sdk_restart_count(2)
            .conditions('[{"type":"Ready","status":"True"}]')
            .kube_cr_uid("uid-abc")
            .kube_namespace("ambient-code")
            .reconciled_repos('["repo1","repo2"]')
            .reconciled_workflow('{"id":"wf-1"}')
        )
        data = patch.to_dict()
        assert data["phase"] == "Running"
        assert data["sdk_restart_count"] == 2
        assert data["kube_cr_uid"] == "uid-abc"
        assert data["kube_namespace"] == "ambient-code"
        assert len(data) == 8

    def test_sparse_update(self):
        from ambient_platform import SessionStatusPatch

        patch = SessionStatusPatch().phase("Completed")
        data = patch.to_dict()
        assert data == {"phase": "Completed"}
        assert "kube_namespace" not in data

    def test_datetime_fields(self):
        from ambient_platform import SessionStatusPatch
        from datetime import datetime, timezone

        now = datetime.now(tz=timezone.utc)
        patch = SessionStatusPatch().start_time(now).completion_time(now)
        data = patch.to_dict()
        assert data["start_time"] == now
        assert data["completion_time"] == now

    def test_has_all_10_methods(self):
        from ambient_platform import SessionStatusPatch

        patch = SessionStatusPatch()
        expected_methods = [
            "phase",
            "start_time",
            "completion_time",
            "sdk_session_id",
            "sdk_restart_count",
            "conditions",
            "reconciled_repos",
            "reconciled_workflow",
            "kube_cr_uid",
            "kube_namespace",
        ]
        for method in expected_methods:
            assert hasattr(patch, method), f"Missing method: {method}"


class TestAgentBuilder:
    def test_valid_with_project_id(self):
        data = Agent.builder().name("my-agent").project_id("proj-1").build()
        assert data["name"] == "my-agent"
        assert data["project_id"] == "proj-1"

    def test_missing_name_raises(self):
        with pytest.raises(ValueError, match="name is required"):
            Agent.builder().prompt("test").build()


class TestProjectBuilder:
    def test_valid_project(self):
        data = (
            Project.builder()
            .name("my-project")
            .display_name("My Project")
            .description("A test project")
            .labels("env=dev")
            .annotations("note=test")
            .status("active")
            .build()
        )
        assert data["name"] == "my-project"
        assert data["display_name"] == "My Project"
        assert data["description"] == "A test project"
        assert data["labels"] == "env=dev"
        assert data["status"] == "active"

    def test_missing_name_raises(self):
        with pytest.raises(ValueError, match="name is required"):
            Project.builder().description("no name").build()


class TestProjectSettingsBuilder:
    def test_valid(self):
        data = (
            ProjectSettings.builder()
            .project_id("proj-123")
            .group_access("admin,dev")
            .runner_secrets("secret-ref")
            .repositories("repo1,repo2")
            .build()
        )
        assert data["project_id"] == "proj-123"
        assert data["group_access"] == "admin,dev"

    def test_missing_project_id_raises(self):
        with pytest.raises(ValueError, match="project_id is required"):
            ProjectSettings.builder().group_access("admin").build()


class TestUserBuilder:
    def test_valid_with_groups(self):
        data = User.builder().name("Alice").username("alice").groups("admin,dev").build()
        assert data["name"] == "Alice"
        assert data["username"] == "alice"
        assert data["groups"] == "admin,dev"

    def test_missing_name_raises(self):
        with pytest.raises(ValueError, match="name is required"):
            User.builder().username("alice").build()

    def test_missing_username_raises(self):
        with pytest.raises(ValueError, match="username is required"):
            User.builder().name("Alice").build()


class TestWorkflowBuilder:
    def test_new_wp3_fields(self):
        data = (
            Workflow.builder()
            .name("ci-workflow")
            .project_id("proj-1")
            .branch("main")
            .path("/workflows/ci")
            .agent_id("agent-1")
            .build()
        )
        assert data["branch"] == "main"
        assert data["path"] == "/workflows/ci"
        assert data["project_id"] == "proj-1"
        assert data["agent_id"] == "agent-1"


class TestWorkflowSkillBuilder:
    def test_valid(self):
        data = (
            WorkflowSkill.builder()
            .workflow_id("wf-1")
            .skill_id("sk-1")
            .position(1)
            .build()
        )
        assert data["workflow_id"] == "wf-1"
        assert data["position"] == 1


class TestWorkflowTaskBuilder:
    def test_valid(self):
        data = (
            WorkflowTask.builder()
            .workflow_id("wf-1")
            .task_id("task-1")
            .position(2)
            .build()
        )
        assert data["task_id"] == "task-1"
        assert data["position"] == 2


class TestListOptions:
    def test_defaults(self):
        opts = ListOptions()
        params = opts.to_params()
        assert params["page"] == 1
        assert params["size"] == 100

    def test_max_size_capped(self):
        opts = ListOptions().size(999999)
        params = opts.to_params()
        assert params["size"] == 65500

    def test_all_fields(self):
        opts = (
            ListOptions()
            .page(3)
            .size(50)
            .search("name like 'test%'")
            .order_by("created_at desc")
            .fields("id,name,status")
        )
        params = opts.to_params()
        assert params["page"] == 3
        assert params["size"] == 50
        assert params["search"] == "name like 'test%'"
        assert params["orderBy"] == "created_at desc"
        assert params["fields"] == "id,name,status"


class TestPatchBuilder:
    def test_sets_only_specified_fields(self):
        patch = SessionPatch().prompt("updated prompt")
        data = patch.to_dict()
        assert data == {"prompt": "updated prompt"}
        assert "name" not in data

    def test_project_patch_all_fields(self):
        patch = (
            ProjectPatch()
            .name("renamed")
            .display_name("Renamed")
            .description("new desc")
            .labels("env=prod")
            .annotations("a=b")
            .status("archived")
        )
        data = patch.to_dict()
        assert len(data) == 6


class TestAPIError:
    def test_str_format(self):
        err = APIError(status_code=404, code="NOT_FOUND", reason="session not found")
        assert str(err) == "ambient API error 404: NOT_FOUND — session not found"

    def test_from_dict(self):
        err = APIError.from_dict(
            {"code": "VALIDATION", "reason": "bad input", "operation_id": "op-1"},
            status_code=400,
        )
        assert err.status_code == 400
        assert err.code == "VALIDATION"
        assert err.operation_id == "op-1"

    def test_is_exception(self):
        err = APIError(status_code=500, code="INTERNAL", reason="boom")
        assert isinstance(err, Exception)


class TestFromDict:
    def test_session_from_dict(self):
        data = {
            "id": "sess-123",
            "kind": "Session",
            "name": "test-session",
            "prompt": "analyze",
            "created_at": "2026-01-15T10:00:00Z",
        }
        s = Session.from_dict(data)
        assert s.id == "sess-123"
        assert s.name == "test-session"
        assert s.prompt == "analyze"
        assert s.created_at is not None
        assert s.created_at.year == 2026

    def test_project_from_dict(self):
        data = {
            "id": "proj-1",
            "kind": "Project",
            "name": "my-project",
            "display_name": "My Project",
            "status": "active",
        }
        p = Project.from_dict(data)
        assert p.display_name == "My Project"
        assert p.status == "active"

    def test_project_settings_from_dict(self):
        data = {
            "id": "ps-1",
            "project_id": "proj-1",
            "group_access": "admin",
            "runner_secrets": "ref",
        }
        ps = ProjectSettings.from_dict(data)
        assert ps.project_id == "proj-1"
        assert ps.group_access == "admin"

    def test_session_list_from_dict(self):
        data = {
            "kind": "SessionList",
            "page": 1,
            "size": 100,
            "total": 2,
            "items": [
                {"id": "s1", "name": "a"},
                {"id": "s2", "name": "b"},
            ],
        }
        sl = SessionList.from_dict(data)
        assert sl.total == 2
        assert sl.page == 1
        assert len(sl.items) == 2
        assert sl.items[0].name == "a"


class TestParseDatetime:
    def test_iso_with_z(self):
        dt = _parse_datetime("2026-01-15T10:00:00Z")
        assert dt is not None
        assert dt.year == 2026

    def test_iso_with_offset(self):
        dt = _parse_datetime("2026-01-15T10:00:00+00:00")
        assert dt is not None

    def test_none(self):
        assert _parse_datetime(None) is None

    def test_invalid_string(self):
        assert _parse_datetime("not-a-date") is None

    def test_datetime_passthrough(self):
        now = datetime.now(tz=timezone.utc)
        assert _parse_datetime(now) is now


class TestObjectReference:
    def test_from_dict(self):
        data = {
            "id": "ref-1",
            "kind": "Session",
            "href": "/v1/sessions/ref-1",
            "created_at": "2026-01-15T10:00:00Z",
        }
        ref = ObjectReference.from_dict(data)
        assert ref.id == "ref-1"
        assert ref.kind == "Session"
        assert ref.created_at is not None

    def test_frozen(self):
        ref = ObjectReference(id="test")
        with pytest.raises(AttributeError):
            ref.id = "changed"


class TestPermissionBuilder:
    def test_valid(self):
        data = (
            Permission.builder()
            .subject_type("user")
            .subject_name("alice")
            .role("admin")
            .project_id("proj-1")
            .build()
        )
        assert data["subject_type"] == "user"
        assert data["subject_name"] == "alice"
        assert data["role"] == "admin"
        assert data["project_id"] == "proj-1"

    def test_missing_role_raises(self):
        with pytest.raises(ValueError, match="role is required"):
            Permission.builder().subject_type("user").subject_name("alice").build()

    def test_missing_subject_name_raises(self):
        with pytest.raises(ValueError, match="subject_name is required"):
            Permission.builder().subject_type("user").role("admin").build()

    def test_missing_subject_type_raises(self):
        with pytest.raises(ValueError, match="subject_type is required"):
            Permission.builder().subject_name("alice").role("admin").build()


class TestPermissionFromDict:
    def test_permission_from_dict(self):
        data = {
            "id": "perm-1",
            "kind": "Permission",
            "subject_type": "user",
            "subject_name": "alice",
            "role": "admin",
            "project_id": "proj-1",
            "created_at": "2026-02-15T10:00:00Z",
        }
        p = Permission.from_dict(data)
        assert p.id == "perm-1"
        assert p.subject_type == "user"
        assert p.role == "admin"

    def test_permission_list_from_dict(self):
        data = {
            "kind": "PermissionList",
            "page": 1,
            "size": 100,
            "total": 1,
            "items": [{"id": "perm-1", "role": "admin"}],
        }
        pl = PermissionList.from_dict(data)
        assert pl.total == 1
        assert len(pl.items) == 1


class TestPermissionPatch:
    def test_sets_fields(self):
        patch = PermissionPatch().role("view").subject_name("bob")
        data = patch.to_dict()
        assert data["role"] == "view"
        assert data["subject_name"] == "bob"


class TestRepositoryRefBuilder:
    def test_valid(self):
        data = (
            RepositoryRef.builder()
            .name("my-repo")
            .url("https://github.com/org/repo")
            .branch("main")
            .provider("github")
            .build()
        )
        assert data["name"] == "my-repo"
        assert data["url"] == "https://github.com/org/repo"
        assert data["branch"] == "main"
        assert data["provider"] == "github"

    def test_missing_name_raises(self):
        with pytest.raises(ValueError, match="name is required"):
            RepositoryRef.builder().url("https://github.com/org/repo").build()

    def test_missing_url_raises(self):
        with pytest.raises(ValueError, match="url is required"):
            RepositoryRef.builder().name("my-repo").build()


class TestRepositoryRefFromDict:
    def test_repository_ref_from_dict(self):
        data = {
            "id": "ref-1",
            "kind": "RepositoryRef",
            "name": "my-repo",
            "url": "https://github.com/org/repo",
            "branch": "main",
            "provider": "github",
            "owner": "org",
            "repo_name": "repo",
        }
        r = RepositoryRef.from_dict(data)
        assert r.name == "my-repo"
        assert r.provider == "github"
        assert r.owner == "org"

    def test_repository_ref_list_from_dict(self):
        data = {
            "kind": "RepositoryRefList",
            "page": 1,
            "size": 100,
            "total": 2,
            "items": [
                {"id": "r1", "name": "repo-a"},
                {"id": "r2", "name": "repo-b"},
            ],
        }
        rl = RepositoryRefList.from_dict(data)
        assert rl.total == 2
        assert len(rl.items) == 2


class TestRepositoryRefPatch:
    def test_sets_fields(self):
        patch = RepositoryRefPatch().branch("develop").url("https://github.com/org/other")
        data = patch.to_dict()
        assert data["branch"] == "develop"
        assert data["url"] == "https://github.com/org/other"


class TestProjectKeyBuilder:
    def test_valid(self):
        data = ProjectKey.builder().name("my-api-key").project_id("proj-1").build()
        assert data["name"] == "my-api-key"
        assert data["project_id"] == "proj-1"

    def test_missing_name_raises(self):
        with pytest.raises(ValueError, match="name is required"):
            ProjectKey.builder().project_id("proj-1").build()


class TestProjectKeyFromDict:
    def test_project_key_from_dict(self):
        data = {
            "id": "pk-1",
            "kind": "ProjectKey",
            "name": "my-api-key",
            "key_prefix": "ak_12345",
            "plaintext_key": "ak_the-full-key",
            "project_id": "proj-1",
            "created_at": "2026-02-15T10:00:00Z",
            "expires_at": "2026-03-15T10:00:00Z",
        }
        pk = ProjectKey.from_dict(data)
        assert pk.id == "pk-1"
        assert pk.name == "my-api-key"
        assert pk.key_prefix == "ak_12345"
        assert pk.plaintext_key == "ak_the-full-key"
        assert pk.expires_at is not None

    def test_project_key_list_from_dict(self):
        data = {
            "kind": "ProjectKeyList",
            "page": 1,
            "size": 100,
            "total": 2,
            "items": [
                {"id": "pk1", "name": "key-a", "key_prefix": "ak_12345"},
                {"id": "pk2", "name": "key-b", "key_prefix": "ak_67890"},
            ],
        }
        pkl = ProjectKeyList.from_dict(data)
        assert pkl.total == 2
        assert len(pkl.items) == 2
        assert pkl.items[0].key_prefix == "ak_12345"


class TestProjectKeyPatch:
    def test_sets_fields(self):
        patch = ProjectKeyPatch().name("renamed-key")
        data = patch.to_dict()
        assert data["name"] == "renamed-key"
        assert len(data) == 1
