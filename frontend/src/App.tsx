import reactLogo from './assets/react.svg'
import viteLogo from './assets/vite.svg'
import './App.css'
import { Button } from '@/components/ui/button'
import { useAppStore } from '@/store'

function App() {
  const { count, increment, decrement } = useAppStore()

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center gap-8">
      <div className="flex gap-4">
        <a href="https://vite.dev" target="_blank" rel="noreferrer">
          <img src={viteLogo} className="logo" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank" rel="noreferrer">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1 className="text-4xl font-bold text-foreground">Vite + React + Shadcn UI</h1>
      <div className="card flex flex-col items-center gap-4">
        <div className="flex items-center gap-4">
          <Button variant="outline" onClick={decrement}>
            -
          </Button>
          <span className="text-2xl font-mono">{count}</span>
          <Button onClick={increment}>
            +
          </Button>
        </div>
        <p className="text-muted-foreground mt-4">
          Edit <code>src/App.tsx</code> and save to test HMR
        </p>
      </div>
    </div>
  )
}

export default App
