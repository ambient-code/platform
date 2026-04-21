package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *ProjectAPI) PatchLabels(ctx context.Context, id string, labels map[string]string) (*types.Project, error) {
	b, err := json.Marshal(labels)
	if err != nil {
		return nil, fmt.Errorf("marshal labels: %w", err)
	}
	return a.Update(ctx, id, map[string]any{"labels": string(b)})
}

func (a *ProjectAPI) PatchAnnotations(ctx context.Context, id string, annotations map[string]string) (*types.Project, error) {
	b, err := json.Marshal(annotations)
	if err != nil {
		return nil, fmt.Errorf("marshal annotations: %w", err)
	}
	return a.Update(ctx, id, map[string]any{"annotations": string(b)})
}
