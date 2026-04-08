import { createI18n } from 'vue-i18n'
import de from './de'
import en from './en'

const LOCALE_KEY = 'ccf_locale'
const saved = localStorage.getItem(LOCALE_KEY)
const detected = navigator.language.startsWith('de') ? 'de' : 'en'

export const i18n = createI18n({
  legacy: false,
  locale: saved ?? detected,
  fallbackLocale: 'de',
  messages: { de, en },
})

export function setLocale(code: string) {
  ;(i18n.global.locale as { value: string }).value = code
  localStorage.setItem(LOCALE_KEY, code)
}

export default i18n
