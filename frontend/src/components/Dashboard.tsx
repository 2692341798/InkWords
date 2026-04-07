import { useEffect, useState } from 'react'
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  Legend
} from 'recharts'
import { Coins, FileText, Hash, User, Loader2, Upload, BookOpen } from 'lucide-react'

interface TechStackStat {
  name: string
  count: number
}

interface UserStats {
  tokens_used: number
  estimated_cost: number
  total_articles: number
  total_words: number
  tech_stack_stats: TechStackStat[]
}

interface UserProfile {
  username: string
  email: string
  avatar_url: string
  subscription_tier: number
  token_limit: number
}

export function Dashboard() {
  const [stats, setStats] = useState<UserStats | null>(null)
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [editingUsername, setEditingUsername] = useState(false)
  const [newUsername, setNewUsername] = useState('')
  const [uploadingAvatar, setUploadingAvatar] = useState(false)

  const fetchData = async () => {
    try {
      const token = localStorage.getItem('token')
      const [statsRes, profileRes] = await Promise.all([
        fetch('/api/v1/user/stats', { headers: { Authorization: `Bearer ${token}` } }),
        fetch('/api/v1/user/profile', { headers: { Authorization: `Bearer ${token}` } })
      ])

      if (statsRes.ok) {
        const json = await statsRes.json()
        setStats(json.data)
      }
      if (profileRes.ok) {
        const json = await profileRes.json()
        setProfile(json.data)
        setNewUsername(json.data.username)
      }
    } catch (e) {
      console.error('Failed to fetch dashboard data:', e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleUpdateUsername = async () => {
    if (!newUsername.trim() || newUsername === profile?.username) {
      setEditingUsername(false)
      return
    }
    
    try {
      const token = localStorage.getItem('token')
      const res = await fetch('/api/v1/user/profile', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({ username: newUsername })
      })
      if (res.ok) {
        setProfile(prev => prev ? { ...prev, username: newUsername } : null)
      }
    } catch (e) {
      console.error('Failed to update username:', e)
    } finally {
      setEditingUsername(false)
    }
  }

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (file.size > 2 * 1024 * 1024) {
      alert('图片大小不能超过 2MB')
      return
    }

    setUploadingAvatar(true)
    const formData = new FormData()
    formData.append('avatar', file)

    try {
      const token = localStorage.getItem('token')
      const res = await fetch('/api/v1/user/avatar', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`
        },
        body: formData
      })
      if (res.ok) {
        const json = await res.json()
        setProfile(prev => prev ? { ...prev, avatar_url: json.data.avatar_url } : null)
      } else {
        const json = await res.json()
        alert(json.message || '上传失败')
      }
    } catch (e) {
      console.error('Failed to upload avatar:', e)
    } finally {
      setUploadingAvatar(false)
    }
  }

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-indigo-500 animate-spin" />
      </div>
    )
  }

  const COLORS = ['#6366f1', '#8b5cf6', '#ec4899', '#d946ef', '#f43f5e', '#f43f5e', '#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16', '#22c55e', '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9', '#3b82f6', '#0284c7']

  return (
    <div className="flex-1 flex flex-col overflow-y-auto bg-zinc-50 p-8 custom-scrollbar">
      <div className="max-w-5xl mx-auto w-full space-y-8">
        
        {/* Header / Profile Section */}
        <div className="bg-white p-6 rounded-xl border border-zinc-200 shadow-sm flex items-center gap-6">
          <div className="relative group">
            <div className="w-20 h-20 rounded-full bg-zinc-100 border border-zinc-200 overflow-hidden flex items-center justify-center shrink-0">
              {profile?.avatar_url ? (
                <img src={profile.avatar_url} alt="Avatar" className="w-full h-full object-cover" />
              ) : (
                <User className="w-10 h-10 text-zinc-400" />
              )}
            </div>
            <label className="absolute inset-0 bg-black/50 text-white flex flex-col items-center justify-center rounded-full opacity-0 group-hover:opacity-100 cursor-pointer transition-opacity">
              {uploadingAvatar ? <Loader2 className="w-5 h-5 animate-spin" /> : <Upload className="w-5 h-5" />}
              <span className="text-[10px] mt-1">上传</span>
              <input type="file" accept="image/*" className="hidden" onChange={handleAvatarUpload} disabled={uploadingAvatar} />
            </label>
          </div>
          
          <div className="flex-1">
            <div className="flex items-center gap-3">
              {editingUsername ? (
                <input
                  autoFocus
                  type="text"
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  onBlur={handleUpdateUsername}
                  onKeyDown={(e) => e.key === 'Enter' && handleUpdateUsername()}
                  className="text-2xl font-bold text-zinc-900 border-b-2 border-indigo-500 focus:outline-none bg-transparent"
                />
              ) : (
                <h1 
                  className="text-2xl font-bold text-zinc-900 cursor-pointer hover:text-indigo-600 transition-colors"
                  onClick={() => setEditingUsername(true)}
                  title="点击修改用户名"
                >
                  {profile?.username || '未命名用户'}
                </h1>
              )}
            </div>
            <p className="text-sm text-zinc-500 mt-1">{profile?.email}</p>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex items-start gap-4">
            <div className="p-3 bg-blue-50 text-blue-600 rounded-lg">
              <Hash className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm font-medium text-zinc-500">消耗 Token</p>
              <h3 className="text-2xl font-bold text-zinc-900 mt-1">{stats?.tokens_used?.toLocaleString() || 0}</h3>
            </div>
          </div>

          <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex items-start gap-4">
            <div className="p-3 bg-green-50 text-green-600 rounded-lg">
              <Coins className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm font-medium text-zinc-500">预估费用 (元)</p>
              <h3 className="text-2xl font-bold text-zinc-900 mt-1">¥{stats?.estimated_cost?.toFixed(2) || '0.00'}</h3>
              <p className="text-[10px] text-zinc-400 mt-1">按 2.3元/1M Tokens 均价估算</p>
            </div>
          </div>

          <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex items-start gap-4">
            <div className="p-3 bg-indigo-50 text-indigo-600 rounded-lg">
              <FileText className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm font-medium text-zinc-500">生成文章数</p>
              <h3 className="text-2xl font-bold text-zinc-900 mt-1">{stats?.total_articles?.toLocaleString() || 0}</h3>
            </div>
          </div>

          <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-sm flex items-start gap-4">
            <div className="p-3 bg-orange-50 text-orange-600 rounded-lg">
              <BookOpen className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm font-medium text-zinc-500">生成总字数</p>
              <h3 className="text-2xl font-bold text-zinc-900 mt-1">{stats?.total_words?.toLocaleString() || 0}</h3>
            </div>
          </div>
        </div>

        {/* Charts Section */}
        <div className="bg-white p-6 rounded-xl border border-zinc-200 shadow-sm">
          <h2 className="text-lg font-semibold text-zinc-800 mb-6">技术栈涉及频率分布</h2>
          
          <div className="h-[400px] w-full">
            {stats?.tech_stack_stats && stats.tech_stack_stats.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={stats.tech_stack_stats.sort((a, b) => b.count - a.count)}
                    dataKey="count"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    innerRadius={80}
                    outerRadius={140}
                    paddingAngle={2}
                    label={({ name, percent }) => `${name} ${((percent || 0) * 100).toFixed(0)}%`}
                    labelLine={true}
                  >
                    {stats.tech_stack_stats.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e4e4e7', boxShadow: '0 4px 6px -1px rgba(0,0,0,0.05)' }}
                    formatter={(value: any, name: any) => [`${value} 篇`, name]}
                  />
                  <Legend verticalAlign="bottom" height={36} />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="w-full h-full flex items-center justify-center text-zinc-400 text-sm">
                暂无技术栈数据
              </div>
            )}
          </div>
        </div>
        
      </div>
    </div>
  )
}