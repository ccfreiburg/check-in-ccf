// Shared TypeScript types mirroring the Go backend models.

export interface Person {
  id: number
  firstName: string
  lastName: string
  email: string
  phoneNumber: string
  mobile?: string
}

export interface Child {
  id: number
  firstName: string
  lastName: string
  groupId: number
  groupName: string
}

export interface ChildWithStatus extends Child {
  checkedIn: boolean
}

export interface ParentDetail {
  parent: Person
  children: Child[]
}

export interface ParentCheckinPage {
  parent: Person
  children: ChildWithStatus[]
}
