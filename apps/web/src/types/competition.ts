export type CompetitionStatus = 'draft' | 'active' | 'paused' | 'completed' | 'cancelled'

export interface Competition {
  id: string
  name: string
  location: string
  start_date: string
  end_date: string
  ruleset_id?: string
  ruleset_name: string
  status: CompetitionStatus
  created_at: string
}

export interface CreateCompetitionPayload {
  name: string
  location: string
  start_date: string
  end_date: string
  ruleset_name: string
  ruleset_id?: string
}

export interface UpdateCompetitionPayload {
  name?: string
  location?: string
  start_date?: string
  end_date?: string
  ruleset_name?: string
  ruleset_id?: string
}

export interface CompetitionListQuery {
  page?: number
  limit?: number
  status?: CompetitionStatus | string
}

export interface CompetitionPagination {
  page: number
  limit: number
  total: number
}

export interface CompetitionListResult {
  data: Competition[]
  pagination: CompetitionPagination
}
