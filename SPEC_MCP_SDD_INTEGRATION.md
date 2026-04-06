# MCP and SDD Integration Specification for opencode-scaffold

## Overview
This specification outlines the integration of Model Context Protocol (MCP) and Spec-Driven Development (SDD) workflow capabilities into the opencode-scaffold CLI tool. The goal is to enhance the existing scaffolding functionality with standardized context management and structured development workflows.

## Goals
1. Add MCP as a first-class CLI command group for easy installation and configuration
2. Implement SDD workflow engine with per-phase model assignment
3. Enable agent-MCP interaction patterns for context sharing
4. Maintain backward compatibility with existing functionality
5. Follow the project's existing architectural patterns and conventions

## MCP Integration

### Command Structure
```
ocs mcp init              # Initialize MCP context in project
ocs mcp context           # Manage MCP contexts (create, list, show, delete)
ocs mcp server            # Run MCP server for external connections
ocs mcp tools             # List available MCP tools
ocs mcp call <tool>       # Call specific MCP tool with arguments
ocs mcp agents            # Configure MCP for specific agents
ocs mcp validate          # Validate MCP compliance and configuration
```

### MCP Client Implementation
- Create `internal/mcp/client.go` for MCP client functionality
- Support stdio and HTTP transports
- Implement Context7 integration for documentation access
- Provide methods for context storage/retrieval and tool invocation

### MCP Skill
- Create `skills/mcp/` directory with MCP management skill
- Include sub-skills for context management, tool discovery, and validation
- Follow existing skill structure patterns

### Configuration
- Extend `opencode.json` to include MCP configuration section
- Support MCP server definitions (stdio, HTTP, remote)
- Provide defaults for commonly used MCP servers (Context7, filesystem, etc.)

## SDD Workflow Enhancement

### SDD Orchestrator Agent
- Create `.opencode/agents/sdd-orchestrator.md` agent definition
- Implement delegation rules and phase coordination logic
- Include SDD workflow instructions and model assignments

### SDD Skills
Create skills for each SDD phase:
- `sdd-explore`: Investigate ideas and compare approaches
- `sdd-propose`: Create change proposals with intent and scope
- `sdd-spec`: Write detailed specifications with requirements
- `sdd-design`: Create technical design documents
- `sdd-tasks`: Break down work into implementation tasks
- `sdd-apply`: Implement tasks from specifications
- `sdd-verify`: Validate implementation against specs
- `sdd-archive`: Archive completed changes and sync to main specs
- `sdd-onboard`: Guided SDD walkthrough for new users
- `judgment-day`: Parallel adversarial review protocol

### Per-Phase Model Assignment
Implement model routing table:
| Phase | Default Model | Purpose |
|-------|---------------|---------|
| orchestrator | opus | Coordination and decision making |
| sdd-explore | sonnet | Code reading and structural analysis |
| sdd-propose | opus | Architectural decisions |
| sdd-spec | sonnet | Structured specification writing |
| sdd-design | opus | Architecture and design decisions |
| sdd-tasks | sonnet | Mechanical task breakdown |
| sdd-apply | sonnet | Implementation and coding |
| sdd-verify | sonnet | Validation and testing |
| sdd-archive | haiku | Archiving and cleanup |
| default | sonnet | General purpose delegation |

### Strict TDD Mode
- Automatic detection of testing frameworks
- Enforcement of test-first development when TDD mode is active
- Integration with existing test commands and verification

### Shared Prompt Files
- Create `.opencode/prompts/` directory for SDD phase prompts
- Implement `{file:<path>}` reference resolution for agent prompts
- Support agent coordination through file-based communication

## Implementation Plan

### Phase 1: Core Infrastructure (Weeks 1-2)
1. Add MCP command group to CLI (`ocs mcp ...`)
2. Implement MCP client with stdio and HTTP support
3. Create MCP skill for context management
4. Extend `opencode.json` schema for MCP configuration
5. Add basic MCP validation commands

### Phase 2: SDD Foundation (Weeks 2-3)
1. Create SDD orchestrator agent definition
2. Implement core SDD skills (explore, propose, spec)
3. Add per-phase model assignment capability
4. Create shared prompt file infrastructure
5. Implement Strict TDD mode detection

### Phase 3: Workflow Completion (Weeks 3-4)
1. Complete remaining SDD skills (design, tasks, apply, verify, archive)
2. Implement SDD continue/ff/new commands
3. Add judgment-day adversarial review skill
4. Create SDD onboarding experience
5. Add comprehensive testing and validation

### Phase 4: Integration and Polish (Week 4)
1. Integrate MCP context storage with SDD workflow
2. Enhance agent definitions to leverage MCP contexts
3. Add context versioning and diffing capabilities
4. Performance optimization and documentation
5. Backward compatibility verification

## Files to Create/Modify

### New Files:
- `cmd/mcp.go` - MCP command implementations
- `internal/mcp/client.go` - MCP client functionality
- `skills/mcp/` - MCP management skill
- `cmd/sdd*.go` - SDD command implementations (newinit, continue, ff, apply, verify, archive, onboard)
- `internal/sdd/` - SDD workflow engine
- `.opencode/agents/sdd-orchestrator.md` - Orchestrator agent definition
- `.opencode/skills/sdd-*/*` - SDD skill files
- `.opencode/prompts/*.md` - Shared prompt files for SDD phases

### Modified Files:
- `cmd/root.go` - Add MCP and SDD commands to root command
- `internal/engine/scaffold.go` - Extend scaffolding options for MCP/SDD
- `opencode.json` template - Add MCP configuration section
- `AGENTS.md` - Add MCP and SDD workflow instructions
- `main.go` - Potentially update version/build information

## Dependencies
- No new external dependencies required for core functionality
- May optionally use existing Context7 MCP server via npx
- Leverages existing Go standard library and project dependencies

## Backward Compatibility
- All existing commands and functionality remain unchanged
- MCP and SDD features are opt-in via flags or configuration
- Existing opencode.json configurations continue to work
- No breaking changes to current API or CLI interface

## Testing Strategy
- Unit tests for MCP client and server functionality
- Integration tests for MCP context operations
- End-to-end tests for SDD workflow execution
- Validation of backward compatibility with existing scaffolds
- Performance benchmarks for context operations

## Documentation
- Update README with MCP and SDD usage instructions
- Add command reference for new MCP and SDD commands
- Provide examples of MCP-enhanced workflows
- Document SDD phases and model assignments
- Include troubleshooting guides for common issues

## Success Criteria
1. `ocs mcp init` successfully configures MCP in a project
2. SDD workflow executes correctly with proper phase transitions
3. Agents can store and retrieve context via MCP
4. Per-phase model routing works as specified
5. Strict TDD mode activates appropriately when testing frameworks detected
6. All existing functionality continues to work unchanged
7. New features follow existing code patterns and conventions