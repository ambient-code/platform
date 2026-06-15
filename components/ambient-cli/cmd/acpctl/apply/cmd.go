// Package apply implements acpctl apply -f / -k for declarative fleet management.
package apply

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var Cmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply declarative Project, Agent, Credential, and RoleBinding manifests",
	Long: `Apply Projects, Agents, Credentials, and RoleBindings from YAML files or a Kustomize directory.

Mirrors kubectl apply semantics: resources are created if they do not exist,
or patched if they do. Output reports created / configured / unchanged per resource.

Supported kinds: Project, Agent, Credential, RoleBinding

File format (one or more documents separated by ---):

  kind: Project
  name: my-project
  description: "..."
  prompt: |
    Workspace context injected into every agent start.
  labels:
    env: dev
  annotations:
    ambient.io/summary: ""

  ---

  kind: Agent
  name: lead
  prompt: |
    You are the Lead...
  labels:
    ambient.io/ready: "true"
  annotations:
    work.ambient.io/current-task: ""
  inbox:
    - from_name: platform-bootstrap
      body: |
        First start. Bootstrap: read project annotations...

Examples:

  acpctl apply -f .ambient/teams/base/
  acpctl apply -f .ambient/teams/base/lead.yaml
  acpctl apply -k .ambient/teams/overlays/dev/
  acpctl apply -k .ambient/teams/overlays/prod/ --dry-run
  cat lead.yaml | acpctl apply -f -

Credential example:

  kind: Credential
  name: my-gitlab-pat
  provider: gitlab
  token: $GITLAB_PAT
  url: https://gitlab.myco.com
  labels:
    team: platform

RoleBinding example:

  kind: RoleBinding
  role: credential:token-reader
  scope: credential
  scope_id: my-gitlab-pat
  user_id: lead
`,
	RunE: run,
}

var applyArgs struct {
	file         string
	kustomize    string
	dryRun       bool
	outputFormat string
	project      string
}

func init() {
	Cmd.Flags().StringVarP(&applyArgs.file, "filename", "f", "", "File, directory, or - for stdin")
	Cmd.Flags().StringVarP(&applyArgs.kustomize, "kustomize", "k", "", "Kustomize directory")
	Cmd.Flags().BoolVar(&applyArgs.dryRun, "dry-run", false, "Print what would be applied without making API calls")
	Cmd.Flags().StringVarP(&applyArgs.outputFormat, "output", "o", "", "Output format: json")
	Cmd.Flags().StringVar(&applyArgs.project, "project", "", "Override project context for Agent resources")
}

// resource is a parsed YAML document from a manifest file.
type resource struct {
	Kind        string            `yaml:"kind"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Prompt      string            `yaml:"prompt"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
	Inbox       []inboxSeed       `yaml:"inbox"`
	Provider    string            `yaml:"provider"`
	Token       string            `yaml:"token"`
	URL         string            `yaml:"url"`
	Email       string            `yaml:"email"`
	Role        string            `yaml:"role"`
	Scope       string            `yaml:"scope"`
	ScopeID     string            `yaml:"scope_id"`
	UserID      string            `yaml:"user_id"`
}

type inboxSeed struct {
	FromName string `yaml:"from_name"`
	Body     string `yaml:"body"`
}

type applyResult struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func run(cmd *cobra.Command, _ []string) error {
	if applyArgs.file == "" && applyArgs.kustomize == "" {
		return fmt.Errorf("one of -f or -k is required")
	}
	if applyArgs.file != "" && applyArgs.kustomize != "" {
		return fmt.Errorf("-f and -k are mutually exclusive")
	}

	var docs []resource
	var err error

	if applyArgs.kustomize != "" {
		docs, err = loadKustomize(applyArgs.kustomize)
	} else {
		docs, err = loadFile(applyArgs.file)
	}
	if err != nil {
		return err
	}

	if applyArgs.dryRun {
		return printDryRun(cmd, docs)
	}

	factory, err := connection.NewClientFactory()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	projectName := applyArgs.project
	if projectName == "" {
		projectName = cfg.GetProject()
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
	defer cancel()

	client, err := factory.ForProject(projectName)
	if err != nil {
		return err
	}

	var results []applyResult
	for _, doc := range docs {
		var result applyResult
		switch strings.ToLower(doc.Kind) {
		case "project":
			result, err = applyProject(ctx, client, doc)
		case "agent":
			result, err = applyAgent(ctx, client, doc, projectName, factory)
		case "credential":
			result, err = applyCredential(ctx, client, doc)
		case "rolebinding":
			result, err = applyRoleBinding(ctx, client, doc)
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: unknown kind %q — skipping\n", doc.Kind)
			continue
		}
		if err != nil {
			return fmt.Errorf("apply %s/%s: %w", strings.ToLower(doc.Kind), docDisplayName(doc), err)
		}
		results = append(results, result)

		if applyArgs.outputFormat != "json" {
			displayName := result.Name
			if displayName == "" {
				displayName = result.Kind
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s/%s %s\n",
				strings.ToLower(result.Kind), displayName, result.Status)
		}
	}

	if applyArgs.outputFormat == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	return nil
}

func applyProject(ctx context.Context, client *sdkclient.Client, doc resource) (applyResult, error) {
	existing, err := client.Projects().Get(ctx, doc.Name)
	if err != nil {
		builder := sdktypes.NewProjectBuilder().Name(doc.Name)
		if doc.Description != "" {
			builder = builder.Description(doc.Description)
		}
		if doc.Prompt != "" {
			builder = builder.Prompt(doc.Prompt)
		}
		proj, buildErr := builder.Build()
		if buildErr != nil {
			return applyResult{}, buildErr
		}
		if _, createErr := client.Projects().Create(ctx, proj); createErr != nil {
			return applyResult{}, createErr
		}
		if len(doc.Labels) > 0 || len(doc.Annotations) > 0 {
			patch := map[string]any{}
			if len(doc.Labels) > 0 {
				patch["labels"] = marshalStringMap(doc.Labels)
			}
			if len(doc.Annotations) > 0 {
				patch["annotations"] = marshalStringMap(doc.Annotations)
			}
			if _, patchErr := client.Projects().Update(ctx, doc.Name, patch); patchErr != nil {
				return applyResult{}, patchErr
			}
		}
		return applyResult{Kind: "Project", Name: doc.Name, Status: "created"}, nil
	}

	patch := buildProjectPatch(existing, doc)
	if len(patch) == 0 {
		return applyResult{Kind: "Project", Name: doc.Name, Status: "unchanged"}, nil
	}
	if _, err = client.Projects().Update(ctx, doc.Name, patch); err != nil {
		return applyResult{}, err
	}
	return applyResult{Kind: "Project", Name: doc.Name, Status: "configured"}, nil
}

func applyCredential(ctx context.Context, client *sdkclient.Client, doc resource) (applyResult, error) {
	existing, err := client.Credentials().Get(ctx, doc.Name)
	if err != nil {
		token := os.ExpandEnv(doc.Token)
		builder := sdktypes.NewCredentialBuilder().
			Name(doc.Name).
			Provider(doc.Provider)
		if token != "" {
			builder = builder.Token(token)
		}
		if doc.Description != "" {
			builder = builder.Description(doc.Description)
		}
		if doc.URL != "" {
			builder = builder.URL(doc.URL)
		}
		if doc.Email != "" {
			builder = builder.Email(doc.Email)
		}
		if len(doc.Labels) > 0 {
			builder = builder.Labels(marshalStringMap(doc.Labels))
		}
		if len(doc.Annotations) > 0 {
			builder = builder.Annotations(marshalStringMap(doc.Annotations))
		}
		cred, buildErr := builder.Build()
		if buildErr != nil {
			return applyResult{}, buildErr
		}
		if _, createErr := client.Credentials().Create(ctx, cred); createErr != nil {
			return applyResult{}, createErr
		}
		return applyResult{Kind: "Credential", Name: doc.Name, Status: "created"}, nil
	}

	patch, changed := buildCredentialPatch(existing, doc)
	if !changed {
		return applyResult{Kind: "Credential", Name: doc.Name, Status: "unchanged"}, nil
	}
	if _, err = client.Credentials().Update(ctx, existing.ID, patch); err != nil {
		return applyResult{}, err
	}
	return applyResult{Kind: "Credential", Name: doc.Name, Status: "configured"}, nil
}

func buildCredentialPatch(existing *sdktypes.Credential, doc resource) (map[string]any, bool) {
	changed := false
	patch := sdktypes.NewCredentialPatchBuilder()
	if doc.Description != "" && doc.Description != existing.Description {
		patch = patch.Description(doc.Description)
		changed = true
	}
	if doc.URL != "" && doc.URL != existing.URL {
		patch = patch.URL(doc.URL)
		changed = true
	}
	if doc.Email != "" && doc.Email != existing.Email {
		patch = patch.Email(doc.Email)
		changed = true
	}
	token := os.ExpandEnv(doc.Token)
	if token != "" && token != existing.Token {
		patch = patch.Token(token)
		changed = true
	}
	if len(doc.Labels) > 0 && marshalStringMap(doc.Labels) != existing.Labels {
		patch = patch.Labels(marshalStringMap(doc.Labels))
		changed = true
	}
	if len(doc.Annotations) > 0 && marshalStringMap(doc.Annotations) != existing.Annotations {
		patch = patch.Annotations(marshalStringMap(doc.Annotations))
		changed = true
	}
	return patch.Build(), changed
}

// ── RoleBinding ──────────────────────────────────────────────────────────────

func applyRoleBinding(ctx context.Context, client *sdkclient.Client, doc resource) (applyResult, error) {
	displayName := roleBindingDisplayName(doc)

	if doc.Role == "" {
		return applyResult{}, fmt.Errorf("role is required")
	}
	if doc.Scope == "" {
		return applyResult{}, fmt.Errorf("scope is required")
	}
	if doc.ScopeID == "" {
		return applyResult{}, fmt.Errorf("scope_id is required")
	}
	if doc.UserID == "" {
		return applyResult{}, fmt.Errorf("user_id is required")
	}

	roleID, err := resolveRoleID(ctx, client, doc.Role)
	if err != nil {
		return applyResult{}, err
	}

	scopeFK, err := resolveScopeFK(ctx, client, doc.Scope, doc.ScopeID)
	if err != nil {
		return applyResult{}, err
	}

	opts := sdktypes.NewListOptions().Size(100).Build()
	existing, err := client.RoleBindings().List(ctx, opts)
	if err != nil {
		return applyResult{}, fmt.Errorf("list role-bindings: %w", err)
	}

	for _, rb := range existing.Items {
		if rb.RoleID == roleID &&
			rb.Scope == doc.Scope &&
			ptrEquals(rb.UserID, doc.UserID) &&
			scopeFKMatches(rb, doc.Scope, scopeFK) {
			return applyResult{Kind: "RoleBinding", Name: displayName, Status: "unchanged"}, nil
		}
	}

	builder := sdktypes.NewRoleBindingBuilder().
		RoleID(roleID).
		Scope(doc.Scope).
		UserID(doc.UserID)

	switch doc.Scope {
	case "credential":
		builder = builder.CredentialID(scopeFK)
	case "project":
		builder = builder.ProjectID(scopeFK)
	case "agent":
		builder = builder.AgentID(scopeFK)
	case "session":
		builder = builder.SessionID(scopeFK)
	}

	rb, buildErr := builder.Build()
	if buildErr != nil {
		return applyResult{}, buildErr
	}
	if _, createErr := client.RoleBindings().Create(ctx, rb); createErr != nil {
		return applyResult{}, fmt.Errorf("create role-binding: %w", createErr)
	}
	return applyResult{Kind: "RoleBinding", Name: displayName, Status: "created"}, nil
}

func resolveRoleID(ctx context.Context, client *sdkclient.Client, roleName string) (string, error) {
	opts := sdktypes.NewListOptions().Size(100).
		Search(fmt.Sprintf("name = '%s'", roleName)).Build()
	list, err := client.Roles().List(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("search roles for %q: %w", roleName, err)
	}
	for _, r := range list.Items {
		if r.Name == roleName {
			return r.ID, nil
		}
	}
	return "", fmt.Errorf("role %q not found", roleName)
}

func resolveScopeFK(ctx context.Context, client *sdkclient.Client, scope, scopeID string) (string, error) {
	switch scope {
	case "credential":
		return resolveCredentialID(ctx, client, scopeID)
	case "project":
		proj, err := client.Projects().Get(ctx, scopeID)
		if err != nil {
			return "", fmt.Errorf("resolve project %q: %w", scopeID, err)
		}
		return proj.ID, nil
	case "agent":
		return resolveAgentID(ctx, client, scopeID)
	case "session":
		return resolveSessionID(ctx, client, scopeID)
	default:
		return "", fmt.Errorf("unsupported scope %q", scope)
	}
}

func resolveCredentialID(ctx context.Context, client *sdkclient.Client, nameOrID string) (string, error) {
	cred, err := client.Credentials().Get(ctx, nameOrID)
	if err == nil {
		return cred.ID, nil
	}
	var apiErr *sdktypes.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		return "", fmt.Errorf("resolve credential %q: %w", nameOrID, err)
	}
	opts := sdktypes.NewListOptions().Size(100).
		Search(fmt.Sprintf("name = '%s'", nameOrID)).Build()
	list, err := client.Credentials().List(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("search credentials for %q: %w", nameOrID, err)
	}
	var matches []sdktypes.Credential
	for _, c := range list.Items {
		if c.Name == nameOrID {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("credential %q not found", nameOrID)
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	var ids []string
	for _, m := range matches {
		ids = append(ids, m.ID)
	}
	return "", fmt.Errorf("multiple credentials named %q found (%s); use the credential ID instead", nameOrID, strings.Join(ids, ", "))
}

func resolveAgentID(ctx context.Context, client *sdkclient.Client, nameOrID string) (string, error) {
	agent, err := client.Agents().Get(ctx, nameOrID)
	if err == nil {
		return agent.ID, nil
	}
	var apiErr *sdktypes.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		return "", fmt.Errorf("resolve agent %q: %w", nameOrID, err)
	}
	opts := sdktypes.NewListOptions().Size(100).
		Search(fmt.Sprintf("name = '%s'", nameOrID)).Build()
	list, err := client.Agents().List(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("search agents for %q: %w", nameOrID, err)
	}
	for _, a := range list.Items {
		if a.Name == nameOrID {
			return a.ID, nil
		}
	}
	return "", fmt.Errorf("agent %q not found", nameOrID)
}

func resolveSessionID(ctx context.Context, client *sdkclient.Client, nameOrID string) (string, error) {
	sess, err := client.Sessions().Get(ctx, nameOrID)
	if err == nil {
		return sess.ID, nil
	}
	var apiErr *sdktypes.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		return "", fmt.Errorf("resolve session %q: %w", nameOrID, err)
	}
	opts := sdktypes.NewListOptions().Size(100).
		Search(fmt.Sprintf("name = '%s'", nameOrID)).Build()
	list, err := client.Sessions().List(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("search sessions for %q: %w", nameOrID, err)
	}
	for _, s := range list.Items {
		if s.Name == nameOrID {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("session %q not found", nameOrID)
}

func scopeFKMatches(rb sdktypes.RoleBinding, scope, fk string) bool {
	switch scope {
	case "credential":
		return ptrEquals(rb.CredentialID, fk)
	case "project":
		return ptrEquals(rb.ProjectID, fk)
	case "agent":
		return ptrEquals(rb.AgentID, fk)
	case "session":
		return ptrEquals(rb.SessionID, fk)
	}
	return false
}

func ptrEquals(p *string, v string) bool {
	return p != nil && *p == v
}

func roleBindingDisplayName(doc resource) string {
	return doc.UserID + "\u2192" + doc.ScopeID
}

func docDisplayName(d resource) string {
	if d.Name != "" {
		return d.Name
	}
	if strings.EqualFold(d.Kind, "RoleBinding") {
		return roleBindingDisplayName(d)
	}
	return d.Kind
}

func marshalStringMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func buildProjectPatch(existing *sdktypes.Project, doc resource) map[string]any {
	patch := map[string]any{}
	if doc.Description != "" && doc.Description != existing.Description {
		patch["description"] = doc.Description
	}
	if doc.Prompt != "" && doc.Prompt != existing.Prompt {
		patch["prompt"] = doc.Prompt
	}
	if len(doc.Labels) > 0 {
		patch["labels"] = marshalStringMap(doc.Labels)
	}
	if len(doc.Annotations) > 0 {
		patch["annotations"] = marshalStringMap(doc.Annotations)
	}
	return patch
}

func applyAgent(ctx context.Context, client *sdkclient.Client, doc resource, projectName string, factory *connection.ClientFactory) (applyResult, error) {
	projClient := client
	if factory != nil {
		if pc, err := factory.ForProject(projectName); err == nil {
			projClient = pc
		}
	}

	project, err := projClient.Projects().Get(ctx, projectName)
	if err != nil {
		return applyResult{}, fmt.Errorf("project %q not found: %w", projectName, err)
	}

	existing, err := projClient.Agents().GetInProject(ctx, project.ID, doc.Name)
	if err != nil {
		builder := sdktypes.NewAgentBuilder().
			ProjectID(project.ID).
			Name(doc.Name)
		if doc.Prompt != "" {
			builder = builder.Prompt(doc.Prompt)
		}
		pa, buildErr := builder.Build()
		if buildErr != nil {
			return applyResult{}, buildErr
		}
		created, createErr := projClient.Agents().CreateInProject(ctx, project.ID, pa)
		if createErr != nil {
			return applyResult{}, createErr
		}
		if len(doc.Labels) > 0 || len(doc.Annotations) > 0 {
			patch := map[string]any{}
			if len(doc.Labels) > 0 {
				patch["labels"] = marshalStringMap(doc.Labels)
			}
			if len(doc.Annotations) > 0 {
				patch["annotations"] = marshalStringMap(doc.Annotations)
			}
			if _, patchErr := projClient.Agents().UpdateInProject(ctx, project.ID, created.ID, patch); patchErr != nil {
				return applyResult{}, patchErr
			}
		}
		if seedErr := seedInbox(ctx, projClient, project.ID, created.ID, doc.Inbox); seedErr != nil {
			return applyResult{}, seedErr
		}
		return applyResult{Kind: "Agent", Name: doc.Name, Status: "created"}, nil
	}

	patch := buildAgentPatch(existing, doc)
	status := "unchanged"
	if len(patch) > 0 {
		if _, err = projClient.Agents().UpdateInProject(ctx, project.ID, existing.ID, patch); err != nil {
			return applyResult{}, err
		}
		status = "configured"
	}
	if seedErr := seedInbox(ctx, projClient, project.ID, existing.ID, doc.Inbox); seedErr != nil {
		return applyResult{}, seedErr
	}
	return applyResult{Kind: "Agent", Name: doc.Name, Status: status}, nil
}

func buildAgentPatch(existing *sdktypes.Agent, doc resource) map[string]any {
	patch := map[string]any{}
	if doc.Prompt != "" && doc.Prompt != existing.Prompt {
		patch["prompt"] = doc.Prompt
	}
	if len(doc.Labels) > 0 {
		patch["labels"] = marshalStringMap(doc.Labels)
	}
	if len(doc.Annotations) > 0 {
		patch["annotations"] = marshalStringMap(doc.Annotations)
	}
	return patch
}

func seedInbox(ctx context.Context, client *sdkclient.Client, projectID, agentID string, seeds []inboxSeed) error {
	if len(seeds) == 0 {
		return nil
	}
	existing, err := client.Agents().ListInboxInProject(ctx, projectID, agentID)
	if err != nil {
		return nil
	}
	existingSet := make(map[string]bool, len(existing))
	for _, msg := range existing {
		existingSet[msg.FromName+"\x00"+msg.Body] = true
	}
	for _, seed := range seeds {
		key := seed.FromName + "\x00" + seed.Body
		if existingSet[key] {
			continue
		}
		if err := client.Agents().SendInboxInProject(ctx, projectID, agentID, seed.FromName, seed.Body); err != nil {
			return err
		}
	}
	return nil
}

// ── YAML loading ──────────────────────────────────────────────────────────────

func loadFile(path string) ([]resource, error) {
	if path == "-" {
		return parseManifests(os.Stdin)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return loadDir(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseManifests(f)
}

func loadDir(dir string) ([]resource, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var all []resource
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		if name == "kustomization.yaml" || name == "kustomization.yml" {
			continue
		}
		docs, err := loadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		all = append(all, docs...)
	}
	return all, nil
}

func parseManifests(r io.Reader) ([]resource, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var docs []resource
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var doc resource
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse YAML: %w", err)
		}
		if doc.Kind == "" {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// ── Kustomize ─────────────────────────────────────────────────────────────────

type kustomization struct {
	Kind      string      `yaml:"kind"`
	Resources []string    `yaml:"resources"`
	Bases     []string    `yaml:"bases"`
	Patches   []kustPatch `yaml:"patches"`
}

type kustPatch struct {
	Path   string     `yaml:"path"`
	Target kustTarget `yaml:"target"`
}

type kustTarget struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

func loadKustomize(dir string) ([]resource, error) {
	kustFile := ""
	for _, name := range []string{"kustomization.yaml", "kustomization.yml"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			kustFile = p
			break
		}
	}
	if kustFile == "" {
		return nil, fmt.Errorf("no kustomization.yaml found in %s", dir)
	}

	data, err := os.ReadFile(kustFile)
	if err != nil {
		return nil, err
	}
	var kust kustomization
	if err := yaml.Unmarshal(data, &kust); err != nil {
		return nil, fmt.Errorf("parse kustomization: %w", err)
	}

	var docs []resource

	for _, base := range kust.Bases {
		basePath := filepath.Join(dir, base)
		baseDocs, err := loadKustomize(basePath)
		if err != nil {
			return nil, fmt.Errorf("base %s: %w", base, err)
		}
		docs = append(docs, baseDocs...)
	}

	for _, res := range kust.Resources {
		resPath := filepath.Join(dir, res)
		info, err := os.Stat(resPath)
		if err != nil {
			return nil, fmt.Errorf("resource %s: %w", res, err)
		}
		var resDocs []resource
		if info.IsDir() {
			resDocs, err = loadKustomize(resPath)
		} else {
			resDocs, err = loadFile(resPath)
		}
		if err != nil {
			return nil, fmt.Errorf("resource %s: %w", res, err)
		}
		docs = mergeResources(docs, resDocs)
	}

	for _, patch := range kust.Patches {
		patchDocs, err := loadFile(filepath.Join(dir, patch.Path))
		if err != nil {
			return nil, fmt.Errorf("patch %s: %w", patch.Path, err)
		}
		for _, p := range patchDocs {
			docs = applyPatch(docs, p, patch.Target)
		}
	}

	return docs, nil
}

// mergeResources adds resDocs into docs, deduplicating by kind+name (later wins).
func mergeResources(docs, incoming []resource) []resource {
	idx := make(map[string]int, len(docs))
	for i, d := range docs {
		idx[resourceKey(d)] = i
	}
	for _, inc := range incoming {
		key := resourceKey(inc)
		if i, exists := idx[key]; exists {
			docs[i] = inc
		} else {
			idx[key] = len(docs)
			docs = append(docs, inc)
		}
	}
	return docs
}

func resourceKey(r resource) string {
	return strings.ToLower(r.Kind) + "/" + r.Name
}

// applyPatch merges patch into all matching resources (strategic merge).
func applyPatch(docs []resource, patch resource, target kustTarget) []resource {
	for i := range docs {
		if !matchesTarget(docs[i], target) {
			continue
		}
		docs[i] = strategicMerge(docs[i], patch)
	}
	return docs
}

func matchesTarget(doc resource, target kustTarget) bool {
	if target.Kind != "" && !strings.EqualFold(doc.Kind, target.Kind) {
		return false
	}
	if target.Name != "" && doc.Name != target.Name {
		return false
	}
	return true
}

// strategicMerge applies patch onto base: scalars overwrite, maps merge.
func strategicMerge(base, patch resource) resource {
	if patch.Name != "" {
		base.Name = patch.Name
	}
	if patch.Description != "" {
		base.Description = patch.Description
	}
	if patch.Prompt != "" {
		base.Prompt = patch.Prompt
	}
	if patch.Provider != "" {
		base.Provider = patch.Provider
	}
	if patch.Token != "" {
		base.Token = patch.Token
	}
	if patch.URL != "" {
		base.URL = patch.URL
	}
	if patch.Email != "" {
		base.Email = patch.Email
	}
	for k, v := range patch.Labels {
		if base.Labels == nil {
			base.Labels = make(map[string]string)
		}
		base.Labels[k] = v
	}
	for k, v := range patch.Annotations {
		if base.Annotations == nil {
			base.Annotations = make(map[string]string)
		}
		base.Annotations[k] = v
	}
	return base
}

// ── Dry-run ───────────────────────────────────────────────────────────────────

func printDryRun(cmd *cobra.Command, docs []resource) error {
	if applyArgs.outputFormat == "json" {
		results := make([]applyResult, 0, len(docs))
		for _, d := range docs {
			results = append(results, applyResult{Kind: d.Kind, Name: docDisplayName(d), Status: "dry-run"})
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "dry-run: would apply:")
	for _, d := range docs {
		fmt.Fprintf(w, "  %s/%s\n", strings.ToLower(d.Kind), docDisplayName(d))
	}
	return nil
}

// ── stdin helper ──────────────────────────────────────────────────────────────

func readStdin() ([]byte, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
	}
	return buf.Bytes(), scanner.Err()
}

var _ = readStdin
