# Architecture Diagrams

Visual representations of the Ambient Code Platform's structure and behavior.

## Core Diagrams

### 1. System Overview

**Recommended for**: First-time learners, stakeholder presentations, deployment planning

High-level view of all major components and how they interact:

- User and frontend layers
- Backend API and Kubernetes control plane
- Execution and storage layers
- Data flow between components

**[View System Overview →](system-overview.md)**

### 2. Session Lifecycle

**Recommended for**: Understanding execution flow, troubleshooting, feature design

Detailed sequence showing how sessions progress from creation through completion:

- Session creation and validation
- Kubernetes job orchestration
- Pod initialization and execution
- Results capture and cleanup
- Error and timeout handling

**[View Session Lifecycle →](session-lifecycle.md)**

## Additional Diagrams

### Platform Architecture
Complete system diagram with all components and connections.
**File**: `platform-architecture.mmd`

### Component Structure
Relationship diagram showing how platform components interact.
**File**: `component-structure.mmd`

### Deployment Stack
Infrastructure topology for deployments.
**File**: `deployment-stack.mmd`

### UX Feature Workflow
Multi-agent workflow for Request For Enhancement features.
**[View UX Feature Workflow →](ux-feature-workflow.md)**

## Using These Diagrams

### For Understanding
1. Start with **System Overview** to understand component relationships
2. Review **Session Lifecycle** to see execution flow
3. Consult specific diagrams for component interactions

### For Debugging
- **Session Lifecycle**: Trace where an execution failed
- **Component Structure**: Verify relationships between components
- **Deployment Stack**: Check infrastructure configuration

### For Development
- **Session Lifecycle**: Reference when implementing new session features
- **System Overview**: Verify placement of new components
- **Platform Architecture**: Understand integration points

### For Documentation
- Reference these diagrams in ADRs and design documents
- Link to specific diagrams when explaining features
- Include in runbooks and troubleshooting guides

## Diagram Conventions

### Styling
- **Neutral/Grayscale Theme**: High contrast for accessibility and printing
- **Component Grouping**: Related components in subgraphs with labels
- **Flow Direction**: Top-to-bottom (TB) for readability
- **Node Types**: Rectangles for components, circles for start/end states

### Symbols
- 👤 User interactions
- 🌐 Web components
- ⚡ API endpoints
- ☸️ Kubernetes components
- 🏃 Execution/runtime
- 📄 Data/state
- 🤖 AI/intelligent components

### Color Coding
- **White/Light Gray**: Frontend and API layers
- **Gray**: Kubernetes and control components
- **Very Light Gray**: Execution components
- **Lightest Gray**: Storage components
- **Green**: Start states
- **Light Red**: Error states

## Maintaining These Diagrams

### Before Release
- Run `mermaid-lint` workflow to validate syntax
- Verify all referenced diagrams exist
- Update diagrams when architecture changes
- Keep documentation in sync with diagrams

### Making Changes
1. Update the Mermaid file (`.mmd`)
2. Update corresponding documentation (`.md`)
3. Run linting to validate syntax
4. Create PR with clear description of changes
5. Request review from architecture maintainers

### Validation
Diagrams are automatically validated via GitHub Actions. The `mermaid-lint` workflow:

- Checks Mermaid syntax validity
- Verifies diagram references in documentation
- Runs on PR and push to main

## Related Documentation

- [Architecture Overview](../README.md)
- [Architectural Decision Records](../adr/)
- [Component Documentation](../../../components/)
- [Deployment Guides](../../deployment/)
