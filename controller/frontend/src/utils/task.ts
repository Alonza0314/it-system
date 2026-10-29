// Pipeline steps the runner adds around the selected testcases
export const PIPELINE_STEPS = ['prepare_free5gc', 'fetch_prs', 'make_nf', 'cleanup']

export function isPipelineStep(name: string): boolean {
  return PIPELINE_STEPS.includes(name)
}
