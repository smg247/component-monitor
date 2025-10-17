export type Status = 'Healthy' | 'Degraded' | 'Down' | 'Suspected' | 'Partial'

export interface Outage {
  id: number
  component_name: string
  severity: string
  start_time: string
  end_time?: string
  description?: string
  discovered_by?: string
  created_by?: string
  resolved_by?: string
  confirmed_by?: string
  confirmed_at?: string
  triage_notes?: string
  auto_resolve: boolean
}

export interface ComponentStatus {
  component_name: string
  status: Status
  active_outages: Outage[]
}

export interface SubComponent {
  name: string
  description: string
  managed: boolean
  requires_confirmation: boolean
}

export interface Component {
  name: string
  description: string
  ship_team: string
  slack_channel: string
  sub_components: SubComponent[]
  owners: Array<{
    rover_group?: string
    service_account?: string
  }>
}

export interface SubComponentStatus {
  component_name: string
  status: Status
  active_outages: Outage[]
}
