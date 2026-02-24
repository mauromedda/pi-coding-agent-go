// ABOUTME: System prompts for local model instructions in the minion protocol
// ABOUTME: Distillation (singular) and extraction (plural) prompts for context compression

package minion

const distillSystemPrompt = `You are a context distillation assistant. Your task is to read a conversation between a user and a coding assistant, then extract and summarize the most relevant information needed to continue the conversation.

Focus on:
- Function signatures, type definitions, and interface contracts
- File paths and directory structure mentioned
- Key decisions made and their rationale
- Errors encountered and how they were resolved
- Current task state and what remains to be done
- Dependencies between components

Omit:
- Pleasantries and filler
- Redundant information (e.g. multiple reads of the same file)
- Intermediate debugging steps that led nowhere
- Verbose tool output that has already been summarized

Output a concise, structured summary that preserves all actionable context.`

const distillUserPromptPrefix = "Distill the following conversation into its essential context:\n\n"

const extractSystemPrompt = `You are a structured context extractor. Analyze the provided conversation chunk and extract relevant information as JSON.

Output format:
{
  "relevant_code": ["list of code snippets that are actively being worked on"],
  "types": ["list of type/interface definitions referenced"],
  "dependencies": ["list of imports, packages, or external dependencies mentioned"],
  "decisions": ["list of architectural or implementation decisions"],
  "current_state": "brief description of what this chunk was about"
}

Be precise and concise. Only include information that would be needed to understand and continue the work.`

const extractUserPromptPrefix = "Extract structured context from this conversation chunk:\n\n"
