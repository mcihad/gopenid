import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertCircle,
  Building2,
  Check,
  Copy,
  ExternalLink,
  KeyRound,
  LogOut,
  Plus,
  RefreshCw,
  Search,
  Shield,
  Shuffle,
  Users,
  X,
} from 'lucide-react'
import { api, auth, login } from './api'
import type { Client, ClientRole, Resource, User } from './types'

type Tab = 'users' | 'departments' | 'roles' | 'clients'
type RoleTab = 'global' | 'client'

const tabs = [
  { id: 'users' as Tab, label: 'Kullanıcılar', icon: Users },
  { id: 'departments' as Tab, label: 'Departmanlar', icon: Building2 },
  { id: 'roles' as Tab, label: 'Roller', icon: Shield },
  { id: 'clients' as Tab, label: 'Clientlar', icon: KeyRound },
]

function App() {
  const [loggedIn, setLoggedIn] = useState(auth.isTokenValid())
  const [tab, setTab] = useState<Tab>('users')

  useEffect(() => {
    const syncSession = () => setLoggedIn(auth.isTokenValid())
    const timer = window.setInterval(syncSession, 30_000)
    window.addEventListener('focus', syncSession)
    window.addEventListener('gopenid:logout', syncSession)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', syncSession)
      window.removeEventListener('gopenid:logout', syncSession)
    }
  }, [])

  if (!loggedIn) return <Login onLogin={() => setLoggedIn(true)} />

  return (
    <div className="app-container">
      <header className="top-nav">
        <div className="brand-section">
          <div className="logo-tag"><KeyRound size={14} />gOpenID</div>
          <div className="brand-title">Kimlik Sunucusu<span>Yönetim</span></div>
        </div>

        <div className="nav-tabs-wrapper">
          {tabs.map((item) => (
            <button key={item.id} className={`product-tab ${tab === item.id ? 'active' : ''}`} onClick={() => setTab(item.id)}>
              <item.icon size={14} />
              {item.label}
            </button>
          ))}
        </div>

        <div className="nav-actions">
          <a className="nav-link-meta" href="/.well-known/openid-configuration" target="_blank" rel="noreferrer">
            <ExternalLink size={14} />
            OIDC keşfi
          </a>
          <button className="btn-signout" onClick={() => { auth.clear(); setLoggedIn(false) }}>
            <LogOut size={14} />
            Çıkış
          </button>
        </div>
      </header>

      <main className="workspace">
        <Header tab={tab} />
        <Overview />
        {tab === 'users' && <UsersPanel />}
        {tab === 'departments' && <ResourcePanel kind="departments" title="Departmanlar" single="Departman" />}
        {tab === 'roles' && <RolesPanel />}
        {tab === 'clients' && <ClientsPanel />}
      </main>
    </div>
  )
}

function Login({ onLogin }: { onLogin: () => void }) {
  const [email, setEmail] = useState('admin@gopenid.local')
  const [password, setPassword] = useState('admin12345')
  const mutation = useMutation({
    mutationFn: () => login(email, password),
    onSuccess: onLogin,
  })

  return (
    <main className="login">
      <form onSubmit={(event) => { event.preventDefault(); mutation.mutate() }}>
        <div className="login-top">
          <div className="logo-tag"><KeyRound size={16} />gOpenID</div>
          <div>
            <h1>Yönetim konsoluna giriş</h1>
            <p>Kullanıcı, rol, client ve OIDC yetkilendirme yönetimi.</p>
          </div>
        </div>

        {mutation.error && <Alert type="error" message={mutation.error.message} />}

        <Field label="E-posta">
          <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="admin@gopenid.local" required />
        </Field>
        <Field label="Parola">
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="••••••••" required />
        </Field>

        <button className="btn-primary" disabled={mutation.isPending}>
          {mutation.isPending ? 'Giriş yapılıyor...' : 'Giriş yap'}
        </button>
      </form>
    </main>
  )
}

function Header({ tab }: { tab: Tab }) {
  const title = tab === 'users' ? 'Kullanıcılar' : tab === 'roles' ? 'Roller' : tab === 'departments' ? 'Departmanlar' : 'OIDC clientlar'
  return (
    <header className="page-header">
      <div className="page-header-text">
        <p>Kurumsal kimlik dizini</p>
        <h2>{title}</h2>
      </div>
      <div className="status-pill"><span className="status-dot-pulse" />Sunucu aktif</div>
    </header>
  )
}

function Overview() {
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const departments = useQuery({ queryKey: ['departments'], queryFn: api.departments.list })
  const roles = useQuery({ queryKey: ['roles'], queryFn: api.roles.list })
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const activeUsers = users.data?.filter((user) => user.active).length ?? 0
  const clientRoles = clients.data?.reduce((total, client) => total + (client.roles?.length ?? 0), 0) ?? 0

  return (
    <section className="metrics-grid">
      <Metric icon={Users} value={activeUsers} label="Aktif kullanıcı" />
      <Metric icon={Building2} value={departments.data?.length ?? 0} label="Departman" />
      <Metric icon={Shield} value={(roles.data?.length ?? 0) + clientRoles} label="Toplam rol" />
      <Metric icon={KeyRound} value={clients.data?.length ?? 0} label="OIDC client" />
    </section>
  )
}

function Metric({ icon: Icon, value, label }: { icon: typeof Users; value: number; label: string }) {
  return (
    <article className="metric-card">
      <Icon size={20} />
      <strong>{value}</strong>
      <p>{label}</p>
    </article>
  )
}

function RolesPanel() {
  const [roleTab, setRoleTab] = useState<RoleTab>('global')
  return (
    <div className="directory-section">
      <div className="section-tabs">
        <button className={`section-tab ${roleTab === 'global' ? 'active' : ''}`} onClick={() => setRoleTab('global')}>Sistem rolleri</button>
        <button className={`section-tab ${roleTab === 'client' ? 'active' : ''}`} onClick={() => setRoleTab('client')}>Client rolleri</button>
      </div>
      {roleTab === 'global' ? <ResourcePanel kind="roles" title="Sistem rolleri" single="Rol" embedded /> : <ClientRolesPanel />}
    </div>
  )
}

function ResourcePanel({ kind, title, single, embedded = false }: { kind: 'departments' | 'roles'; title: string; single: string; embedded?: boolean }) {
  const qc = useQueryClient()
  const resource = api[kind]
  const query = useQuery({ queryKey: [kind], queryFn: resource.list })
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<Resource | null>(null)
  const [draft, setDraft] = useState({ name: '', description: '' })
  const [searchQuery, setSearchQuery] = useState('')

  const save = useMutation({
    mutationFn: () => edit ? resource.update(edit.ID, draft) : resource.create(draft),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [kind] })
      closeModal()
    },
  })
  const remove = useMutation({
    mutationFn: (id: number) => resource.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: [kind] }),
  })

  const filteredData = useMemo(() => {
    const q = searchQuery.toLocaleLowerCase('tr')
    return (query.data ?? []).filter((row) => row.name.toLocaleLowerCase('tr').includes(q) || row.description?.toLocaleLowerCase('tr').includes(q))
  }, [query.data, searchQuery])

  const openCreateModal = () => {
    setEdit(null)
    setDraft({ name: '', description: '' })
    save.reset()
    setIsModalOpen(true)
  }
  const openEditModal = (row: Resource) => {
    setEdit(row)
    setDraft({ name: row.name, description: row.description })
    save.reset()
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft({ name: '', description: '' })
  }
  const getUserCount = (id: number) => {
    if (kind === 'departments') return users.data?.filter((user) => user.departmentId === id).length ?? 0
    return users.data?.filter((user) => user.roles?.some((role) => role.ID === id)).length ?? 0
  }

  return (
    <div className={embedded ? '' : 'directory-section'}>
      <div className="control-bar">
        <div className="control-left">
          <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder={`${title} içinde ara...`} />
        </div>
        <div className="control-right">
          <button className="btn-secondary" onClick={() => query.refetch()}><RefreshCw size={14} />Yenile</button>
          <button className="btn-primary" onClick={openCreateModal}><Plus size={14} />{single} ekle</button>
        </div>
      </div>

      <TableState query={query} />

      {query.data && query.data.length > 0 && filteredData.length === 0 && <EmptySearch text={`"${searchQuery}" için kayıt bulunamadı.`} />}

      {filteredData.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr>
                <th>Ad</th>
                <th>Açıklama</th>
                <th>Atanan kullanıcı</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {filteredData.map((row) => (
                <tr key={row.ID} onClick={() => openEditModal(row)}>
                  <td><strong>{row.name}</strong></td>
                  <td>{row.description || 'Açıklama girilmemiş.'}</td>
                  <td>{getUserCount(row.ID)}</td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(row) }}
                      onDelete={(event) => {
                        event.stopPropagation()
                        if (confirm(`${single} silinsin mi?`)) remove.mutate(row.ID)
                      }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `${single} düzenle: ${edit.name}` : `Yeni ${single}`}>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            <div className="modal-form">
              {save.error && <Alert type="error" message={save.error.message} />}
              <Field label="Ad">
                <input placeholder={single === 'Rol' ? 'örn. admin' : 'örn. Mühendislik'} value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required />
              </Field>
              <Field label="Açıklama">
                <textarea placeholder="Bu kaydın kullanım amacını yazın..." value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
              </Field>
            </div>
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Değişiklikleri kaydet' : 'Oluştur'} />
        </form>
      </Modal>
    </div>
  )
}

function ClientRolesPanel() {
  const qc = useQueryClient()
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const [selectedClientID, setSelectedClientID] = useState<number | 'all'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<{ client: Client; role: ClientRole } | null>(null)
  const [draft, setDraft] = useState({ name: '', description: '', clientID: 0 })

  const roles = useMemo(() => {
    const q = searchQuery.toLocaleLowerCase('tr')
    return (clients.data ?? [])
      .filter((client) => selectedClientID === 'all' || client.ID === selectedClientID)
      .flatMap((client) => (client.roles ?? []).map((role) => ({ client, role })))
      .filter(({ client, role }) => role.name.toLocaleLowerCase('tr').includes(q) || role.description?.toLocaleLowerCase('tr').includes(q) || client.name.toLocaleLowerCase('tr').includes(q) || client.clientId.toLocaleLowerCase('tr').includes(q))
  }, [clients.data, selectedClientID, searchQuery])

  const save = useMutation({
    mutationFn: () => {
      if (edit) return api.clients.roles.update(edit.client.ID, edit.role.ID, { name: draft.name, description: draft.description })
      return api.clients.roles.create(draft.clientID, { name: draft.name, description: draft.description })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      closeModal()
    },
  })
  const remove = useMutation({
    mutationFn: ({ clientID, roleID }: { clientID: number; roleID: number }) => api.clients.roles.remove(clientID, roleID),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      qc.invalidateQueries({ queryKey: ['users'] })
    },
  })

  const openCreateModal = () => {
    const firstClientID = selectedClientID === 'all' ? clients.data?.[0]?.ID ?? 0 : selectedClientID
    setEdit(null)
    setDraft({ name: '', description: '', clientID: firstClientID })
    save.reset()
    setIsModalOpen(true)
  }
  const openEditModal = (client: Client, role: ClientRole) => {
    setEdit({ client, role })
    setDraft({ name: role.name, description: role.description, clientID: client.ID })
    save.reset()
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft({ name: '', description: '', clientID: 0 })
  }

  return (
    <div>
      <div className="control-bar">
        <div className="control-left role-filters">
          <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Client rolü veya client ara..." />
          <select value={selectedClientID} onChange={(event) => setSelectedClientID(event.target.value === 'all' ? 'all' : Number(event.target.value))}>
            <option value="all">Tüm clientlar</option>
            {clients.data?.map((client) => <option key={client.ID} value={client.ID}>{client.name} ({client.clientId})</option>)}
          </select>
        </div>
        <div className="control-right">
          <button className="btn-secondary" onClick={() => clients.refetch()}><RefreshCw size={14} />Yenile</button>
          <button className="btn-primary" onClick={openCreateModal} disabled={!clients.data?.length}><Plus size={14} />Client rolü ekle</button>
        </div>
      </div>

      <TableState query={clients} />
      {clients.data && clients.data.length > 0 && roles.length === 0 && <EmptySearch text="Bu filtrelerle eşleşen client rolü yok." />}

      {roles.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr>
                <th>Rol</th>
                <th>Client</th>
                <th>Açıklama</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {roles.map(({ client, role }) => (
                <tr key={`${client.ID}-${role.ID}`} onClick={() => openEditModal(client, role)}>
                  <td><strong>{role.name}</strong></td>
                  <td><span className="tag-badge primary">{client.name}</span><span className="mono-text">{client.clientId}</span></td>
                  <td>{role.description || 'Açıklama girilmemiş.'}</td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(client, role) }}
                      onDelete={(event) => {
                        event.stopPropagation()
                        if (confirm(`${role.name} client rolü silinsin mi?`)) remove.mutate({ clientID: client.ID, roleID: role.ID })
                      }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `Client rolü düzenle: ${edit.role.name}` : 'Yeni client rolü'}>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            <div className="modal-form">
              {save.error && <Alert type="error" message={save.error.message} />}
              <Field label="Client">
                <select value={draft.clientID} onChange={(event) => setDraft({ ...draft, clientID: Number(event.target.value) })} disabled={Boolean(edit)} required>
                  <option value={0}>Client seçin</option>
                  {clients.data?.map((client) => <option key={client.ID} value={client.ID}>{client.name} ({client.clientId})</option>)}
                </select>
              </Field>
              <Field label="Rol adı">
                <input placeholder="örn. reader" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required />
              </Field>
              <Field label="Açıklama">
                <textarea placeholder="Rolün client içindeki yetki kapsamı..." value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
              </Field>
            </div>
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Değişiklikleri kaydet' : 'Oluştur'} />
        </form>
      </Modal>
    </div>
  )
}

function UsersPanel() {
  const qc = useQueryClient()
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const departments = useQuery({ queryKey: ['departments'], queryFn: api.departments.list })
  const roles = useQuery({ queryKey: ['roles'], queryFn: api.roles.list })
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<User | null>(null)
  const [draft, setDraft] = useState(userDraft())
  const [searchQuery, setSearchQuery] = useState('')
  const [userModalTab, setUserModalTab] = useState<'general' | 'clients'>('general')

  const save = useMutation({
    mutationFn: () => edit ? api.users.update(edit.ID, draft) : api.users.create(draft),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users'] })
      closeModal()
    },
  })
  const remove = useMutation({
    mutationFn: (id: number) => api.users.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })

  const filteredUsers = useMemo(() => {
    const q = searchQuery.toLocaleLowerCase('tr')
    return (users.data ?? []).filter((user) => user.name.toLocaleLowerCase('tr').includes(q) || user.email.toLocaleLowerCase('tr').includes(q) || user.department?.name?.toLocaleLowerCase('tr').includes(q) || user.roles?.some((role) => role.name.toLocaleLowerCase('tr').includes(q)))
  }, [users.data, searchQuery])

  const openCreateModal = () => {
    setEdit(null)
    setDraft(userDraft())
    save.reset()
    setUserModalTab('general')
    setIsModalOpen(true)
  }
  const openEditModal = (user: User) => {
    setEdit(user)
    setDraft(fromUser(user))
    save.reset()
    setUserModalTab('general')
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft(userDraft())
  }

  return (
    <div className="directory-section">
      <div className="control-bar">
        <div className="control-left">
          <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Kullanıcı, e-posta, departman veya rol ara..." />
        </div>
        <div className="control-right">
          <button className="btn-secondary" onClick={() => users.refetch()}><RefreshCw size={14} />Yenile</button>
          <button className="btn-primary" onClick={openCreateModal}><Plus size={14} />Kullanıcı ekle</button>
        </div>
      </div>

      <TableState query={users} />
      {users.data && users.data.length > 0 && filteredUsers.length === 0 && <EmptySearch text={`"${searchQuery}" için kullanıcı bulunamadı.`} />}

      {filteredUsers.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr>
                <th>Kullanıcı</th>
                <th>E-posta</th>
                <th>Departman</th>
                <th>Sistem rolleri</th>
                <th>Clientlar</th>
                <th>Durum</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((user) => (
                <tr key={user.ID} onClick={() => openEditModal(user)}>
                  <td><strong>{user.name}</strong></td>
                  <td className="mono-text">{user.email}</td>
                  <td>{user.department?.name || 'Departman yok'}</td>
                  <td>{user.roles?.length ? user.roles.map((role) => <span className="tag-badge" key={role.ID}>{role.name}</span>) : <Muted>Yok</Muted>}</td>
                  <td>{user.authorizedClients?.length ? user.authorizedClients.map((client) => <span className="tag-badge primary" key={client.ID}>{client.name}</span>) : <Muted>Yok</Muted>}</td>
                  <td><Status active={user.active} /></td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(user) }}
                      onDelete={(event) => {
                        event.stopPropagation()
                        if (confirm('Kullanıcı silinsin mi?')) remove.mutate(user.ID)
                      }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `Kullanıcı düzenle: ${edit.name}` : 'Yeni kullanıcı'}>
        <div className="modal-tabs">
          <button type="button" className={`modal-tab ${userModalTab === 'general' ? 'active' : ''}`} onClick={() => setUserModalTab('general')}>Kullanıcı bilgileri</button>
          <button type="button" className={`modal-tab ${userModalTab === 'clients' ? 'active' : ''}`} onClick={() => setUserModalTab('clients')}>Client yetkileri</button>
        </div>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            {save.error && <Alert type="error" message={save.error.message} />}
            {userModalTab === 'general' ? (
              <div className="modal-form">
                <div className="form-row">
                  <Field label="Ad soyad"><input placeholder="Ayşe Yılmaz" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required /></Field>
                  <Field label="E-posta"><input type="email" placeholder="ayse@firma.test" value={draft.email} onChange={(event) => setDraft({ ...draft, email: event.target.value })} required /></Field>
                </div>
                <div className="form-row">
                  <Field label="Parola"><input placeholder={edit ? 'Değişmeyecekse boş bırakın' : '••••••••'} type="password" value={draft.password} onChange={(event) => setDraft({ ...draft, password: event.target.value })} required={!edit} /></Field>
                  <Field label="Departman">
                    <select value={draft.departmentId ?? ''} onChange={(event) => setDraft({ ...draft, departmentId: event.target.value ? Number(event.target.value) : undefined })}>
                      <option value="">Departman yok</option>
                      {departments.data?.map((department) => <option key={department.ID} value={department.ID}>{department.name}</option>)}
                    </select>
                  </Field>
                </div>
                <label className="checkbox-group"><input type="checkbox" checked={draft.active} onChange={(event) => setDraft({ ...draft, active: event.target.checked })} />Aktif hesap</label>
                <RolePicker title="Sistem rolleri" empty="Önce Roller > Sistem rolleri bölümünden rol ekleyin." items={roles.data ?? []} selectedIDs={draft.roleIds} onToggle={(id) => setDraft(toggleID(draft, 'roleIds', id))} />
              </div>
            ) : (
              <div className="modal-form">
                <RolePicker title="Yetkili clientlar" empty="Önce Clientlar bölümünden client ekleyin." items={clients.data ?? []} selectedIDs={draft.clientIds} label={(client) => `${client.name} (${client.clientId})`} onToggle={(id) => {
                  const client = clients.data?.find((item) => item.ID === id)
                  const next = toggleID(draft, 'clientIds', id)
                  if (draft.clientIds.includes(id) && client?.roles?.length) {
                    next.clientRoleIds = next.clientRoleIds.filter((roleID) => !client.roles?.some((role) => role.ID === roleID))
                  }
                  setDraft(next)
                }} />
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
    </div>
  )
}

function ClientsPanel() {
  const qc = useQueryClient()
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<Client | null>(null)
  const [draft, setDraft] = useState({ clientId: '', clientSecret: '', name: '', redirectUris: '' })
  const [modalTab, setModalTab] = useState<'general' | 'roles'>('general')
  const [roleDraft, setRoleDraft] = useState({ name: '', description: '' })
  const [editRole, setEditRole] = useState<ClientRole | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [copied, setCopied] = useState(false)

  const save = useMutation({
    mutationFn: () => edit ? api.clients.update(edit.ID, draft) : api.clients.create(draft),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      closeModal()
    },
  })
  const remove = useMutation({
    mutationFn: (id: number) => api.clients.remove(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      qc.invalidateQueries({ queryKey: ['users'] })
    },
  })
  const saveRole = useMutation({
    mutationFn: () => {
      if (!currentClient) throw new Error('Client seçili değil.')
      if (editRole) return api.clients.roles.update(currentClient.ID, editRole.ID, roleDraft)
      return api.clients.roles.create(currentClient.ID, roleDraft)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      resetRoleEditor()
    },
  })
  const removeRole = useMutation({
    mutationFn: (role: ClientRole) => {
      if (!currentClient) throw new Error('Client seçili değil.')
      return api.clients.roles.remove(currentClient.ID, role.ID)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      qc.invalidateQueries({ queryKey: ['users'] })
      resetRoleEditor()
    },
  })

  const filteredClients = useMemo(() => {
    const q = searchQuery.toLocaleLowerCase('tr')
    return (clients.data ?? []).filter((client) => client.name.toLocaleLowerCase('tr').includes(q) || client.clientId.toLocaleLowerCase('tr').includes(q))
  }, [clients.data, searchQuery])
  const currentClient = edit ? clients.data?.find((client) => client.ID === edit.ID) ?? edit : null
  const getUserCount = (clientID: number) => users.data?.filter((user) => user.authorizedClients?.some((client) => client.ID === clientID)).length ?? 0
  const startRoleEdit = (role: ClientRole) => {
    setEditRole(role)
    setRoleDraft({ name: role.name, description: role.description })
    saveRole.reset()
  }
  const resetRoleEditor = () => {
    setEditRole(null)
    setRoleDraft({ name: '', description: '' })
    saveRole.reset()
  }

  const openCreateModal = () => {
    setEdit(null)
    setDraft({ clientId: '', clientSecret: randomSecret(), name: '', redirectUris: '' })
    setModalTab('general')
    resetRoleEditor()
    save.reset()
    setCopied(false)
    setIsModalOpen(true)
  }
  const openEditModal = (client: Client) => {
    setEdit(client)
    setDraft({ clientId: client.clientId, clientSecret: client.clientSecret, name: client.name, redirectUris: client.redirectUris })
    setModalTab('general')
    resetRoleEditor()
    save.reset()
    setCopied(false)
    setIsModalOpen(true)
  }
  const closeModal = () => {
    setIsModalOpen(false)
    setEdit(null)
    setDraft({ clientId: '', clientSecret: '', name: '', redirectUris: '' })
    setModalTab('general')
    resetRoleEditor()
  }
  const copySecret = async () => {
    await navigator.clipboard.writeText(draft.clientSecret)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <div className="directory-section">
      <div className="control-bar">
        <div className="control-left"><SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Client adı veya client ID ara..." /></div>
        <div className="control-right">
          <button className="btn-secondary" onClick={() => clients.refetch()}><RefreshCw size={14} />Yenile</button>
          <button className="btn-primary" onClick={openCreateModal}><Plus size={14} />Client ekle</button>
        </div>
      </div>

      <TableState query={clients} />
      {clients.data && clients.data.length > 0 && filteredClients.length === 0 && <EmptySearch text={`"${searchQuery}" için client bulunamadı.`} />}

      {filteredClients.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr>
                <th>Client</th>
                <th>Client ID</th>
                <th>Secret</th>
                <th>Redirect URI</th>
                <th>Roller</th>
                <th>Kullanıcı</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {filteredClients.map((client) => (
                <tr key={client.ID} onClick={() => openEditModal(client)}>
                  <td><strong>{client.name}</strong></td>
                  <td className="mono-text">{client.clientId}</td>
                  <td className="mono-text secret-preview">{maskSecret(client.clientSecret)}</td>
                  <td>{client.redirectUris.split(',').map((uri) => <div className="mono-text" key={uri}>{uri.trim()}</div>)}</td>
                  <td>{client.roles?.length ?? 0}</td>
                  <td>{getUserCount(client.ID)}</td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(client) }}
                      onDelete={(event) => {
                        event.stopPropagation()
                        if (confirm('Client silinsin mi?')) remove.mutate(client.ID)
                      }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `Client düzenle: ${edit.name}` : 'Yeni OIDC client'}>
        {currentClient && (
          <div className="modal-tabs">
            <button type="button" className={`modal-tab ${modalTab === 'general' ? 'active' : ''}`} onClick={() => setModalTab('general')}>Client bilgileri</button>
            <button type="button" className={`modal-tab ${modalTab === 'roles' ? 'active' : ''}`} onClick={() => setModalTab('roles')}>Client rolleri ({currentClient.roles?.length ?? 0})</button>
          </div>
        )}

        {modalTab === 'general' ? (
          <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
            <div className="modal-body">
              <div className="modal-form">
                {save.error && <Alert type="error" message={save.error.message} />}
                <div className="form-row">
                  <Field label="Client adı"><input placeholder="örn. Web uygulaması" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required /></Field>
                  <Field label="Client ID"><input placeholder="örn. web-app" value={draft.clientId} onChange={(event) => setDraft({ ...draft, clientId: event.target.value })} required disabled={Boolean(edit)} /></Field>
                </div>
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
                <Field label="İzinli redirect URI değerleri">
                  <textarea placeholder="http://localhost:3000/callback, https://app.test/oauth/callback" value={draft.redirectUris} onChange={(event) => setDraft({ ...draft, redirectUris: event.target.value })} required />
                </Field>
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
                        <tr>
                          <th>Rol</th>
                          <th>Açıklama</th>
                          <th></th>
                        </tr>
                      </thead>
                      <tbody>
                        {currentClient?.roles?.map((role) => (
                          <tr key={role.ID} onClick={() => startRoleEdit(role)}>
                            <td><strong>{role.name}</strong></td>
                            <td>{role.description || 'Açıklama girilmemiş.'}</td>
                            <td>
                              <Actions
                                onEdit={(event) => { event.stopPropagation(); startRoleEdit(role) }}
                                onDelete={(event) => {
                                  event.stopPropagation()
                                  if (confirm(`${role.name} client rolü silinsin mi?`)) removeRole.mutate(role)
                                }}
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
            <div className="modal-footer">
              <button type="button" className="btn-primary" onClick={closeModal}>Kapat</button>
            </div>
          </>
        )}
      </Modal>
    </div>
  )
}

function Modal({ isOpen, onClose, title, children }: { isOpen: boolean; onClose: () => void; title: string; children: React.ReactNode }) {
  if (!isOpen) return null
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <h2>{title}</h2>
          <button className="modal-close" onClick={onClose} aria-label="Pencereyi kapat"><X size={16} /></button>
        </div>
        {children}
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="form-group"><label>{label}</label><div className="input-container no-icon">{children}</div></div>
}

function ModalFooter({ onCancel, pending, submitText }: { onCancel: () => void; pending: boolean; submitText: string }) {
  return (
    <div className="modal-footer">
      <button type="button" className="btn-secondary" onClick={onCancel}>Vazgeç</button>
      <button type="submit" className="btn-primary" disabled={pending}>{pending ? 'Kaydediliyor...' : submitText}</button>
    </div>
  )
}

function SearchInput({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder: string }) {
  return <div className="search-box-wrap"><Search size={14} /><input type="text" placeholder={placeholder} value={value} onChange={(event) => onChange(event.target.value)} /></div>
}

function RolePicker<T extends { ID: number; name: string }>({ title, empty, items, selectedIDs, onToggle, label }: { title: string; empty: string; items: T[]; selectedIDs: number[]; onToggle: (id: number) => void; label?: (item: T) => string }) {
  return (
    <div className="form-group">
      <label>{title}</label>
      {items.length === 0 ? <Muted>{empty}</Muted> : <div className="roles-grid">{items.map((item) => <CheckRow key={item.ID} selected={selectedIDs.includes(item.ID)} label={label ? label(item) : item.name} onClick={() => onToggle(item.ID)} />)}</div>}
    </div>
  )
}

function CheckRow({ selected, label, onClick }: { selected: boolean; label: string; onClick: () => void }) {
  return <div className={`role-chip-card ${selected ? 'selected' : ''}`} onClick={onClick}><span className="checkbox-custom">{selected && <Check size={12} />}</span><span>{label}</span></div>
}

function Actions({ onEdit, onDelete }: { onEdit: React.MouseEventHandler<HTMLButtonElement>; onDelete: React.MouseEventHandler<HTMLButtonElement> }) {
  return <div className="action-link-group"><button type="button" className="action-link" onClick={onEdit}>Düzenle</button><button type="button" className="action-link danger" onClick={onDelete}>Sil</button></div>
}

function Alert({ type, message }: { type: 'error' | 'info'; message: string }) {
  return <div className={`alert-box ${type}`}><AlertCircle size={16} /><span>{message}</span></div>
}

function EmptySearch({ text }: { text: string }) {
  return <div className="state-empty"><Search size={24} /><strong>Eşleşen kayıt yok</strong><p>{text}</p></div>
}

function Status({ active }: { active: boolean }) {
  return <span className={active ? 'status-badge active' : 'status-badge disabled'}>{active ? 'Aktif' : 'Pasif'}</span>
}

function TableState({ query }: { query: { isLoading: boolean; error: Error | null; data?: unknown[] } }) {
  if (query.isLoading) return <div className="state-empty"><RefreshCw size={20} className="animate-spin" /><strong>Veriler yükleniyor</strong><p>Sunucu ile bağlantı kuruluyor.</p></div>
  if (query.error) return <div className="state-empty error-state"><AlertCircle size={24} /><strong>Bağlantı hatası</strong><p>{query.error.message || 'Bağlantı ayarlarını kontrol edin.'}</p></div>
  if (!query.data?.length) return <div className="state-empty"><AlertCircle size={24} /><strong>Kayıt yok</strong><p>Bu bölümde henüz kayıt oluşturulmamış.</p></div>
  return null
}

function Muted({ children }: { children: React.ReactNode }) {
  return <span className="muted-text">{children}</span>
}

function userDraft() {
  return { name: '', email: '', password: '', active: true, departmentId: undefined as number | undefined, roleIds: [] as number[], clientIds: [] as number[], clientRoleIds: [] as number[] }
}

function fromUser(user: User) {
  return {
    name: user.name,
    email: user.email,
    password: '',
    active: user.active,
    departmentId: user.department?.ID,
    roleIds: user.roles?.map((role) => role.ID) ?? [],
    clientIds: user.authorizedClients?.map((client) => client.ID) ?? [],
    clientRoleIds: user.clientRoles?.map((role) => role.ID) ?? [],
  }
}

function toggleID<T extends ReturnType<typeof userDraft>>(draft: T, key: 'roleIds' | 'clientIds' | 'clientRoleIds', id: number): T {
  const ids = draft[key].includes(id) ? draft[key].filter((itemID) => itemID !== id) : [...draft[key], id]
  return { ...draft, [key]: ids }
}

function randomSecret() {
  const bytes = new Uint8Array(32)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

function maskSecret(secret: string) {
  if (secret.length <= 10) return '••••••'
  return `${secret.slice(0, 4)}••••••••${secret.slice(-4)}`
}

export default App
