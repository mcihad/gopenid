import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound, LogOut, Monitor } from 'lucide-react'
import { me } from '../lib/api'
import type { User } from '../lib/types'
import { Alert, Field, Muted, PageHeader, Tag } from '../components/ui'
import { formatDate } from '../lib/format'

type ProfileTab = 'profile' | 'security' | 'access' | 'sessions'

export function ProfilePage() {
    const [tab, setTab] = useState<ProfileTab>('profile')
    return (
        <div className="directory-section">
            <PageHeader eyebrow="Hesabım" title="Profil ve güvenlik" />
            <div className="section-tabs">
                <button className={`section-tab ${tab === 'profile' ? 'active' : ''}`} onClick={() => setTab('profile')}>Profil</button>
                <button className={`section-tab ${tab === 'security' ? 'active' : ''}`} onClick={() => setTab('security')}>Parola</button>
                <button className={`section-tab ${tab === 'access' ? 'active' : ''}`} onClick={() => setTab('access')}>Yetkilerim</button>
                <button className={`section-tab ${tab === 'sessions' ? 'active' : ''}`} onClick={() => setTab('sessions')}>Oturumlar</button>
            </div>
            {tab === 'profile' && <ProfileForm />}
            {tab === 'security' && <PasswordForm />}
            {tab === 'access' && <AccessOverview />}
            {tab === 'sessions' && <SessionsList />}
        </div>
    )
}

function ProfileForm() {
    const profile = useQuery({ queryKey: ['me'], queryFn: me.profile })
    if (profile.isLoading) return <Muted>Yükleniyor...</Muted>
    if (!profile.data) return <Muted>Profil bilgisi yüklenemedi.</Muted>
    // Key by user id so the editor remounts with fresh initial state when data
    // changes — avoids syncing props to state inside an effect.
    return <ProfileEditor key={profile.data.ID} profile={profile.data} />
}

function ProfileEditor({ profile }: { profile: User }) {
    const qc = useQueryClient()
    const [draft, setDraft] = useState({ name: profile.name, phone: profile.phone ?? '', title: profile.title ?? '', avatarUrl: profile.avatarUrl ?? '' })

    const save = useMutation({
        mutationFn: () => me.update(draft),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['me'] }),
    })

    return (
        <form className="profile-card" onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
            {save.error && <Alert type="error" message={save.error.message} />}
            {save.isSuccess && <Alert type="success" message="Profil güncellendi." />}
            <div className="profile-identity">
                {draft.avatarUrl ? <img className="profile-avatar" src={draft.avatarUrl} alt="" /> : <span className="profile-avatar placeholder">{draft.name.charAt(0) || '?'}</span>}
                <div>
                    <strong>{profile.name}</strong>
                    <span className="mono-text">{profile.email}</span>
                </div>
            </div>
            <div className="form-row">
                <Field label="Ad soyad"><input value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required /></Field>
                <Field label="Ünvan"><input value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="örn. Yazılım Mühendisi" /></Field>
            </div>
            <div className="form-row">
                <Field label="Telefon"><input value={draft.phone} onChange={(event) => setDraft({ ...draft, phone: event.target.value })} placeholder="+90 5xx xxx xx xx" /></Field>
                <Field label="Avatar URL"><input value={draft.avatarUrl} onChange={(event) => setDraft({ ...draft, avatarUrl: event.target.value })} placeholder="https://..." /></Field>
            </div>
            <div className="form-actions">
                <button className="btn-primary" disabled={save.isPending}>{save.isPending ? 'Kaydediliyor...' : 'Profili kaydet'}</button>
            </div>
        </form>
    )
}

function PasswordForm() {
    const [draft, setDraft] = useState({ currentPassword: '', newPassword: '', confirm: '' })
    const change = useMutation({
        mutationFn: () => me.changePassword({ currentPassword: draft.currentPassword, newPassword: draft.newPassword }),
        onSuccess: () => setDraft({ currentPassword: '', newPassword: '', confirm: '' }),
    })
    const mismatch = draft.newPassword.length > 0 && draft.newPassword !== draft.confirm

    return (
        <form className="profile-card" onSubmit={(event) => { event.preventDefault(); if (!mismatch) change.mutate() }}>
            {change.error && <Alert type="error" message={change.error.message} />}
            {change.isSuccess && <Alert type="success" message="Parolanız değiştirildi. Diğer oturumlarınız sonlandırıldı." />}
            <p className="section-note"><KeyRound size={13} /> Parola değişiminde mevcut tüm oturumlarınız (refresh token) iptal edilir.</p>
            <Field label="Mevcut parola"><input type="password" value={draft.currentPassword} onChange={(event) => setDraft({ ...draft, currentPassword: event.target.value })} required /></Field>
            <div className="form-row">
                <Field label="Yeni parola" hint="En az 8 karakter"><input type="password" value={draft.newPassword} onChange={(event) => setDraft({ ...draft, newPassword: event.target.value })} required minLength={8} /></Field>
                <Field label="Yeni parola (tekrar)"><input type="password" value={draft.confirm} onChange={(event) => setDraft({ ...draft, confirm: event.target.value })} required /></Field>
            </div>
            {mismatch && <Alert type="error" message="Parolalar eşleşmiyor." />}
            <div className="form-actions">
                <button className="btn-primary" disabled={change.isPending || mismatch}>{change.isPending ? 'Değiştiriliyor...' : 'Parolayı değiştir'}</button>
            </div>
        </form>
    )
}

function AccessOverview() {
    const roles = useQuery({ queryKey: ['me', 'roles'], queryFn: me.roles })
    const departments = useQuery({ queryKey: ['me', 'departments'], queryFn: me.departments })
    const groups = useQuery({ queryKey: ['me', 'groups'], queryFn: me.groups })
    const clients = useQuery({ queryKey: ['me', 'clients'], queryFn: me.clients })

    return (
        <div className="access-grid">
            <AccessCard title="Sistem rollerim" items={roles.data?.map((r) => r.name)} />
            <AccessCard title="Departmanlarım" items={departments.data?.map((d) => d.name)} />
            <AccessCard title="Gruplarım" items={groups.data?.map((g) => g.name)} />
            <AccessCard title="Erişebildiğim uygulamalar" items={clients.data?.map((c) => `${c.name} (${c.clientId})`)} />
        </div>
    )
}

function AccessCard({ title, items }: { title: string; items?: string[] }) {
    return (
        <article className="profile-card">
            <h3 className="access-card-title">{title}</h3>
            {items && items.length > 0 ? <div className="tag-cloud">{items.map((item) => <Tag key={item}>{item}</Tag>)}</div> : <Muted>Kayıt yok.</Muted>}
        </article>
    )
}

function SessionsList() {
    const sessions = useQuery({ queryKey: ['me', 'sessions'], queryFn: me.sessions })
    if (sessions.isLoading) return <Muted>Yükleniyor...</Muted>
    const rows = sessions.data ?? []
    if (rows.length === 0) return <Muted>Aktif oturum bulunmuyor.</Muted>

    return (
        <div className="data-table-container">
            <table className="carbon-table">
                <thead>
                    <tr><th>Uygulama</th><th>Kapsam</th><th>Oluşturulma</th><th>Geçerlilik</th><th>Durum</th></tr>
                </thead>
                <tbody>
                    {rows.map((session) => (
                        <tr key={session.ID}>
                            <td><Monitor size={14} /> {session.clientId || 'gopenid'}</td>
                            <td className="mono-text">{session.scope || '—'}</td>
                            <td>{formatDate(session.CreatedAt)}</td>
                            <td>{formatDate(session.expiresAt)}</td>
                            <td>{session.revoked ? <span className="status-badge blocked">İptal</span> : <span className="status-badge active">Aktif</span>}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
            <p className="section-note"><LogOut size={13} /> Çıkış yaparak tüm oturumlarınızı iptal edebilirsiniz.</p>
        </div>
    )
}
