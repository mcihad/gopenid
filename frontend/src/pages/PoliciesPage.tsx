import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import type { Policy, PolicyAssignment, PolicySubject } from '../lib/types'
import { Actions, Alert, CheckRow, ControlBar, EmptySearch, Field, Modal, ModalFooter, Muted, PageHeader, SearchInput, TableState, Tag } from '../components/ui'
import { lower } from '../lib/format'

const weekdays = ['Paz', 'Pzt', 'Sal', 'Çar', 'Per', 'Cum', 'Cmt']

type PolicyDraft = {
  name: string
  description: string
  type: 'ip' | 'time'
  effect: 'allow' | 'deny'
  ipCidrs: string
  daysOfWeek: number[]
  startTime: string
  endTime: string
}

export function PoliciesPage() {
  const qc = useQueryClient()
  const policies = useQuery({ queryKey: ['policies'], queryFn: api.policies.list })
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [edit, setEdit] = useState<Policy | null>(null)
  const [draft, setDraft] = useState<PolicyDraft>(policyDraft())
  const [searchQuery, setSearchQuery] = useState('')
  const [assignFor, setAssignFor] = useState<Policy | null>(null)

  const save = useMutation({
    mutationFn: () => (edit ? api.policies.update(edit.ID, draft) : api.policies.create(draft)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['policies'] }); closeModal() },
  })
  const remove = useMutation({
    mutationFn: (id: number) => api.policies.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['policies'] }),
  })

  const filtered = useMemo(() => {
    const q = lower(searchQuery)
    return (policies.data ?? []).filter((policy) => lower(policy.name).includes(q) || lower(policy.description).includes(q))
  }, [policies.data, searchQuery])

  const openCreateModal = () => { setEdit(null); setDraft(policyDraft()); save.reset(); setIsModalOpen(true) }
  const openEditModal = (policy: Policy) => { setEdit(policy); setDraft(fromPolicy(policy)); save.reset(); setIsModalOpen(true) }
  const closeModal = () => { setIsModalOpen(false); setEdit(null); setDraft(policyDraft()) }

  return (
    <div className="directory-section">
      <PageHeader eyebrow="Güvenlik" title="Giriş politikaları" />
      <ControlBar onRefresh={() => policies.refetch()} action={<button className="btn-primary" onClick={openCreateModal}><Plus size={14} />Politika ekle</button>}>
        <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="Politika ara..." />
      </ControlBar>

      <p className="section-note">Politikalar IP veya tarih/saat koşuluna göre giriş izni verir ya da reddeder. Hiyerarşi: <strong>kullanıcı &gt; grup &gt; uygulama</strong> (en özel seviye kazanır).</p>

      <TableState query={policies} />
      {policies.data && policies.data.length > 0 && filtered.length === 0 && <EmptySearch text={`"${searchQuery}" için politika bulunamadı.`} />}

      {filtered.length > 0 && (
        <div className="data-table-container">
          <table className="carbon-table">
            <thead>
              <tr><th>Politika</th><th>Tür</th><th>Etki</th><th>Koşul</th><th>Atamalar</th><th></th></tr>
            </thead>
            <tbody>
              {filtered.map((policy) => (
                <tr key={policy.ID} onClick={() => openEditModal(policy)}>
                  <td><strong>{policy.name}</strong>{policy.description && <div className="muted-text">{policy.description}</div>}</td>
                  <td><Tag>{policy.type === 'ip' ? 'IP' : 'Zaman'}</Tag></td>
                  <td><span className={`status-badge ${policy.effect === 'allow' ? 'active' : 'blocked'}`}>{policy.effect === 'allow' ? 'İzin ver' : 'Reddet'}</span></td>
                  <td className="mono-text">{conditionSummary(policy)}</td>
                  <td><button type="button" className="action-link" onClick={(event) => { event.stopPropagation(); setAssignFor(policy) }}>Atamaları yönet</button></td>
                  <td>
                    <Actions
                      onEdit={(event) => { event.stopPropagation(); openEditModal(policy) }}
                      onDelete={(event) => { event.stopPropagation(); if (confirm('Politika silinsin mi?')) remove.mutate(policy.ID) }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal isOpen={isModalOpen} onClose={closeModal} title={edit ? `Politika düzenle: ${edit.name}` : 'Yeni politika'}>
        <form onSubmit={(event) => { event.preventDefault(); save.mutate() }}>
          <div className="modal-body">
            <div className="modal-form">
              {save.error && <Alert type="error" message={save.error.message} />}
              <Field label="Ad"><input placeholder="örn. Mesai saatleri" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} required /></Field>
              <Field label="Açıklama"><input placeholder="Politikanın amacı" value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} /></Field>
              <div className="form-row">
                <Field label="Tür">
                  <select value={draft.type} onChange={(event) => setDraft({ ...draft, type: event.target.value as 'ip' | 'time' })}>
                    <option value="time">Tarih / Saat</option>
                    <option value="ip">IP adresi</option>
                  </select>
                </Field>
                <Field label="Etki">
                  <select value={draft.effect} onChange={(event) => setDraft({ ...draft, effect: event.target.value as 'allow' | 'deny' })}>
                    <option value="allow">İzin ver</option>
                    <option value="deny">Reddet</option>
                  </select>
                </Field>
              </div>

              {draft.type === 'ip' ? (
                <Field label="IP / CIDR listesi" hint="Virgülle ayırın. Örn: 203.0.113.0/24, 198.51.100.7">
                  <textarea placeholder="10.0.0.0/24, 192.168.1.10" value={draft.ipCidrs} onChange={(event) => setDraft({ ...draft, ipCidrs: event.target.value })} required />
                </Field>
              ) : (
                <>
                  <div className="form-group">
                    <label>Günler <span className="field-hint-inline">(boş = her gün)</span></label>
                    <div className="weekday-grid">
                      {weekdays.map((day, index) => (
                        <CheckRow key={index} selected={draft.daysOfWeek.includes(index)} label={day} onClick={() => setDraft(toggleDay(draft, index))} />
                      ))}
                    </div>
                  </div>
                  <div className="form-row">
                    <Field label="Başlangıç saati"><input type="time" value={draft.startTime} onChange={(event) => setDraft({ ...draft, startTime: event.target.value })} /></Field>
                    <Field label="Bitiş saati"><input type="time" value={draft.endTime} onChange={(event) => setDraft({ ...draft, endTime: event.target.value })} /></Field>
                  </div>
                </>
              )}
            </div>
          </div>
          <ModalFooter onCancel={closeModal} pending={save.isPending} submitText={edit ? 'Politikayı kaydet' : 'Oluştur'} />
        </form>
      </Modal>

      {assignFor && <AssignmentsModal policy={assignFor} onClose={() => setAssignFor(null)} />}
    </div>
  )
}

function AssignmentsModal({ policy, onClose }: { policy: Policy; onClose: () => void }) {
  const qc = useQueryClient()
  const assignments = useQuery({ queryKey: ['assignments', policy.ID], queryFn: () => api.policies.assignments.list(policy.ID) })
  const users = useQuery({ queryKey: ['users'], queryFn: api.users.list })
  const groups = useQuery({ queryKey: ['groups'], queryFn: api.groups.list })
  const clients = useQuery({ queryKey: ['clients'], queryFn: api.clients.list })
  const [subjectType, setSubjectType] = useState<PolicySubject>('client')
  const [subjectId, setSubjectId] = useState<number>(0)

  const invalidate = () => qc.invalidateQueries({ queryKey: ['assignments', policy.ID] })
  const create = useMutation({
    mutationFn: () => api.policies.assignments.create(policy.ID, { subjectType, subjectId }),
    onSuccess: () => { invalidate(); setSubjectId(0) },
  })
  const remove = useMutation({
    mutationFn: (assignmentId: number) => api.policies.assignments.remove(policy.ID, assignmentId),
    onSuccess: invalidate,
  })

  const options = subjectType === 'client' ? (clients.data ?? []).map((c) => ({ id: c.ID, label: `${c.name} (${c.clientId})` }))
    : subjectType === 'group' ? (groups.data ?? []).map((g) => ({ id: g.ID, label: g.name }))
      : (users.data ?? []).map((u) => ({ id: u.ID, label: `${u.name} — ${u.email}` }))

  const subjectLabel = (assignment: PolicyAssignment) => {
    if (assignment.subjectType === 'client') return clients.data?.find((c) => c.ID === assignment.subjectId)?.name ?? `#${assignment.subjectId}`
    if (assignment.subjectType === 'group') return groups.data?.find((g) => g.ID === assignment.subjectId)?.name ?? `#${assignment.subjectId}`
    return users.data?.find((u) => u.ID === assignment.subjectId)?.name ?? `#${assignment.subjectId}`
  }
  const subjectTypeLabel = (t: PolicySubject) => (t === 'client' ? 'Uygulama' : t === 'group' ? 'Grup' : 'Kullanıcı')

  return (
    <Modal isOpen onClose={onClose} title={`Atamalar: ${policy.name}`}>
      <div className="modal-body">
        <div className="modal-form">
          {create.error && <Alert type="error" message={create.error.message} />}
          <div className="assign-row">
            <select value={subjectType} onChange={(event) => { setSubjectType(event.target.value as PolicySubject); setSubjectId(0) }}>
              <option value="client">Uygulama</option>
              <option value="group">Grup</option>
              <option value="user">Kullanıcı</option>
            </select>
            <select value={subjectId} onChange={(event) => setSubjectId(Number(event.target.value))}>
              <option value={0}>Seçin...</option>
              {options.map((opt) => <option key={opt.id} value={opt.id}>{opt.label}</option>)}
            </select>
            <button type="button" className="btn-primary" disabled={!subjectId || create.isPending} onClick={() => create.mutate()}><Plus size={14} />Ata</button>
          </div>

          {assignments.isLoading ? <Muted>Yükleniyor...</Muted> : (assignments.data?.length ?? 0) === 0 ? <Muted>Bu politika henüz hiçbir özneye atanmadı.</Muted> : (
            <div className="data-table-container compact-table">
              <table className="carbon-table">
                <thead><tr><th>Tür</th><th>Özne</th><th></th></tr></thead>
                <tbody>
                  {assignments.data?.map((assignment) => (
                    <tr key={assignment.ID}>
                      <td><Tag>{subjectTypeLabel(assignment.subjectType)}</Tag></td>
                      <td>{subjectLabel(assignment)}</td>
                      <td><button type="button" className="action-link danger" onClick={() => remove.mutate(assignment.ID)}><Trash2 size={13} /> Kaldır</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
      <div className="modal-footer"><button type="button" className="btn-primary" onClick={onClose}>Kapat</button></div>
    </Modal>
  )
}

function conditionSummary(policy: Policy) {
  if (policy.type === 'ip') return policy.ipCidrs || '—'
  const days = policy.daysOfWeek && policy.daysOfWeek.length > 0 ? policy.daysOfWeek.map((d) => weekdays[d]).join(',') : 'Her gün'
  const range = policy.startTime && policy.endTime ? `${policy.startTime}-${policy.endTime}` : 'Tüm gün'
  return `${days} · ${range}`
}

function policyDraft(): PolicyDraft {
  return { name: '', description: '', type: 'time', effect: 'allow', ipCidrs: '', daysOfWeek: [], startTime: '08:00', endTime: '18:00' }
}

function fromPolicy(policy: Policy): PolicyDraft {
  return {
    name: policy.name,
    description: policy.description,
    type: policy.type,
    effect: policy.effect,
    ipCidrs: policy.ipCidrs ?? '',
    daysOfWeek: policy.daysOfWeek ?? [],
    startTime: policy.startTime ?? '',
    endTime: policy.endTime ?? '',
  }
}

function toggleDay(draft: PolicyDraft, day: number): PolicyDraft {
  const days = draft.daysOfWeek.includes(day) ? draft.daysOfWeek.filter((d) => d !== day) : [...draft.daysOfWeek, day].sort((a, b) => a - b)
  return { ...draft, daysOfWeek: days }
}
