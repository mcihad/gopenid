import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Plus, Shuffle } from 'lucide-react'
import { api } from '../lib/api'
import type { Client, ClientRole } from '../lib/types'
import { Actions, Alert, ControlBar, EmptySearch, Field, Modal, ModalFooter, Muted, PageHeader, SearchInput, TableState } from '../components/ui'
import { lower, maskSecret, randomSecret } from '../lib/format'

type ClientModalTab = 'general' | 'security' | 'roles'

type ClientDraft = {
  clientId: string
  clientSecret: string
  name: string
  description: string
  homeUrl: string
  logoUrl: string
  redirectUris: string
  tokenTtlSeconds: number
  refreshTtlSeconds: number
}

export function ClientsPage() {
  const qc = useQueryClient()
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<Client | null>(null)
  const [draft, setDraft] = useState<ClientDraft>(clientDraft())
  const [modalTab, setModalTab] = useState<ClientModalTab>('general')
  const [roleDraft, setRoleDraft] = useState({ name: '', description: '' })
  const [editRole, setEditRole] = useState<ClientRole | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [copied, setCopied] = useState(false)

  const save = useMutation({
    mutationFn: () => (edit ? api.clients.update(edit.ID, draft) : api.clients.create(draft)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['clients'] }); closeModal() },
  })
  const remove = useMutation({
    mutationFn: (id: number) => api.clients.remove(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['clients'] }); qc.invalidateQueries({ queryKey: ['users'] }) },
  })
  const saveRole = useMutation({
    mutationFn: () => {
      if (!currentClient) throw new Error('Client seçili değil.')
      if (editRole) return api.clients.roles.update(currentClient.ID, editRole.ID, roleDraft)
      return api.clients.roles.create(currentClient.ID, roleDraft)
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['clients'] }); resetRoleEditor() },
  })
  const removeRole = useMutation({
    mutationFn: (role: ClientRole) => {
      if (!currentClient) throw new Error('Client seçili değil.')
      return api.clients.roles.remove(currentClient.ID, role.ID)
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['clients'] }); qc.invalidateQueries({ queryKey: ['users'] }); resetRoleEditor() },
  })

  const filteredClients = useMemo(() => {
    const q = lower(searchQuery)
    return (clients.data ?? []).filter((client) => lower(client.name).includes(q) || lower(client.clientId).includes(q))
  }, [clients.data, searchQuery])
  const currentClient = edit ? clients.data?.find((client) => client.ID === edit.ID) ?? edit : null
  const getUserCount = (clientID: number) => users.data?.filter((user) => user.authorizedClients?.some((client) => client.ID === clientID)).length ?? 0
  const startRoleEdit = (role: ClientRole) => { setEditRole(role); setRoleDraft({ name: role.name, description: role.description }); saveRole.reset() }
  const resetRoleEditor = () => { setEditRole(null); setRoleDraft({ name: '', description: '' }); saveRole.reset() }

  const openCreateModal = () => { setEdit(null); setDraft(clientDraft()); setModalTab('general'); resetRoleEditor(); save.reset(); setCopied(false); setIsModalOpen(true) }
  const openEditModal = (client: Client) => { setEdit(client); setDraft(fromClient(client)); setModalTab('general'); resetRoleEditor(); save.reset(); setCopied(false); setIsModalOpen(true) }
  const closeModal = () => { setIsModalOpen(false); setEdit(null); setDraft(clientDraft()); setModalTab('general'); resetRoleEditor() }
  const copySecret = async () => { await navigator.clipboard.writeText(draft.clientSecret); setCopied(true); window.setTimeout(() => setCopied(false), 1600) }

  return (
    <div className="directory-section">
      <PageHeader eyebrow="OIDC entegrasyonu" title="Uygulamalar (Clientlar)" />
      <ControlBar onRefresh={() => clients.refetch()} action={<button className="btn-primary" onClick={openCreateModal}><Plus size={14} />Client ekle</button>}>
        <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Client adı veya client ID ara..." />
      </ControlBar>

      <TableState query={clients} />
      {clients.data && clients.data.length > 0 && filteredClients.length === 0 && <EmptySearch text={`"${searchQuery}" için client bulunamadı.`} />}

      {filteredClients.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Client</th><th>Client ID</th><th>Secret</th><th>Token / Oturum ömrü</th><th>Roller</th><th>Kullanıcı</th><th></th></tr>
            </thead>
            <tbody>
              {filteredClients.map((client) => (
                <tr key={client.ID} onClick={() => openEditModal(client)}>
                  <td>
                    <div className="client-name-cell">
                      {client.logoUrl ? <img className="client-logo-thumb" src={client.logoUrl} alt="" /> : <span className="client-logo-thumb placeholder">{client.name.charAt(0)}</span>}
                      <div><strong>{client.name}</strong>{client.homeUrl && <div className="muted-text">{client.homeUrl}</div>}</div>
                    </div>
                  </td>
                  <td className="mono-text">{client.clientId}</td>
                  <td className="mono-text secret-preview">{maskSecret(client.clientSecret)}</td>
                  <td>{formatDuration(client.tokenTtlSeconds)} / {formatDuration(client.refreshTtlSeconds)}</td>
                  <td>{client.roles?.length ?? 0}</td>
                  <td>{getUserCount(client.ID)}</td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(client) }}
                      onDelete={(event) => { event.stopPropagation(); if (confirm('Client silinsin mi?')) remove.mutate(client.ID) }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal wide isOpen={isModalOpen} onClose={closeModal} title={edit ? `Client düzenle: ${edit.name}` : 'Yeni OIDC client'}>
        <div className="modal-tabs">
          <button type="button" className={`modal-tab ${modalTab === 'general' ? 'active' : ''}`} onClick={() => setModalTab('general')}>Bilgiler</button>
          <button type="button" className={`modal-tab ${modalTab === 'security' ? 'active' : ''}`} onClick={() => setModalTab('security')}>Güvenlik & Oturum</button>
          {currentClient && <button type="button" className={`modal-tab ${modalTab === 'roles' ? 'active' : ''}`} onClick={() => setModalTab('roles')}>Client rolleri ({currentClient.roles?.length ?? 0})</button>}
        </div>

        {modalTab !== 'roles' ? (
          <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
            <div className="modal-body">
              <div className="modal-form">
                {save.error && <Alert type="error" message={save.error.message} />}
                {modalTab === 'general' ? (
                  <>
                    <div className="form-row">
                      <Field label="Client adı"><input placeholder="örn. Web uygulaması" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required /></Field>
                      <Field label="Client ID"><input placeholder="örn. web-app" value={draft.clientId} onChange={(event) => setDraft({ ...draft, clientId: event.target.value })} required disabled={Boolean(edit)} /></Field>
                    </div>
                    <Field label="Açıklama" hint="Giriş sayfasında kullanıcıya gösterilir."><textarea placeholder="Bu uygulamanın amacı..." value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} /></Field>
                    <div className="form-row">
                      <Field label="Ana sayfa URL"><input placeholder="https://app.test" value={draft.homeUrl} onChange={(event) => setDraft({ ...draft, homeUrl: event.target.value })} /></Field>
                      <Field label="Logo URL" hint="Giriş sayfasında logo olarak gösterilir."><input placeholder="https://app.test/logo.png" value={draft.logoUrl} onChange={(event) => setDraft({ ...draft, logoUrl: event.target.value })} /></Field>
                    </div>
                    <Field label="İzinli redirect URI değerleri">
                      <textarea placeholder="http://localhost:3000/callback, https://app.test/oauth/callback" value={draft.redirectUris} onChange={(event) => setDraft({ ...draft, redirectUris: event.target.value })} required />
                    </Field>
                  </>
                ) : (
                  <>
                    <div className="secret-panel">
                      <div>
                        <label>Client secret</label>
                        <p>Secret sadece güvenilir backend clientları tarafından kullanılmalı.</p>
                      </div>
                      <div className="secret-row">
                        <input className="mono-input" value={draft.clientSecret} onChange={(event) => setDraft({ ...draft, clientSecret: event.target.value })} required />
                        <button type="button" className="btn-tertiary" onClick={() => setDraft({ ...draft, clientSecret: randomSecret() })}><Shuffle size={14} />Üret</button>
                        <button type="button" className="btn-secondary" onClick={copySecret} disabled={!draft.clientSecret}><Copy size={14} />{copied ? 'Kopyalandı' : 'Kopyala'}</button>
                      </div>
                    </div>
                    <div className="form-row">
                      <Field label="Token (oturum) ömrü — dakika" hint="0 = sunucu varsayılanı"><input type="number" min={0} value={Math.round(draft.tokenTtlSeconds / 60)} onChange={(event) => setDraft({ ...draft, tokenTtlSeconds: Number(event.target.value) * 60 })} /></Field>
                      <Field label="Yenileme (refresh) ömrü — dakika" hint="0 = sunucu varsayılanı"><input type="number" min={0} value={Math.round(draft.refreshTtlSeconds / 60)} onChange={(event) => setDraft({ ...draft, refreshTtlSeconds: Number(event.target.value) * 60 })} /></Field>
                    </div>
                  </>
                )}
              </div>
            </div>
            <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Clientı kaydet' : 'Client oluştur'} />
          </form>
        ) : (
          <>
            <div className="modal-body">
              <div className="modal-form">
                {saveRole.error && <Alert type="error" message={saveRole.error.message} />}
                {removeRole.error && <Alert type="error" message={removeRole.error.message} />}
                {(currentClient?.roles?.length ?? 0) === 0 ? (
                  <Muted>Bu client için rol yok.</Muted>
                ) : (
                  <div className="data-table-container compact-table">
                    <table className="carbon-table">
                      <thead>
                        <tr><th>Rol</th><th>Açıklama</th><th></th></tr>
                      </thead>
                      <tbody>
                        {currentClient?.roles?.map((role) => (
                          <tr key={role.ID} onClick={() => startRoleEdit(role)}>
                            <td><strong>{role.name}</strong></td>
                            <td>{role.description || 'Açıklama girilmemiş.'}</td>
                            <td>
                              <Actions
                                onEdit={(event) => { event.stopPropagation(); startRoleEdit(role) }}
                                onDelete={(event) => { event.stopPropagation(); if (confirm(`${role.name} client rolü silinsin mi?`)) removeRole.mutate(role) }}
                              />
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                <form className="role-editor-panel" onSubmit={(event) => { event.preventDefault(); saveRole.mutate() }}>
                  <div className="form-row">
                    <Field label="Rol adı"><input placeholder="örn. reader" value={roleDraft.name} onChange={(event) => setRoleDraft({ ...roleDraft, name: event.target.value })} required /></Field>
                    <Field label="Açıklama"><input placeholder="Rolün client içindeki yetki kapsamı" value={roleDraft.description} onChange={(event) => setRoleDraft({ ...roleDraft, description: event.target.value })} /></Field>
                  </div>
                  <div className="role-editor-actions">
                    {editRole && <button type="button" className="btn-secondary" onClick={resetRoleEditor}>Vazgeç</button>}
                    <button type="submit" className="btn-primary" disabled={saveRole.isPending}>{saveRole.isPending ? 'Kaydediliyor...' : editRole ? 'Rolü kaydet' : 'Client rolü ekle'}</button>
                  </div>
                </form>
              </div>
            </div>
            <div className="modal-footer"><button type="button" className="btn-primary" onClick={closeModal}>Kapat</button></div>
          </>
        )}
      </Modal>
    </div>
  )
}

function clientDraft(): ClientDraft {
  return { clientId: '', clientSecret: randomSecret(), name: '', description: '', homeUrl: '', logoUrl: '', redirectUris: '', tokenTtlSeconds: 0, refreshTtlSeconds: 0 }
}

function fromClient(client: Client): ClientDraft {
  return {
    clientId: client.clientId,
    clientSecret: client.clientSecret,
    name: client.name,
    description: client.description ?? '',
    homeUrl: client.homeUrl ?? '',
    logoUrl: client.logoUrl ?? '',
    redirectUris: client.redirectUris,
    tokenTtlSeconds: client.tokenTtlSeconds ?? 0,
    refreshTtlSeconds: client.refreshTtlSeconds ?? 0,
  }
}

function formatDuration(seconds: number) {
  if (!seconds) return 'Varsayılan'
  if (seconds % 86400 === 0) return `${seconds / 86400}g`
  if (seconds % 3600 === 0) return `${seconds / 3600}sa`
  if (seconds % 60 === 0) return `${seconds / 60}dk`
  return `${seconds}sn`
}
