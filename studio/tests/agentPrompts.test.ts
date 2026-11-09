import { describe, expect, test } from 'bun:test'
import { agentPromptTargets, buildAgentPrompt } from '../app/utils/agentPrompts'

const target = (key: string) => agentPromptTargets.find(t => t.key === key)!
const project = { id: 49, name: 'vibeasik (repo baru)' }

describe('buildAgentPrompt', () => {
  test('Claude Code memakai mode divisi dengan sub agent', () => {
    const p = buildAgentPrompt(target('claude'), project)
    expect(p).toContain('vibeasik_todo_dispatch_plan` {project_id: 49, coordinator: "claude"}')
    expect(p).toContain('jangan dipakai ulang')
    expect(p).not.toContain('vibeasik_dispatch_agent')
  })

  test('Antigravity memakai mode worker dan berhenti bila dispatch dimatikan', () => {
    const p = buildAgentPrompt(target('agy'), project)
    expect(p).toContain('coordinator: "agy"')
    expect(p).toContain('vibeasik_dispatch_agent')
    expect(p).toContain('delegate/workers diabaikan')
    expect(p).toContain('percakapan baru per todo')
  })

  test('semua prompt melarang scan dan menyebut proyek', () => {
    for (const t of agentPromptTargets) {
      const p = buildAgentPrompt(t, project)
      expect(p).toContain('project_id 49')
      expect(p).toContain('jangan scan')
      expect(p).toContain('"vibeasik (repo baru)"')
      expect(p).toContain('Hemat konteks')
    }
  })
})
