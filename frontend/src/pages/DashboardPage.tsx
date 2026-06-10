import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import type { LinkProps } from '@tanstack/react-router'
import { Building2, FileClock, KeyRound, Layers, Shield, ShieldAlert, UserCog, Users } from 'lucide-react'
import { api } from '../lib/api'
import { PageHeader } from '../components/ui'
import { formatDate } from '../lib/format'

export function DashboardPage() {
    const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
    const departments = useQuery({ queryKey: ['departments'], queryFn: api.departments.list })
    const groups = useQuery({ queryKey: ['groups'], queryFn: api.groups.list })
    const roles = useQuery({ queryKey: ['roles'], queryFn: api.roles.list })
    const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
    const policies = useQuery({ queryKey: ['policies'], queryFn: api.policies.list })
    const audit = useQuery({ queryKey: ['audit', ''], queryFn: () => api.auditLogs({ limit: 8 }) })
    const auditItems = audit.data?.items ?? []

    const activeUsers = users.data?.filter((user) => user.active && !user.blocked).length ?? 0
    const blockedUsers = users.data?.filter((user) => user.blocked).length ?? 0
    const clientRoles = clients.data?.reduce((total, client) => total + (client.roles?.length ?? 0), 0) ?? 0

    return (
        <div className="directory-section">
            <PageHeader eyebrow="Genel bakış" title="Kontrol paneli" />

            <section className="metrics-grid">
                <Metric icon={Users} value={activeUsers} label="Aktif kullanıcı" to="/users" />
                <Metric icon={Building2} value={departments.data?.length ?? 0} label="Departman" to="/departments" />
                <Metric icon={Layers} value={groups.data?.length ?? 0} label="Grup" to="/groups" />
                <Metric icon={Shield} value={(roles.data?.length ?? 0) + clientRoles} label="Toplam rol" to="/roles" />
                <Metric icon={KeyRound} value={clients.data?.length ?? 0} label="OIDC client" to="/clients" />
                <Metric icon={ShieldAlert} value={policies.data?.length ?? 0} label="Politika" to="/policies" />
            </section>

            {blockedUsers > 0 && (
                <div className="alert-box error">
                    <ShieldAlert size={16} />
                    <span>{blockedUsers} kullanıcı engelli durumda. Detaylar için Kullanıcılar bölümüne göz atın.</span>
                </div>
            )}

            <div className="dashboard-grid">
                <article className="dashboard-card">
                    <header><FileClock size={16} /><h3>Son denetim olayları</h3><Link to="/audit" className="card-link">Tümünü gör</Link></header>
                    {auditItems.length ? (
                        <ul className="event-list">
                            {auditItems.map((log) => (
                                <li key={log.ID}>
                                    <span className={`event-dot ${log.success ? 'ok' : 'fail'}`} />
                                    <div>
                                        <strong>{log.email || 'bilinmeyen'}</strong>
                                        <span className="muted-text"> · {log.event} · {log.ip || '—'}</span>
                                    </div>
                                    <span className="event-time">{formatDate(log.CreatedAt)}</span>
                                </li>
                            ))}
                        </ul>
                    ) : <p className="muted-text">Henüz olay kaydedilmedi.</p>}
                </article>

                <article className="dashboard-card">
                    <header><UserCog size={16} /><h3>Hızlı erişim</h3></header>
                    <div className="quick-links">
                        <Link to="/users" className="quick-link"><Users size={14} />Kullanıcı yönetimi</Link>
                        <Link to="/clients" className="quick-link"><KeyRound size={14} />OIDC uygulamaları</Link>
                        <Link to="/policies" className="quick-link"><ShieldAlert size={14} />Giriş politikaları</Link>
                        <Link to="/profile" className="quick-link"><UserCog size={14} />Profilim</Link>
                        <a className="quick-link" href="/.well-known/openid-configuration" target="_blank" rel="noreferrer"><FileClock size={14} />OIDC keşfi</a>
                    </div>
                </article>
            </div>
        </div>
    )
}

function Metric({ icon: Icon, value, label, to }: { icon: typeof Users; value: number; label: string; to: LinkProps['to'] }) {
    return (
        <Link to={to} className="metric-card">
            <Icon size={20} />
            <strong>{value}</strong>
            <p>{label}</p>
        </Link>
    )
}
