package rbac

type Resource string

const (
	ResourceUser            Resource = "user"
	ResourceProject         Resource = "project"
	ResourceProjectSettings Resource = "project_settings"
	ResourceProjectDocument Resource = "project_document"
	ResourceAgent           Resource = "agent"
	ResourceSession         Resource = "session"
	ResourceSessionMessage  Resource = "session_message"
	ResourceSessionCheckIn  Resource = "session_checkin"
	ResourceAgentMessage    Resource = "agent_message"
	ResourceBlackboard      Resource = "blackboard"
	ResourceRole            Resource = "role"
	ResourceRoleBinding     Resource = "role_binding"
)

type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionList    Action = "list"
	ActionWatch   Action = "watch"
	ActionIgnite  Action = "ignite"
	ActionCheckin Action = "checkin"
	ActionMessage Action = "message"
)

type Permission struct {
	Resource Resource
	Action   Action
}

func (p Permission) String() string {
	return string(p.Resource) + ":" + string(p.Action)
}

const (
	RolePlatformAdmin  = "platform:admin"
	RolePlatformViewer = "platform:viewer"

	RoleProjectOwner  = "project:owner"
	RoleProjectEditor = "project:editor"
	RoleProjectViewer = "project:viewer"

	RoleAgentOperator = "agent:operator"
	RoleAgentObserver = "agent:observer"
	RoleAgentRunner   = "agent:runner"
)

var (
	PermUserRead   = Permission{ResourceUser, ActionRead}
	PermUserList   = Permission{ResourceUser, ActionList}
	PermUserCreate = Permission{ResourceUser, ActionCreate}
	PermUserUpdate = Permission{ResourceUser, ActionUpdate}
	PermUserDelete = Permission{ResourceUser, ActionDelete}

	PermProjectCreate = Permission{ResourceProject, ActionCreate}
	PermProjectRead   = Permission{ResourceProject, ActionRead}
	PermProjectUpdate = Permission{ResourceProject, ActionUpdate}
	PermProjectDelete = Permission{ResourceProject, ActionDelete}
	PermProjectList   = Permission{ResourceProject, ActionList}

	PermProjectSettingsRead   = Permission{ResourceProjectSettings, ActionRead}
	PermProjectSettingsUpdate = Permission{ResourceProjectSettings, ActionUpdate}

	PermProjectDocumentRead   = Permission{ResourceProjectDocument, ActionRead}
	PermProjectDocumentCreate = Permission{ResourceProjectDocument, ActionCreate}
	PermProjectDocumentUpdate = Permission{ResourceProjectDocument, ActionUpdate}
	PermProjectDocumentDelete = Permission{ResourceProjectDocument, ActionDelete}
	PermProjectDocumentList   = Permission{ResourceProjectDocument, ActionList}

	PermAgentCreate = Permission{ResourceAgent, ActionCreate}
	PermAgentRead   = Permission{ResourceAgent, ActionRead}
	PermAgentUpdate = Permission{ResourceAgent, ActionUpdate}
	PermAgentDelete = Permission{ResourceAgent, ActionDelete}
	PermAgentList   = Permission{ResourceAgent, ActionList}
	PermAgentIgnite = Permission{ResourceAgent, ActionIgnite}

	PermSessionRead   = Permission{ResourceSession, ActionRead}
	PermSessionList   = Permission{ResourceSession, ActionList}
	PermSessionDelete = Permission{ResourceSession, ActionDelete}

	PermSessionMessageWatch = Permission{ResourceSessionMessage, ActionWatch}

	PermSessionCheckInCreate = Permission{ResourceSessionCheckIn, ActionCreate}
	PermSessionCheckInRead   = Permission{ResourceSessionCheckIn, ActionRead}
	PermSessionCheckInList   = Permission{ResourceSessionCheckIn, ActionList}

	PermAgentMessageSend   = Permission{ResourceAgentMessage, ActionMessage}
	PermAgentMessageRead   = Permission{ResourceAgentMessage, ActionRead}
	PermAgentMessageDelete = Permission{ResourceAgentMessage, ActionDelete}

	PermBlackboardWatch = Permission{ResourceBlackboard, ActionWatch}
	PermBlackboardRead  = Permission{ResourceBlackboard, ActionRead}

	PermRoleRead          = Permission{ResourceRole, ActionRead}
	PermRoleList          = Permission{ResourceRole, ActionList}
	PermRoleCreate        = Permission{ResourceRole, ActionCreate}
	PermRoleUpdate        = Permission{ResourceRole, ActionUpdate}
	PermRoleDelete        = Permission{ResourceRole, ActionDelete}
	PermRoleBindingRead   = Permission{ResourceRoleBinding, ActionRead}
	PermRoleBindingList   = Permission{ResourceRoleBinding, ActionList}
	PermRoleBindingCreate = Permission{ResourceRoleBinding, ActionCreate}
	PermRoleBindingDelete = Permission{ResourceRoleBinding, ActionDelete}
)
