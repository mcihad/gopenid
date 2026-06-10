import { Component, useEffect, useState } from 'react'
import type { ErrorInfo, ReactNode } from 'react'
import { AlertCircle, X } from 'lucide-react'

type Toast = { id: number; message: string; type: 'error' | 'info' | 'success' }

export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('render failed', error, info)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="state-empty error-state app-error-boundary">
          <AlertCircle size={24} />
          <strong>Beklenmeyen hata</strong>
          <p>{this.state.error.message || 'Sayfa yüklenirken bir sorun oluştu.'}</p>
          <button className="btn-primary" onClick={() => this.setState({ error: null })}>Tekrar dene</button>
        </div>
      )
    }
    return this.props.children
  }
}

export function ToastHost() {
  const [items, setItems] = useState<Toast[]>([])

  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent<Omit<Toast, 'id'>>).detail
      const toast = { ...detail, id: Date.now() }
      setItems((current) => [...current, toast].slice(-3))
      window.setTimeout(() => setItems((current) => current.filter((item) => item.id !== toast.id)), 4500)
    }
    window.addEventListener('gopenid:toast', handler)
    return () => window.removeEventListener('gopenid:toast', handler)
  }, [])

  if (items.length === 0) return null
  return (
    <div className="toast-stack">
      {items.map((item) => (
        <div key={item.id} className={`toast-item ${item.type}`}>
          <AlertCircle size={16} />
          <span>{item.message}</span>
          <button type="button" onClick={() => setItems((current) => current.filter((toast) => toast.id !== item.id))} aria-label="Bildirimi kapat"><X size={14} /></button>
        </div>
      ))}
    </div>
  )
}
