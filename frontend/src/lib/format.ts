// Small presentation helpers shared across pages. Kept separate from the React
// component module so fast-refresh treats components and utilities correctly.

export function randomSecret() {
    const bytes = new Uint8Array(32)
    crypto.getRandomValues(bytes)
    return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

export function maskSecret(secret: string) {
    if (secret.length <= 10) return '••••••'
    return `${secret.slice(0, 4)}••••••••${secret.slice(-4)}`
}

export function formatDate(value?: string | null) {
    if (!value) return '—'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return '—'
    return date.toLocaleString('tr-TR', { dateStyle: 'short', timeStyle: 'short' })
}

export function lower(value: string | undefined) {
    return (value ?? '').toLocaleLowerCase('tr')
}
