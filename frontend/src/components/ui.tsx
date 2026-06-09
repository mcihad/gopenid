import { AlertCircle, Check, RefreshCw, Search, X } from 'lucide-react'

// PageHeader renders the eyebrow + title block shown at the top of every page.
export function PageHeader({ eyebrow, title, action }: { eyebrow: string; title: string; action?: React.ReactNode }) {
  return (
    <header className="page-header">
      <div className="page-header-text">
        <p>{eyebrow}</p>
        <h2>{title}</h2>
      </div>
      <div className="page-header-actions">
        <div className="status-pill"><span className="status-dot-pulse" />Sunucu aktif</div>
        {action}
      </div>
    </header>
  )
}

// Modal renders a centered dialog with a header and arbitrary children.
export function Modal({ isOpen, onClose, title, children, wide }: { isOpen: boolean; onClose: () => void; title: string; children: React.ReactNode; wide?: boolean }) {
  if (!isOpen) return null
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className={`modal-content ${wide ? 'wide' : ''}`} onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <h2>{title}</h2>
          <button className="modal-close" onClick={onClose} aria-label="Pencereyi kapat"><X size={16} /></button>
        </div>
        {children}
      </div>
    </div>
  )
}

export function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="form-group">
      <label>{label}</label>
      <div className="input-container no-icon">{children}</div>
      {hint && <span className="field-hint">{hint}</span>}
    </div>
  )
}

export function ModalFooter({ onCancel, pending, submitText }: { onCancel: () => void; pending: boolean; submitText: string }) {
  return (
    <div className="modal-footer">
      <button type="button" className="btn-secondary" onClick={onCancel}>Vazgeç</button>
      <button type="submit" className="btn-primary" disabled={pending}>{pending ? 'Kaydediliyor...' : submitText}</button>
    </div>
  )
}

export function SearchInput({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder: string }) {
  return <div className="search-box-wrap"><Search size={14} /><input type="text" placeholder={placeholder} value={value} onChange={(event) => onChange(event.target.value)} /></div>
}

export function ControlBar({ children, onRefresh, action }: { children?: React.ReactNode; onRefresh: () => void; action: React.ReactNode }) {
  return (
    <div className="control-bar">
      <div className="control-left">{children}</div>
      <div className="control-right">
        <button className="btn-secondary" onClick={onRefresh}><RefreshCw size={14} />Yenile</button>
        {action}
      </div>
    </div>
  )
}

export function CheckRow({ selected, label, onClick }: { selected: boolean; label: string; onClick: () => void }) {
  return <div className={`role-chip-card ${selected ? 'selected' : ''}`} onClick={onClick}><span className="checkbox-custom">{selected && <Check size={12} />}</span><span>{label}</span></div>
}

export function Picker<T extends { ID: number; name: string }>({ title, empty, items, selectedIDs, onToggle, label }: { title: string; empty: string; items: T[]; selectedIDs: number[]; onToggle: (id: number) => void; label?: (item: T) => string }) {
  return (
    <div className="form-group">
      <label>{title}</label>
      {items.length === 0 ? <Muted>{empty}</Muted> : <div className="roles-grid">{items.map((item) => <CheckRow key={item.ID} selected={selectedIDs.includes(item.ID)} label={label ? label(item) : item.name} onClick={() => onToggle(item.ID)} />)}</div>}
    </div>
  )
}

export function Actions({ onEdit, onDelete }: { onEdit: React.MouseEventHandler<HTMLButtonElement>; onDelete: React.MouseEventHandler<HTMLButtonElement> }) {
  return <div className="action-link-group"><button type="button" className="action-link" onClick={onEdit}>Düzenle</button><button type="button" className="action-link danger" onClick={onDelete}>Sil</button></div>
}

export function Alert({ type, message }: { type: 'error' | 'info' | 'success'; message: string }) {
  return <div className={`alert-box ${type}`}><AlertCircle size={16} /><span>{message}</span></div>
}

export function EmptySearch({ text }: { text: string }) {
  return <div className="state-empty"><Search size={24} /><strong>Eşleşen kayıt yok</strong><p>{text}</p></div>
}

export function Status({ active, blocked }: { active: boolean; blocked?: boolean }) {
  if (blocked) return <span className="status-badge blocked">Engelli</span>
  return <span className={active ? 'status-badge active' : 'status-badge disabled'}>{active ? 'Aktif' : 'Pasif'}</span>
}

export function TableState({ query }: { query: { isLoading: boolean; error: Error | null; data?: unknown[] } }) {
  if (query.isLoading) return <div className="state-empty"><RefreshCw size={20} className="animate-spin" /><strong>Veriler yükleniyor</strong><p>Sunucu ile bağlantı kuruluyor.</p></div>
  if (query.error) return <div className="state-empty error-state"><AlertCircle size={24} /><strong>Bağlantı hatası</strong><p>{query.error.message || 'Bağlantı ayarlarını kontrol edin.'}</p></div>
  if (!query.data?.length) return <div className="state-empty"><AlertCircle size={24} /><strong>Kayıt yok</strong><p>Bu bölümde henüz kayıt oluşturulmamış.</p></div>
  return null
}

export function Muted({ children }: { children: React.ReactNode }) {
  return <span className="muted-text">{children}</span>
}

export function Tag({ children, primary }: { children: React.ReactNode; primary?: boolean }) {
  return <span className={`tag-badge ${primary ? 'primary' : ''}`}>{children}</span>
}
