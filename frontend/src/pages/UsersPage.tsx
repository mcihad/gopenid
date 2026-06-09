import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Ban, Plus, ShieldCheck } from 'lucide-react'
import { api } from '../lib/api'
import type { Client, User } from '../lib/types'
import { Alert, CheckRow, ControlBar, EmptySearch, Field, Modal, ModalFooter, Muted, PageHeader, Picker, SearchInput, Status, TableState, Tag } from '../components/ui'
import { formatDate, lower } from '../lib/format'

type UserModalTab = 'general' | 'organization' | 'clients'

type UserDraft = {
  name: string
  email: string
  password: string
  phone: string
  title: string
  avatarUrl: string
  active: boolean
  departmentId?: number
  roleIds: number[]
  clientIds: number[]
  clientRoleIds: number[]
  departmentIds: number[]
  groupIds: number[]
}

export function UsersPage() {
  const qc = useQueryClient()
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const departments = useQuery({ queryKey: ['departments'], queryFn: api.departments.list })
  const roles = useQuery({ queryKey: ['roles'], queryFn: api.roles.list })
  const groups = useQuery({ queryKey: ['groups'], queryFn: api.groups.list })
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<User | null>(null)
  const [draft, setDraft] = useState<UserDraft>(userDraft())
  const [searchQuery, setSearchQuery] = useState('')
  const [modalTab, setModalTab] = useState<UserModalTab>('general')

  const invalidate = () => qc.invalidateQueries({ queryKey: ['users'] })

  const save = useMutation({
    mutationFn: () => (edit ? api.users.update(edit.ID, draft) : api.users.create(draft)),
    onSuccess: () => { invalidate(); closeModal() },
  })
  const remove = useMutation({ mutationFn: (id: number) => api.users.remove(id), onSuccess: invalidate })
  const block = useMutation({
    mutationFn: (user: User) => {
      if (user.blocked) return api.users.unblock(user.ID)
      const reason = prompt('Engelleme nedeni (opsiyonel):') ?? ''
      return api.users.block(user.ID, reason)
    },
    onSuccess: invalidate,
  })

  const filteredUsers = useMemo(() => {
    const q = lower(searchQuery)
    return (users.data ?? []).filter((user) =>
      lower(user.name).includes(q) || lower(user.email).includes(q) || lower(user.title).includes(q) ||
      lower(user.department?.name).includes(q) || user.roles?.some((role) => lower(role.name).includes(q)) ||
      user.groups?.some((group) => lower(group.name).includes(q)),
    )
  }, [users.data, searchQuery])

  const openCreateModal = () => { setEdit(null); setDraft(userDraft()); save.reset(); setModalTab('general'); setIsModalOpen(true) }
  const openEditModal = (user: User) => { setEdit(user); setDraft(fromUser(user)); save.reset(); setModalTab('general'); setIsModalOpen(true) }
  const closeModal = () => { setIsModalOpen(false); setEdit(null); setDraft(userDraft()) }

  return (
    <div className="directory-section">
      <PageHeader eyebrow="Kurumsal kimlik dizini" title="Kullanıcılar" />
      <ControlBar onRefresh={() => users.refetch()} action={<button className="btn-primary" onClick={openCreateModal}><Plus size={14} />Kullanıcı ekle</button>}>
        <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Kullanıcı, e-posta, ünvan, departman, grup veya rol ara..." />
      </ControlBar>

      <TableState query={users} />
      {users.data && users.data.length > 0 && filteredUsers.length === 0 && <EmptySearch text={`"${searchQuery}" için kullanıcı bulunamadı.`} />}

      {filteredUsers.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Kullanıcı</th><th>E-posta</th><th>Departman</th><th>Gruplar</th><th>Roller</th><th>Son giriş</th><th>Durum</th><th></th></tr>
            </thead>
            <tbody>
              {filteredUsers.map((user) => (
                <tr key={user.ID} onClick={() => openEditModal(user)}>
                  <td>
                    <strong>{user.name}</strong>
                    {user.title && <div className="muted-text">{user.title}</div>}
                  </td>
                  <td className="mono-text">{user.email}</td>
                  <td>{user.department?.name || <Muted>Yok</Muted>}</td>
                  <td>{user.groups?.length ? user.groups.map((group) => <Tag key={group.ID}>{group.name}</Tag>) : <Muted>Yok</Muted>}</td>
                  <td>{user.roles?.length ? user.roles.map((role) => <Tag key={role.ID}>{role.name}</Tag>) : <Muted>Yok</Muted>}</td>
                  <td>{formatDate(user.lastLoginAt)}</td>
                  <td><Status active={user.active} blocked={user.blocked} /></td>
                  <td>
                    <div className="action-link-group">
                      <button type="button" className="action-link" onClick={(event) => { event.stopPropagation(); openEditModal(user) }}>Düzenle</button>
                      <button type="button" className={`action-link ${user.blocked ? '' : 'danger'}`} onClick={(event) => { event.stopPropagation(); block.mutate(user) }}>{user.blocked ? 'Engeli kaldır' : 'Engelle'}</button>
                      <button type="button" className="action-link danger" onClick={(event) => { event.stopPropagation(); if (confirm('Kullanıcı silinsin mi?')) remove.mutate(user.ID) }}>Sil</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal wide isOpen={isModalOpen} onClose={closeModal} title={edit ? `Kullanıcı düzenle: ${edit.name}` : 'Yeni kullanıcı'}>
        <div className="modal-tabs">
          <button type="button" className={`modal-tab ${modalTab === 'general' ? 'active' : ''}`} onClick={() => setModalTab('general')}>Bilgiler</button>
          <button type="button" className={`modal-tab ${modalTab === 'organization' ? 'active' : ''}`} onClick={() => setModalTab('organization')}>Organizasyon</button>
          <button type="button" className={`modal-tab ${modalTab === 'clients' ? 'active' : ''}`} onClick={() => setModalTab('clients')}>Client yetkileri</button>
        </div>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            {save.error && <Alert type="error" message={save.error.message} />}
            {edit?.blocked && <Alert type="error" message={`Bu kullanıcı engelli${edit.blockedReason ? `: ${edit.blockedReason}` : '.'}`} />}

            {modalTab === 'general' && (
              <div className="modal-form">
                <div className="form-row">
                  <Field label="Ad soyad"><input placeholder="Ayşe Yılmaz" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required /></Field>
                  <Field label="E-posta"><input type="email" placeholder="ayse@firma.test" value={draft.email} onChange={(event) => setDraft({ ...draft, email: event.target.value })} required /></Field>
                </div>
                <div className="form-row">
                  <Field label="Ünvan"><input placeholder="örn. Yazılım Mühendisi" value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} /></Field>
                  <Field label="Telefon"><input placeholder="+90 5xx xxx xx xx" value={draft.phone} onChange={(event) => setDraft({ ...draft, phone: event.target.value })} /></Field>
                </div>
                <div className="form-row">
                  <Field label="Parola" hint={edit ? 'Değişmeyecekse boş bırakın' : undefined}><input placeholder={edit ? '••••••••' : '••••••••'} type="password" value={draft.password} onChange={(event) => setDraft({ ...draft, password: event.target.value })} required={!edit} /></Field>
                  <Field label="Birincil departman">
                    <select value={draft.departmentId ?? ''} onChange={(event) => setDraft({ ...draft, departmentId: event.target.value ? Number(event.target.value) : undefined })}>
                      <option value="">Departman yok</option>
                      {departments.data?.map((department) => <option key={department.ID} value={department.ID}>{department.name}</option>)}
                    </select>
                  </Field>
                </div>
                <Field label="Avatar URL"><input placeholder="https://..." value={draft.avatarUrl} onChange={(event) => setDraft({ ...draft, avatarUrl: event.target.value })} /></Field>
                <label className="checkbox-group"><input type="checkbox" checked={draft.active} onChange={(event) => setDraft({ ...draft, active: event.target.checked })} />Aktif hesap</label>
              </div>
            )}

            {modalTab === 'organization' && (
              <div className="modal-form">
                <Picker title="Departmanlar (çoklu)" empty="Önce Departmanlar bölümünden departman ekleyin." items={departments.data ?? []} selectedIDs={draft.departmentIds} onToggle={(id) => setDraft(toggleID(draft, 'departmentIds', id))} />
                <Picker title="Kullanıcı grupları" empty="Önce Gruplar bölümünden grup ekleyin." items={groups.data ?? []} selectedIDs={draft.groupIds} onToggle={(id) => setDraft(toggleID(draft, 'groupIds', id))} />
                <Picker title="Sistem rolleri" empty="Önce Roller > Sistem rolleri bölümünden rol ekleyin." items={roles.data ?? []} selectedIDs={draft.roleIds} onToggle={(id) => setDraft(toggleID(draft, 'roleIds', id))} />
              </div>
            )}

            {modalTab === 'clients' && (
              <div className="modal-form">
                <Picker title="Yetkili clientlar" empty="Önce Clientlar bölümünden client ekleyin." items={clients.data ?? []} selectedIDs={draft.clientIds} label={(client) => `${client.name} (${client.clientId})`} onToggle={(id) => setDraft(toggleClient(draft, id, clients.data ?? []))} />
                {draft.clientIds.length > 0 && (
                  <div className="form-group">
                    <label>Client rolleri</label>
                    <div className="roles-grid expanded">
                      {clients.data?.filter((client) => draft.clientIds.includes(client.ID)).map((client) => (
                        <div className="client-role-group" key={client.ID}>
                          <strong>{client.name} <span className="mono-text">{client.clientId}</span></strong>
                          {(client.roles?.length ?? 0) === 0 ? <Muted>Bu client için rol yok.</Muted> : client.roles?.map((role) => (
                            <CheckRow key={role.ID} selected={draft.clientRoleIds.includes(role.ID)} label={role.name} onClick={() => setDraft(toggleID(draft, 'clientRoleIds', role.ID))} />
                          ))}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Kullanıcıyı kaydet' : 'Kullanıcı oluştur'} />
        </form>
      </Modal>

      <div className="legend-row">
        <span><ShieldCheck size={13} /> Engelli olmayan aktif kullanıcılar giriş yapabilir.</span>
        <span><Ban size={13} /> Engellenen kullanıcıların oturumları otomatik iptal edilir.</span>
      </div>
    </div>
  )
}

function userDraft(): UserDraft {
  return { name: '', email: '', password: '', phone: '', title: '', avatarUrl: '', active: true, departmentId: undefined, roleIds: [], clientIds: [], clientRoleIds: [], departmentIds: [], groupIds: [] }
}

function fromUser(user: User): UserDraft {
  return {
    name: user.name,
    email: user.email,
    password: '',
    phone: user.phone ?? '',
    title: user.title ?? '',
    avatarUrl: user.avatarUrl ?? '',
    active: user.active,
    departmentId: user.department?.ID ?? user.departmentId,
    roleIds: user.roles?.map((role) => role.ID) ?? [],
    clientIds: user.authorizedClients?.map((client) => client.ID) ?? [],
    clientRoleIds: user.clientRoles?.map((role) => role.ID) ?? [],
    departmentIds: user.departments?.map((dept) => dept.ID) ?? [],
    groupIds: user.groups?.map((group) => group.ID) ?? [],
  }
}

type ListKey = 'roleIds' | 'clientIds' | 'clientRoleIds' | 'departmentIds' | 'groupIds'

function toggleID(draft: UserDraft, key: ListKey, id: number): UserDraft {
  const ids = draft[key].includes(id) ? draft[key].filter((itemID) => itemID !== id) : [...draft[key], id]
  return { ...draft, [key]: ids }
}

// toggleClient also clears the client's roles when the client is deselected.
function toggleClient(draft: UserDraft, id: number, clients: Client[]): UserDraft {
  const next = toggleID(draft, 'clientIds', id)
  const client = clients.find((item) => item.ID === id)
  if (draft.clientIds.includes(id) && client?.roles?.length) {
    next.clientRoleIds = next.clientRoleIds.filter((roleID) => !client.roles?.some((role) => role.ID === roleID))
  }
  return next
}
