import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { AuditLog } from '../lib/types'
import { ControlBar, EmptySearch, PageHeader, SearchInput, Status, TableState, Tag } from '../components/ui'
import { formatDate, lower } from '../lib/format'

const events: Array<{ id: string; label: string }> = [
  { id: '', label: 'Tüm olaylar' },
  { id: 'login', label: 'Giriş' },
  { id: 'login_failed', label: 'Başarısız giriş' },
  { id: 'logout', label: 'Çıkış' },
  { id: 'token_refresh', label: 'Token yenileme' },
  { id: 'access_denied', label: 'Erişim reddi' },
  { id: 'token_revoke', label: 'Token iptali' },
]

const eventLabels: Record<string, string> = Object.fromEntries(events.filter((e) => e.id).map((e) => [e.id, e.label]))

export function AuditPage() {
  const [event, setEvent] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const logs = useQuery({ queryKey: ['audit', event], queryFn: () => api.auditLogs({ event: event || undefined, limit: 200 }) })

  const filtered = useMemo(() => {
    const q = lower(searchQuery)
    return (logs.data ?? []).filter((log) =>
      lower(log.email).includes(q) || lower(log.ip).includes(q) || lower(log.clientId).includes(q) ||
      lower(log.message).includes(q) || lower(log.browser).includes(q) || lower(log.os).includes(q),
    )
  }, [logs.data, searchQuery])

  return (
    <div className="directory-section">
      <PageHeader eyebrow="İzleme" title="Denetim kayıtları" />
      <ControlBar onRefresh={() => logs.refetch()} action={null}>
        <div className="role-filters">
          <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="E-posta, IP, tarayıcı veya mesaj ara..." />
          <select value={event} onChange={(e) => setEvent(e.target.value)}>
            {events.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
          </select>
        </div>
      </ControlBar>

      <p className="section-note">Tüm giriş, çıkış ve token olayları; IP adresi, cihaz, tarayıcı ve işletim sistemi bilgisiyle kaydedilir.</p>

      <TableState query={logs} />
      {logs.data && logs.data.length > 0 && filtered.length === 0 && <EmptySearch text={`"${searchQuery}" için kayıt bulunamadı.`} />}

      {filtered.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Zaman</th><th>Olay</th><th>Kullanıcı</th><th>Uygulama</th><th>IP</th><th>Cihaz / Tarayıcı</th><th>Sonuç</th></tr>
            </thead>
            <tbody>
              {filtered.map((log) => (
                <tr key={log.ID}>
                  <td className="mono-text">{formatDate(log.CreatedAt)}</td>
                  <td><Tag>{eventLabels[log.event] ?? log.event}</Tag></td>
                  <td>{log.email || <span className="muted-text">—</span>}{log.message && <div className="muted-text">{log.message}</div>}</td>
                  <td className="mono-text">{log.clientId || '—'}</td>
                  <td className="mono-text">{log.ip || '—'}</td>
                  <td>{deviceLabel(log)}</td>
                  <td><Status active={log.success} blocked={!log.success} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function deviceLabel(log: AuditLog) {
  const parts = [log.device, log.browser, log.os].filter((p) => p && p !== 'unknown')
  return parts.length ? parts.join(' · ') : '—'
}
