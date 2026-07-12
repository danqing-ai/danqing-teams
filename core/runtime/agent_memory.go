// Package runtime provides the agent memory implementation.
//
// TODO(nil): Episodic memory and Notepad (inter-agent wisdom bus) are temporarily removed.
//
// Original design (removed because of near-zero recall rate and redundant injection):
//   1. Episodic (agent-specific):
//      - WriteEpisodic: afterTurn() writes {agentID, turnSummary}.
//      - Recall: strings.Contains keyword matching against entries for the agent.
//      - Injected into system prompt as "Agent memory" section.
//      - Near-zero hit rate — naive substring match against goal text.
//   2. Notepad (inter-agent wisdom bus):
//      - AppendNotepad: delegate_agent appends "[agentName] summary" on subagent completion.
//      - NotepadList: injected as "Mission wisdom" into every subsequent system prompt.
//      - Redundant — subagent results are already in tool result messages.
//      - No size cap — unbounded growth with delegation count.
//
// Future design — compaction-time BM25 indexing:
//
// Rather than indexing per-turn summaries, index the full conversation as it is
// about to be dropped by compaction. This captures all messages (tool calls,
// tool results, LLM responses) — not just summaries — and ensures zero information
// loss, even for content that was never summarized.
//
//   - Record: at compaction time (maybeCompact), before messages[0:cutIdx] are
//     discarded, index them into a session-scoped BM25 index. Each message is
//     a document with fields: role, content (text), tool_call_id, turn.
//   - Recall: before each turn, BM25-query the index with the current goal.
//     Return top-K message chunks as "Agent memory" in the system prompt.
//   - Index pruning: when a session ends, discard the index (pure in-memory).
//     For long-running sessions, evict oldest documents by token budget.
//   - The buildSystemPrompt path and config already have a memory []string slot
//     ready for injection.
//

package runtime
