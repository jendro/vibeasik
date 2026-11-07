import type { Finding, ResolutionResponse } from '~/types/diagnostics'

// Salin hasil penyelesaian (LLM / mesin heuristik) ke temuan di tampilan — satu tempat untuk resolve satu & batch.
// Hasil yang hanya digabung ke todo yang sudah ada tidak membawa rekomendasi baru, jadi rekomendasi lama temuan
// tidak ditimpa (selaras ResolutionResponse.ApplyTo di local-engine).
export const applyResolutionToFinding = (finding: Finding, res: ResolutionResponse) => {
  finding.status = 'resolved'
  finding.resolution_title = res.title
  finding.model_used = res.model_used
  if (res.created_todo) {
    finding.todo_id = res.created_todo.id
  }
  if (res.generated_via === 'audit_merge' || res.generated_via === 'reject_guard') {
    if (!finding.explanation) {
      finding.explanation = res.explanation
    }
    return
  }
  finding.explanation = res.explanation
  finding.action_steps = res.action_steps
  finding.code_before = res.code_before
  finding.code_after = res.code_after
  finding.target_lines = res.target_lines
  finding.placement_plan = res.placement_plan
  if (res.theory_basis) {
    finding.theory_basis = res.theory_basis
  }
}
