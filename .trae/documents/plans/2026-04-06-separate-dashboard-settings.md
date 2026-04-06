# Separate Dashboard and Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Separate the current combined Settings page into two distinct routes (`/dashboard` and `/settings`) with their own Sidebar navigation items.

**Architecture:** We will extract the "Dashboard" tab content from `Settings.tsx` into a new `Dashboard.tsx` page component. We will then update `App.tsx` to add the new route, and `Sidebar.tsx` to display both navigation buttons at the bottom.

**Tech Stack:** React 18, React Router DOM, Tailwind CSS, Shadcn UI, Zustand, Lucide React

---

### Task 1: Create Dashboard Page Component

**Files:**
- Create: `frontend/src/pages/Dashboard.tsx`

- [ ] **Step 1: Write the Dashboard component**

Create the file `frontend/src/pages/Dashboard.tsx` with the extracted Dashboard content:

```tsx
import { useEffect, useState } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePlatformStore } from '@/store/platformStore'
import { toast } from 'sonner'
import { RefreshCw } from 'lucide-react'

export function Dashboard() {
  const { publications, fetchPlatforms, syncStats } = usePlatformStore()
  const [isSyncing, setIsSyncing] = useState(false)

  useEffect(() => {
    fetchPlatforms()
    syncStats()
  }, [fetchPlatforms, syncStats])

  const handleSync = async () => {
    setIsSyncing(true)
    try {
      await syncStats()
      toast.success('数据同步成功')
    } catch (err: any) {
      toast.error('同步失败')
    } finally {
      setIsSyncing(false)
    }
  }

  // 计算总数据
  const totalViews = publications.reduce((acc, p) => acc + p.views, 0)
  const totalLikes = publications.reduce((acc, p) => acc + p.likes, 0)
  const totalComments = publications.reduce((acc, p) => acc + p.comments, 0)

  return (
    <div className="h-screen overflow-hidden bg-zinc-50 flex">
      <Sidebar />
      <div className="flex-1 overflow-auto p-8 max-w-5xl mx-auto">
        <h1 className="text-3xl font-bold mb-8">数据大盘</h1>
        
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-zinc-500">总浏览量</CardTitle></CardHeader>
            <CardContent><div className="text-3xl font-bold">{totalViews}</div></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-zinc-500">总点赞量</CardTitle></CardHeader>
            <CardContent><div className="text-3xl font-bold">{totalLikes}</div></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-zinc-500">总评论量</CardTitle></CardHeader>
            <CardContent><div className="text-3xl font-bold">{totalComments}</div></CardContent>
          </Card>
        </div>
        
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold">发布记录</h2>
          <Button variant="outline" size="sm" onClick={handleSync} disabled={isSyncing}>
            <RefreshCw className={`w-4 h-4 mr-2 ${isSyncing ? 'animate-spin' : ''}`} />
            同步数据
          </Button>
        </div>
        
        <div className="border rounded-md bg-white">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>文章 ID</TableHead>
                <TableHead>平台</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">浏览量</TableHead>
                <TableHead className="text-right">点赞量</TableHead>
                <TableHead className="text-right">评论量</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {publications.length === 0 ? (
                <TableRow><TableCell colSpan={6} className="text-center text-zinc-500 py-8">暂无发布记录</TableCell></TableRow>
              ) : (
                publications.map(p => (
                  <TableRow key={p.id}>
                    <TableCell className="font-mono text-xs">{p.blog_id.slice(0, 8)}...</TableCell>
                    <TableCell className="capitalize">{p.platform_type}</TableCell>
                    <TableCell>{p.status === 1 ? '已发布' : '草稿'}</TableCell>
                    <TableCell className="text-right">{p.views}</TableCell>
                    <TableCell className="text-right">{p.likes}</TableCell>
                    <TableCell className="text-right">{p.comments}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/Dashboard.tsx
git commit -m "feat(frontend): create standalone Dashboard page"
```

---

### Task 2: Refactor Settings Page Component

**Files:**
- Modify: `frontend/src/pages/Settings.tsx`

- [ ] **Step 1: Remove Dashboard logic from Settings.tsx**

Modify `frontend/src/pages/Settings.tsx` to remove Dashboard tab and related variables (`isSyncing`, `totalViews`, `totalLikes`, `totalComments`, `handleSync`).

Search for and replace the component code to strictly focus on Integrations and Account:

```tsx
import { useEffect, useState } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { usePlatformStore } from '@/store/platformStore'
import { toast } from 'sonner'
import { RefreshCw, Link as LinkIcon, Unlink } from 'lucide-react'

export function Settings() {
  const { platforms, fetchPlatforms, bindPlatform, unbindPlatform } = usePlatformStore()
  const [cookieInput, setCookieInput] = useState('')
  const [isBinding, setIsBinding] = useState(false)
  const [dialogOpen, setDialogOpen] = useState<string | null>(null)

  useEffect(() => {
    fetchPlatforms()
  }, [fetchPlatforms])

  const handleBind = async (platform: string) => {
    if (!cookieInput.trim()) {
      toast.error('请输入 Cookie')
      return
    }
    setIsBinding(true)
    try {
      await bindPlatform(platform, cookieInput)
      toast.success(`成功绑定 ${platform}`)
      setDialogOpen(null)
      setCookieInput('')
    } catch (err: any) {
      toast.error(err.message)
    } finally {
      setIsBinding(false)
    }
  }

  const handleUnbind = async (platform: string) => {
    if (!window.confirm(`确定要解绑 ${platform} 吗？`)) return
    try {
      await unbindPlatform(platform)
      toast.success(`已解绑 ${platform}`)
    } catch (err: any) {
      toast.error(err.message)
    }
  }

  return (
    <div className="h-screen overflow-hidden bg-zinc-50 flex">
      <Sidebar />
      <div className="flex-1 overflow-auto p-8 max-w-5xl mx-auto">
        <h1 className="text-3xl font-bold mb-8">设置中心</h1>
        
        <Tabs defaultValue="integrations" className="w-full">
          <TabsList className="mb-6">
            <TabsTrigger value="integrations">平台授权</TabsTrigger>
            <TabsTrigger value="account">账号设置</TabsTrigger>
          </TabsList>
          
          <TabsContent value="integrations">
            <Card>
              <CardHeader>
                <CardTitle>第三方平台授权</CardTitle>
                <CardDescription>
                  由于官方 OpenAPI 限制，目前采用手动填入浏览器 Cookie 的方式进行授权。您的 Cookie 会在服务端加密存储，并在使用时自动解密调用。
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {platforms.map(p => (
                  <div key={p.platform_type} className="flex items-center justify-between p-4 border rounded-lg bg-white">
                    <div className="flex items-center gap-4">
                      <div className={`w-10 h-10 rounded-full flex items-center justify-center text-white font-bold capitalize ${p.platform_type === 'juejin' ? 'bg-blue-500' : 'bg-orange-500'}`}>
                        {p.platform_type.charAt(0)}
                      </div>
                      <div>
                        <h3 className="font-semibold capitalize">{p.platform_type}</h3>
                        <p className="text-sm text-zinc-500">
                          状态：{p.is_bound ? <span className="text-green-600 font-medium">已绑定</span> : <span className="text-zinc-400">未绑定</span>}
                        </p>
                      </div>
                    </div>
                    
                    <div>
                      {p.is_bound ? (
                        <div className="flex gap-2">
                          <Dialog open={dialogOpen === p.platform_type} onOpenChange={(open) => { setDialogOpen(open ? p.platform_type : null); setCookieInput(''); }}>
                            <DialogTrigger asChild>
                              <Button variant="outline" size="sm"><RefreshCw className="w-4 h-4 mr-2" />更新 Cookie</Button>
                            </DialogTrigger>
                            <DialogContent>
                              <DialogHeader>
                                <DialogTitle>更新 {p.platform_type} Cookie</DialogTitle>
                                <DialogDescription>请粘贴您在浏览器中获取的最新 Cookie 字符串。</DialogDescription>
                              </DialogHeader>
                              <Textarea value={cookieInput} onChange={e => setCookieInput(e.target.value)} placeholder="session_id=xxx; ..." rows={5} />
                              <DialogFooter>
                                <Button onClick={() => handleBind(p.platform_type)} disabled={isBinding}>保存</Button>
                              </DialogFooter>
                            </DialogContent>
                          </Dialog>
                          <Button variant="destructive" size="sm" onClick={() => handleUnbind(p.platform_type)}><Unlink className="w-4 h-4 mr-2" />解绑</Button>
                        </div>
                      ) : (
                        <Dialog open={dialogOpen === p.platform_type} onOpenChange={(open) => { setDialogOpen(open ? p.platform_type : null); setCookieInput(''); }}>
                          <DialogTrigger asChild>
                            <Button size="sm"><LinkIcon className="w-4 h-4 mr-2" />绑定</Button>
                          </DialogTrigger>
                          <DialogContent>
                            <DialogHeader>
                              <DialogTitle>绑定 {p.platform_type}</DialogTitle>
                              <DialogDescription>请粘贴您在浏览器中获取的 Cookie 字符串。</DialogDescription>
                            </DialogHeader>
                            <Textarea value={cookieInput} onChange={e => setCookieInput(e.target.value)} placeholder="session_id=xxx; ..." rows={5} />
                            <DialogFooter>
                              <Button onClick={() => handleBind(p.platform_type)} disabled={isBinding}>保存绑定</Button>
                            </DialogFooter>
                          </DialogContent>
                        </Dialog>
                      )}
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="account">
            <Card>
              <CardHeader>
                <CardTitle>账号设置</CardTitle>
                <CardDescription>您的账号基本信息与 Token 消耗情况。</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-zinc-500">正在开发中...</p>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/Settings.tsx
git commit -m "refactor(frontend): remove dashboard from settings page"
```

---

### Task 3: Update Routing in App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add Dashboard Route**

Update `frontend/src/App.tsx` to import `Dashboard` and map it to `/dashboard`.

```tsx
import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'sonner'
import { Home } from '@/pages/Home'
import { Settings } from '@/pages/Settings'
import { Dashboard } from '@/pages/Dashboard'
import { Login } from '@/components/Login'

function App() {
  const [isAuthenticated] = useState<boolean>(() => {
    const urlParams = new URLSearchParams(window.location.search)
    const token = urlParams.get('token')
    if (token) {
      localStorage.setItem('token', token)
      return true
    }
    return !!localStorage.getItem('token')
  })

  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    if (urlParams.has('token')) {
      window.history.replaceState({}, document.title, window.location.pathname)
    }
  }, [])

  if (!isAuthenticated) {
    return <Login />
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <Toaster position="top-center" richColors />
    </BrowserRouter>
  )
}

export default App
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(frontend): add /dashboard route"
```

---

### Task 4: Update Sidebar Navigation

**Files:**
- Modify: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: Split the Settings and Dashboard buttons**

Edit `frontend/src/components/Sidebar.tsx`. 
First, add `BarChart2` (or similar) to the `lucide-react` imports:

```tsx
import { GitBranch, CheckCircle2, CircleDashed, Loader2, BookOpen, ChevronRight, ChevronDown, Plus, LogOut, FolderArchive, Square, CheckSquare, RefreshCw, Trash2, Settings as SettingsIcon, Home as HomeIcon, BarChart2 } from 'lucide-react'
```

Then update the bottom section where it currently renders "返回工作台", "设置与大盘" and "退出登录":

```tsx
      <div className="p-4 border-t border-zinc-200 mt-auto shrink-0 flex flex-col gap-2">
        <Button 
          variant="ghost" 
          className="w-full flex items-center justify-start gap-2 text-zinc-700 hover:bg-zinc-100"
          onClick={() => navigate('/')}
        >
          <HomeIcon className="w-4 h-4" />
          返回工作台
        </Button>
        <Button 
          variant="ghost" 
          className="w-full flex items-center justify-start gap-2 text-zinc-700 hover:bg-zinc-100"
          onClick={() => navigate('/dashboard')}
        >
          <BarChart2 className="w-4 h-4" />
          数据大盘
        </Button>
        <Button 
          variant="ghost" 
          className="w-full flex items-center justify-start gap-2 text-zinc-700 hover:bg-zinc-100"
          onClick={() => navigate('/settings')}
        >
          <SettingsIcon className="w-4 h-4" />
          设置中心
        </Button>
        <div className="h-px bg-zinc-100 my-1 w-full" />
        <Button
          variant="ghost"
          className="w-full flex items-center justify-start gap-2 text-zinc-600 hover:text-red-600 hover:bg-red-50"
          onClick={() => {
            localStorage.removeItem('token')
            window.location.href = '/'
          }}
        >
          <LogOut className="w-4 h-4" />
          退出登录
        </Button>
      </div>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/Sidebar.tsx
git commit -m "feat(frontend): separate dashboard and settings links in sidebar"
```