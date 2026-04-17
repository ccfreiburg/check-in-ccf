export default {
  // ── Glossar ──────────────────────────────────────────────────────────────
  status: {
    pending: 'Angemeldet',
    checked_in: 'Eingecheckt',
    not_registered: 'Noch nicht angemeldet',
  },

  // ── Gemeinsame UI ────────────────────────────────────────────────────────
  common: {
    loading: 'Wird geladen\u2026',
    please_wait: 'Bitte warten\u2026',
    back: 'Zurück',
    error: 'Fehler',
    not_found: 'Nicht gefunden.',
    close: '\u00d7',
    ccf_alt: 'CCF',
  },

  // ── Navigation ───────────────────────────────────────────────────────────
  nav: {
    menu_label: 'Menü',
    menu_heading: 'Menü',
    logout: 'Abmelden',
    first_registration: 'Erstregistrierung',
    name_tag_handout: 'Namensschildausgabe',
    children_today: 'Kinder heute',
    dashboard: 'Dashboard',
    stats: 'Verlauf',
    admin: 'Admin-Modus',
    lang_switch: 'Sprache wechseln',
    lang_modal_heading: 'Sprache wählen',
  },

  // ── Login ────────────────────────────────────────────────────────────────
  login: {
    title: 'Admin Login',    username_placeholder: 'E-Mail (ChurchTools)',    password_placeholder: 'Passwort',
    signing_in: 'Anmelden\u2026',
    sign_in: 'Anmelden',
    error_fallback: 'Anmeldung fehlgeschlagen',
  },

  // ── Parent App ───────────────────────────────────────────────────────────
  parent: {
    title: 'Kinder Anmeldung',
    greeting: 'Hallo {firstName} {lastName}',
    subtitle: 'Du kannst hier deine Kinder für den heutigen Tag anmelden.',
    link_expired: 'Dieser Link ist möglicherweise abgelaufen. Bitte beim Dienst einen neuen QR-Code anfordern.',
    qr_button_title: 'QR-Code',
    qr_alt: 'QR-Code',
    share_qr: 'QR-Code teilen',
    share_title: 'Kinder Anmeldung',
    share_text: 'Anmeldelink für {firstName} {lastName}',
    link_copied: 'Link in die Zwischenablage kopiert \u2713',
    link_fallback: 'Link: {url}',
    install_app: '📲 App installieren',
    ios_push_heading: '📲 Für Benachrichtigungen auf iPhone:',
    ios_push_steps: 'Tippe Teilen → „Zum Home-Bildschirm" → App öffnen.',
    enable_push: '🔔 Benachrichtigungen aktivieren',
    push_granted: '🔔 Benachrichtigungen aktiviert \u2713',
    push_denied: 'Benachrichtigungen wurden blockiert. Bitte in den Browser-Einstellungen freigeben.',
    push_error: 'Fehler: {error}',
    no_children: 'Keine Kinder hinterlegt. Bitte beim Dienst melden.',
    registered: '{firstName} wurde angemeldet \u2713',
    register_error: 'Fehler beim Anmelden',
  },

  // ── Erstregistrierung ────────────────────────────────────────────────────
  first_registration: {
    title: 'Erstregistrierung',
    search_placeholder: 'Name suchen\u2026',
    filter: 'Filter',
    filter_clear_title: 'Filter löschen',
    fathers: 'Väter',
    mothers: 'Mütter',
    no_sex_data: 'Kein Geschlecht in der Datenbank — bitte CT-Sync durchführen.',
    no_parents: 'Keine Eltern gefunden',
    no_children: 'Keine Kinder gefunden',
    load_error: 'Fehler beim Laden',
    guests_filter_all: 'Alle',
    guests_filter_only: 'Nur Gäste',
    guests_filter_none: 'Ohne Gäste',
    filter_summary: {
      multiple_groups: 'Mehrere Gruppen',
      fathers: 'Väter',
      mothers: 'Mütter',
      both: 'Väter & Mütter',
    },
  },

  // ── Kinder heute / Volunteer App ─────────────────────────────────────────
  children_today: {
    title: 'Kinder heute',
    no_children: 'Keine Kinder in dieser Auswahl.',
    error_fallback: 'Fehler',
  },

  // ── Namensschildausgabe ───────────────────────────────────────────────────
  tag_handout: {
    title: 'Namensschildausgabe',
    no_checkins: 'Heute noch keine Anmeldungen.',
    error_fallback: 'Fehler',
  },

  // ── Kind Detail ───────────────────────────────────────────────────────────
  child_detail: {
    title: 'Kind',
    checked_in_since: 'seit {time}',
    parents_section: 'Eltern',
    check_in: 'Check In',
    check_out: 'Check Out',
    call_parents: 'Eltern rufen',
    stop_calling: 'Rufen beenden',
    no_push: 'Keine Push-Benachrichtigung aktiviert.',
    name_tag_received: 'Namensschild erhalten \u2713',
    name_tag_handover: 'Namensschild übergeben',
    step_back: 'Schritt zurück',
    full_reset: 'Ganz zurück',
    error_send: 'Fehler beim Senden',
    error_fallback: 'Fehler',
  },

  // ── Admin-Modus ───────────────────────────────────────────────────────────
  admin: {
    title: 'Admin-Modus',
    end_event_heading: 'Event beenden',
    end_event_description: 'Alle heutigen Check-in-Einträge werden gelöscht. Dies kann nicht rückgängig gemacht werden.',
    end_event_confirm: 'Wirklich alle heutigen Check-ins löschen?',
    end_event_busy: 'Wird beendet\u2026',
    end_event_button: 'Event beenden',
    end_event_success: 'Event beendet – alle Einträge gelöscht \u2713',
    end_event_error: 'Fehler',
    sync_heading: 'ChurchTools Synchronisierung',
    sync_description: 'Daten aus ChurchTools neu laden (Personen, Gruppen, Beziehungen).',
    sync_busy: 'Synchronisiere\u2026',
    sync_button: 'Jetzt synchronisieren',
    sync_success: 'Synchronisierung erfolgreich \u2713',
    sync_error: 'Fehler beim Synchronisieren',    reports_heading: 'Veranstaltungsberichte',
    reports_description: 'CSV-Protokolle aller beendeten Events mit Registrierungs-, Check-in- und Check-out-Zeiten.',
    reports_empty: 'Noch keine Berichte vorhanden.',
    reports_download: 'Herunterladen',
    reports_error: 'Fehler beim Laden der Berichte',  },

  // ── ChildCard ─────────────────────────────────────────────────────────────
  child_card: {
    register: 'Anmelden',
    call_parents_notice: 'Bitte zum Kind kommen – Nachricht gesendet um {time}',
    name_tag_done: 'Namensschildausgabe \u2713',
    name_tag_action: 'Namensschildausgabe',
    check_in: 'Check In',
    call_parents: 'Eltern rufen',
    check_out: 'Check Out',
    detail_short: '\u2026',
  },

  // ── ChildList ─────────────────────────────────────────────────────────────
  child_list: {
    empty_fallback: 'Keine Einträge.',
  },

  // ── CheckinFilters ────────────────────────────────────────────────────────
  filters: {
    toggle: 'Filter',
    clear_title: 'Filter löschen',
    name_placeholder: 'Name suchen\u2026',
    status_pending: 'Angemeldet',
    status_checked_in: 'Eingecheckt',
    tag_received: 'Namensschild erhalten',
    tag_missing: 'Kein Namensschild',
    summary_all_groups: 'Alle Gruppen',
    summary_multiple_groups: 'Mehrere Gruppen',
    summary_all_status: 'Alle Status',
    summary_multiple_status: 'Mehrere Status',
    summary_tag_received: 'Namensschild erhalten',
    summary_tag_missing: 'Kein Namensschild',
  },

  // ── ParentDetailView (Erstregistrierung detail) ─────────────────────────
  parent_detail: {
    title: 'Erstregistrierung',
    email: 'E-Mail',
    phone: 'Telefon',
    mobile: 'Mobil',
    children_heading: 'Kinder',
    qr_generating: 'QR Code wird generiert\u2026',
    qr_alt: 'QR Code',
    qr_instructions: 'Diesen QR-Code den Eltern übergeben, damit sie ihre Kinder anmelden können.',
    qr_download: 'QR-Code herunterladen',
    qr_regenerate: 'Neuen Code generieren',
    load_error: 'Fehler beim Laden der Eltern',
    qr_error: 'Fehler beim Generieren des QR-Codes',
    guest_edit: 'Bearbeiten',
    guest_delete: 'Löschen',
    guest_delete_confirm: 'Gastfamilie wirklich löschen? Diese Aktion kann nicht rückgängig gemacht werden.',
  },

  // ── Dashboard ────────────────────────────────────────────────────────────
  dashboard: {
    title: 'Dashboard',
    today_heading: 'Heute',
    history_heading: 'Verlauf (letzte Events)',
    no_data_today: 'Heute noch keine Anmeldungen.',
    group: 'Gruppe',
    registered: 'Angemeldet',
    checked_in: 'Eingecheckt',
    checked_out: 'Ausgecheckt',
    total: 'Gesamt',
  },

  // ── Verlauf (Statistik) ────────────────────────────────────────────────
  stats: {
    title: 'Besuchsentwicklung',
    no_data: 'Noch keine abgeschlossenen Events vorhanden.',
    n_events: '{n} Events',
    total_all_groups: 'Gesamt – alle Gruppen',
  },

  // ── Gastfamilie Formular ──────────────────────────────────────────────────
  guest_form: {
    title_new: 'Neue Gastfamilie',
    title_edit: 'Gastfamilie bearbeiten',
    parent_heading: 'Elternteil',
    children_heading: 'Kinder',
    first_name: 'Vorname',
    last_name: 'Nachname',
    role: 'Rolle',
    role_father: 'Vater',
    role_mother: 'Mutter',
    role_other: 'Sonstiges',
    mobile: 'Mobilnummer',
    dob: 'Geburtsdatum',
    group: 'Gruppe',
    group_placeholder: 'Gruppe wählen',
    add_child: 'Kind hinzufügen',
    child_n: 'Kind {n}',
    no_children_hint: 'Noch kein Kind hinzugefügt.',
    submit: 'Gastfamilie speichern',
    delete: 'Gastfamilie löschen',
    delete_confirm: 'Gastfamilie wirklich löschen?',
    load_error: 'Fehler beim Laden',
    error_parent_name: 'Name erforderlich',
    error_child_name: 'Name erforderlich',
    error_group_required: 'Gruppe erforderlich',
    error_mobile_invalid: 'Bitte gültige Telefonnummer eingeben',
  },
}
