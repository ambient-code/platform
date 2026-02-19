package repositoryRefs

import (
	"net/url"
	"strings"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type RepositoryRef struct {
	api.Meta
	Name      string  `json:"name"`
	Url       string  `json:"url"`
	Branch    *string `json:"branch"`
	Provider  *string `json:"provider"`
	Owner     *string `json:"owner"`
	RepoName  *string `json:"repo_name"`
	ProjectId *string `json:"project_id"`
}

type RepositoryRefList []*RepositoryRef
type RepositoryRefIndex map[string]*RepositoryRef

func (l RepositoryRefList) Index() RepositoryRefIndex {
	index := RepositoryRefIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func detectProvider(repoURL string) string {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	switch {
	case strings.Contains(host, "github"):
		return "github"
	case strings.Contains(host, "gitlab"):
		return "gitlab"
	default:
		return ""
	}
}

func parseOwnerRepo(repoURL string) (string, string) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", ""
	}
	path := strings.Trim(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func (d *RepositoryRef) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()

	if d.Provider == nil || *d.Provider == "" {
		if p := detectProvider(d.Url); p != "" {
			d.Provider = &p
		}
	}

	if (d.Owner == nil || *d.Owner == "") || (d.RepoName == nil || *d.RepoName == "") {
		owner, repo := parseOwnerRepo(d.Url)
		if owner != "" {
			d.Owner = &owner
		}
		if repo != "" {
			d.RepoName = &repo
		}
	}

	return nil
}

type RepositoryRefPatchRequest struct {
	Name      *string `json:"name,omitempty"`
	Url       *string `json:"url,omitempty"`
	Branch    *string `json:"branch,omitempty"`
	Provider  *string `json:"provider,omitempty"`
	Owner     *string `json:"owner,omitempty"`
	RepoName  *string `json:"repo_name,omitempty"`
	ProjectId *string `json:"project_id,omitempty"`
}
