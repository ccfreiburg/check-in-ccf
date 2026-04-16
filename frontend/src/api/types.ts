// Shared TypeScript types mirroring the Go backend models.

export interface Person {
  id: number
  firstName: string
  lastName: string
  birthdate?: string
  email: string
  phoneNumber: string
  mobile?: string
  sex?: string
  isGuest?: boolean
}

export interface Child {
  id: number
  firstName: string
  lastName: string
  birthdate?: string
  groupId: number
  groupName: string
  hasFather?: boolean
  hasMother?: boolean
}

export interface Parent {
  id: number
  firstName: string
  lastName: string
  sex: string
  email: string
  phoneNumber: string
  mobile?: string
  isGuest?: boolean
  groups: { id: number; name: string }[]
}

/** Status in the 2-step flow: '' = not registered today */
export type CheckInStatus = '' | 'pending' | 'checked_in'

export interface ChildWithStatus extends Child {
  status: CheckInStatus
  lastNotifiedAt?: string | null
}

export interface ParentDetail {
  parent: Person
  children: Child[]
}

export interface ParentCheckinPage {
  parent: Person
  children: ChildWithStatus[]
}

/** A local-DB check-in record returned by admin endpoints. */
export interface CheckInRecord {
  ID: number
  EventDate: string
  ChildID: number
  FirstName: string
  LastName: string
  Birthdate: string
  GroupID: number
  GroupName: string
  ParentID: number
  Status: CheckInStatus
  TagReceived: boolean
  RegisteredAt: string | null
  CheckedInAt: string | null
  CheckedOutAt: string | null
  LastNotifiedAt: string | null
  IsGuest: boolean
  CreatedAt: string
}

/** Metadata for a saved event report CSV file. */
export interface EventReport {
  filename: string
  date: string
  size: number
}

export interface GuestChildInput {
  firstName: string
  lastName: string
  birthdate: string
  groupId: number
  groupName: string
}

export interface GuestParentInput {
  firstName: string
  lastName: string
  sex: string
  mobile: string
}

export interface CreateGuestRequest {
  parent: GuestParentInput
  children: GuestChildInput[]
}

export type UpdateGuestRequest = CreateGuestRequest
