import { Button } from './button'
import { AlertTriangle } from 'lucide-react'

interface ConfirmDialogProps {
  isOpen: boolean
  title: string
  message: string
  onConfirm: () => void
  onCancel: () => void
  confirmText?: string
  cancelText?: string
  isDestructive?: boolean
}

export function ConfirmDialog({
  isOpen,
  title,
  message,
  onConfirm,
  onCancel,
  confirmText = '确定',
  cancelText = '取消',
  isDestructive = true,
}: ConfirmDialogProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in-0">
      <div className="bg-white rounded-xl shadow-lg w-full max-w-md p-6 animate-in zoom-in-95 relative overflow-hidden">
        <div className="flex items-start gap-4">
          <div className={`p-2 rounded-full shrink-0 ${isDestructive ? 'bg-red-100 text-red-600' : 'bg-indigo-100 text-indigo-600'}`}>
            <AlertTriangle className="w-6 h-6" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-zinc-900 mb-2">{title}</h3>
            <p className="text-sm text-zinc-500 mb-6 leading-relaxed">{message}</p>
            <div className="flex items-center justify-end gap-3">
              <Button variant="outline" onClick={onCancel}>
                {cancelText}
              </Button>
              <Button 
                variant={isDestructive ? 'destructive' : 'default'} 
                onClick={onConfirm}
                className={!isDestructive ? 'bg-indigo-600 hover:bg-indigo-700 text-white' : ''}
              >
                {confirmText}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
