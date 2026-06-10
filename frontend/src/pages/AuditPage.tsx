import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { AuditLog } from '../lib/types'
import { ControlBar, EmptySearch, PageHeader, SearchInput, Status, TableState, Tag } from '../components/ui'
import { formatDate } from '../lib/format'

const events: Array<{ id: string; label: string }> = [
  { id: '', label: 'Tüm olaylar' },
  { id: 'login', label: 'Giriş' },
  { id: 'login_failed', label: 'Başarısız giriş' },
  { id: 'logout', label: 'Çıkış' },
  { id: 'token_refresh', label: 'Token yenileme' },
  { id: 'access_denied', label: 'Erişim reddi' },
  { id: 'token_revoke', label: 'Token iptali' },
  { id: 'user_blocked', label: 'Kullanıcı engellendi' },
  { id: 'user_unblocked', label: 'Kullanıcı engeli kaldırıldı' },
]

const eventLabels: Record<string, string> = Object.fromEntries(events.filter((e) => e.id).map((e) => [e.id, e.label]))

export function AuditPage() {
  const [event, setEvent] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [clientId, setClientId] = useState('')
  const [ip, setIp] = useState('')
  const [success, setSuccess] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const pageSize = 25
  const logs = useQuery({
    queryKey: ['audit', event, searchQuery, clientId, ip, success, from, to, page],
    queryFn: () => api.auditLogs({
      event: event || undefined,
      email: searchQuery || undefined,
      clientId: clientId || undefined,
      ip: ip || undefined,
      success: success === '' ? undefined : success === 'true',
      from: from || undefined,
      to: to || undefined,
      page,
      pageSize,
    }),
  })

  const items = logs.data?.items ?? []
  const totalPages = Math.max(1, Math.ceil((logs.data?.total ?? 0) / pageSize))

  return (
    <div className="directory-section">
      <PageHeader eyebrow="İzleme" title="Denetim kayıtları" />
      <ControlBar onRefresh={() => logs.refetch()} action={null}>
        <div className="audit-filters">
          <SearchInput value={searchQuery} onChange={(value) => { setSearchQuery(value); setPage(1) }} placeholder="E-posta ara..." />
          <input value={clientId} onChange={(e) => { setClientId(e.target.value); setPage(1) }} placeholder="Client ID" />
          <input value={ip} onChange={(e) => { setIp(e.target.value); setPage(1) }} placeholder="IP" />
          <select value={event} onChange={(e) => { setEvent(e.target.value); setPage(1) }}>
            {events.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
          </select>
          <select value={success} onChange={(e) => { setSuccess(e.target.value); setPage(1) }}>
            <option value="">Tüm sonuçlar</option>
            <option value="true">Başarılı</option>
            <option value="false">Başarısız</option>
          </select>
          <input type="date" value={from} onChange={(e) => { setFrom(e.target.value); setPage(1) }} />
          <input type="date" value={to} onChange={(e) => { setTo(e.target.value); setPage(1) }} />
        </div>
      </ControlBar>

      <p className="section-note">Tüm giriş, çıkış ve token olayları; IP adresi, cihaz, tarayıcı ve işletim sistemi bilgisiyle kaydedilir.</p>

      <TableState query={{ ...logs, data: items }} />
      {logs.data && logs.data.total > 0 && items.length === 0 && <EmptySearch text="Bu filtreler için kayıt bulunamadı." />}

      {items.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Zaman</th><th>Olay</th><th>Kullanıcı</th><th>Uygulama</th><th>IP</th><th>Cihaz / Tarayıcı</th><th>Sonuç</th></tr>
            </thead>
            <tbody>
              {items.map((log) => (
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
      {logs.data && logs.data.total > 0 && (
        <div className="pagination-bar">
          <span>{logs.data.total} kayıt · Sayfa {page} / {totalPages}</span>
          <div>
            <button type="button" className="btn-secondary" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>Önceki</button>
            <button type="button" className="btn-secondary" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>Sonraki</button>
          </div>
        </div>
      )}
    </div>
  )
}

function deviceLabel(log: AuditLog) {
  const parts = [log.device, log.browser, log.os].filter((p) => p && p !== 'unknown')
  return parts.length ? parts.join(' · ') : '—'
}
