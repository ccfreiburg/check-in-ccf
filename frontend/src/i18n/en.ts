export default {
  // ── Glossar ──────────────────────────────────────────────────────────────
  status: {
    pending: 'Registered',
    checked_in: 'Checked In',
    not_registered: 'Not registered yet',
  },

  // ── Common UI ────────────────────────────────────────────────────────────
  common: {
    loading: 'Loading\u2026',
    please_wait: 'Please wait\u2026',
    back: 'Back',
    error: 'Error',
    not_found: 'Not found.',
    close: '\u00d7',
    ccf_alt: 'CCF',
  },

  // ── Navigation ───────────────────────────────────────────────────────────
  nav: {
    menu_label: 'Menu',
    menu_heading: 'Menu',
    logout: 'Sign out',
    first_registration: 'First Registration',
    name_tag_handout: 'Name Tag Handout',
    children_today: 'Children Today',
    dashboard: 'Dashboard',
    admin: 'Admin Mode',
    lang_switch: 'Switch language',
    lang_modal_heading: 'Select language',
  },

  // ── Login ────────────────────────────────────────────────────────────────
  login: {
    title: 'Admin Login',    username_placeholder: 'Email (ChurchTools)',    password_placeholder: 'Password',
    signing_in: 'Signing in\u2026',
    sign_in: 'Sign in',
    error_fallback: 'Login failed',
  },

  // ── Parent App ───────────────────────────────────────────────────────────
  parent: {
    title: 'Child Registration',
    greeting: 'Hello {firstName} {lastName}',
    subtitle: 'Register your children for today\'s service here.',
    link_expired: 'This link may have expired. Please ask staff for a new QR code.',
    qr_button_title: 'QR Code',
    qr_alt: 'QR Code',
    share_qr: 'Share QR Code',
    share_title: 'Child Registration',
    share_text: 'Registration link for {firstName} {lastName}',
    link_copied: 'Link copied to clipboard \u2713',
    link_fallback: 'Link: {url}',
    install_app: '📲 Install App',
    ios_push_heading: '📲 For notifications on iPhone:',
    ios_push_steps: 'Tap Share → "Add to Home Screen" → open the app.',
    enable_push: '🔔 Enable notifications',
    push_granted: '🔔 Notifications enabled \u2713',
    push_denied: 'Notifications were blocked. Please enable them in browser settings.',
    push_error: 'Error: {error}',
    no_children: 'No children on file. Please speak to a volunteer.',
    registered: '{firstName} has been registered \u2713',
    register_error: 'Error during registration',
  },

  // ── First Registration ───────────────────────────────────────────────────
  first_registration: {
    title: 'First Registration',
    search_placeholder: 'Search by name\u2026',
    filter: 'Filter',
    filter_clear_title: 'Clear filters',
    fathers: 'Fathers',
    mothers: 'Mothers',
    no_sex_data: 'No sex data in database — please run CT Sync.',
    no_parents: 'No parents found',
    no_children: 'No children found',
    load_error: 'Error loading data',
    guests_filter_all: 'All',
    guests_filter_only: 'Guests only',
    guests_filter_none: 'No guests',
    filter_summary: {
      multiple_groups: 'Multiple groups',
      fathers: 'Fathers',
      mothers: 'Mothers',
      both: 'Fathers & Mothers',
    },
  },

  // ── Children Today / Volunteer App ───────────────────────────────────────
  children_today: {
    title: 'Children Today',
    no_children: 'No children match the current filter.',
    error_fallback: 'Error',
  },

  // ── Name Tag Handout ──────────────────────────────────────────────────────
  tag_handout: {
    title: 'Name Tag Handout',
    no_checkins: 'No registrations yet today.',
    error_fallback: 'Error',
  },

  // ── Child Detail ──────────────────────────────────────────────────────────
  child_detail: {
    title: 'Child',
    checked_in_since: 'since {time}',
    parents_section: 'Parents',
    check_in: 'Check In',
    check_out: 'Check Out',
    call_parents: 'Call Parents',
    stop_calling: 'Stop Calling',
    no_push: 'No push notification registered.',
    name_tag_received: 'Name Tag received \u2713',
    name_tag_handover: 'Hand over Name Tag',
    step_back: 'Step back',
    full_reset: 'Full reset',
    error_send: 'Error sending notification',
    error_fallback: 'Error',
  },

  // ── Admin Mode ────────────────────────────────────────────────────────────
  admin: {
    title: 'Admin Mode',
    end_event_heading: 'End Event',
    end_event_description: 'All of today\'s check-in entries will be deleted. This cannot be undone.',
    end_event_confirm: 'Really delete all of today\'s check-ins?',
    end_event_busy: 'Ending\u2026',
    end_event_button: 'End Event',
    end_event_success: 'Event ended – all entries deleted \u2713',
    end_event_error: 'Error',
    sync_heading: 'ChurchTools Sync',
    sync_description: 'Reload data from ChurchTools (persons, groups, relationships).',
    sync_busy: 'Syncing\u2026',
    sync_button: 'Sync now',
    sync_success: 'Sync successful \u2713',
    sync_error: 'Error syncing',    reports_heading: 'Event Reports',
    reports_description: 'CSV logs of all completed events with registration, check-in, and check-out times.',
    reports_empty: 'No reports yet.',
    reports_download: 'Download',
    reports_error: 'Error loading reports',  },

  // ── ChildCard ─────────────────────────────────────────────────────────────
  child_card: {
    register: 'Register',
    call_parents_notice: 'Please come to your child – notification sent at {time}',
    name_tag_done: 'Name Tag handed out \u2713',
    name_tag_action: 'Hand out Name Tag',
    check_in: 'Check In',
    call_parents: 'Call Parents',
    check_out: 'Check Out',
    detail_short: '\u2026',
  },

  // ── ChildList ─────────────────────────────────────────────────────────────
  child_list: {
    empty_fallback: 'No entries.',
  },

  // ── CheckinFilters ────────────────────────────────────────────────────────
  filters: {
    toggle: 'Filter',
    clear_title: 'Clear filters',
    name_placeholder: 'Search by name\u2026',
    status_pending: 'Registered',
    status_checked_in: 'Checked In',
    tag_received: 'Name Tag received',
    tag_missing: 'No Name Tag',
    summary_all_groups: 'All groups',
    summary_multiple_groups: 'Multiple groups',
    summary_all_status: 'All statuses',
    summary_multiple_status: 'Multiple statuses',
    summary_tag_received: 'Name Tag received',
    summary_tag_missing: 'No Name Tag',
  },

  // ── ParentDetailView (First Registration detail) ─────────────────────────
  parent_detail: {
    title: 'First Registration',
    email: 'Email',
    phone: 'Phone',
    mobile: 'Mobile',
    children_heading: 'Children',
    qr_generating: 'Generating QR code\u2026',
    qr_alt: 'QR Code',
    qr_instructions: 'Hand this QR code to the parent so they can register their children.',
    qr_download: 'Download QR code',
    qr_regenerate: 'Generate new code',
    load_error: 'Failed to load parent',
    qr_error: 'Failed to generate QR code',
    guest_edit: 'Edit',
    guest_delete: 'Delete',
    guest_delete_confirm: 'Really delete this guest family? This cannot be undone.',
  },

  // ── Dashboard ─────────────────────────────────────────────────────────────
  dashboard: {
    title: 'Dashboard',
    today_heading: 'Today',
    history_heading: 'History (last events)',
    no_data_today: 'No registrations yet today.',
    group: 'Group',
    registered: 'Registered',
    checked_in: 'Checked in',
    checked_out: 'Checked out',
    total: 'Total',
  },

  // ── Guest Form ────────────────────────────────────────────────────────────
  guest_form: {
    title_new: 'New Guest Family',
    title_edit: 'Edit Guest Family',
    parent_heading: 'Parent',
    children_heading: 'Children',
    first_name: 'First name',
    last_name: 'Last name',
    role: 'Role',
    role_father: 'Father',
    role_mother: 'Mother',
    role_other: 'Other',
    mobile: 'Mobile number',
    dob: 'Date of birth',
    group: 'Group',
    group_placeholder: 'Select group',
    add_child: 'Add child',
    child_n: 'Child {n}',
    no_children_hint: 'No children added yet.',
    submit: 'Save guest family',
    delete: 'Delete guest family',
    delete_confirm: 'Really delete this guest family?',
    load_error: 'Error loading data',
    error_parent_name: 'Name required',
    error_child_name: 'Name required',
    error_group_required: 'Group required',
    error_mobile_invalid: 'Please enter a valid phone number',
  },
}
