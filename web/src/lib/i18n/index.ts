import { browser } from '$app/environment';
import { getPb } from '$lib/pocketbase';
import { init, register } from 'svelte-i18n';
import { defaultLocale, normalizeLocale } from './locales';

register('cs', () => import('./locales/cs.json'))
register('en', () => import('./locales/en.json'))
register('de', () => import('./locales/de.json'))
register('es', () => import('./locales/es.json'))
register('eu', () => import('./locales/eu.json'))
register('fr', () => import('./locales/fr.json'))
register('hu', () => import('./locales/hu.json'))
register('it', () => import('./locales/it.json'))
register('nl', () => import('./locales/nl.json'))
register('no', () => import('./locales/no.json'))
register('pl', () => import('./locales/pl.json'))
register('pt', () => import('./locales/pt.json'))
register('ru', () => import('./locales/ru.json'))
register('zh', () => import('./locales/zh.json'))

const userLang = browser ? getPb().authStore.record?.language : null;
const navigatorLang = browser ? window.navigator.language : null;

const initial = normalizeLocale(userLang ?? navigatorLang ?? defaultLocale);

init({
    fallbackLocale: defaultLocale,
    initialLocale: initial,
    formats: {
        date: {
            monthName: { month: 'long' }
        },
        number: {},
        time: {}
    },
})
