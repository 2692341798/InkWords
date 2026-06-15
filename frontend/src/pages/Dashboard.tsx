import { useEffect, useState, useMemo } from 'react'
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { Coins, FileText, Hash, User, Loader2, Upload, BookOpen } from 'lucide-react'
import { userService } from '@/services/user'
import { toast } from 'sonner'

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
      const { stats, profile } = await userService.getDashboardData()
      setStats(stats)
      setProfile(profile)
      setNewUsername(profile.username)
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
      await userService.updateUsername(newUsername)
      setProfile(prev => prev ? { ...prev, username: newUsername } : null)
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
      toast.error('图片大小不能超过 2MB')
      return
    }

    setUploadingAvatar(true)
    const formData = new FormData()
    formData.append('avatar', file)

    try {
      const avatarUrl = await userService.uploadAvatar(formData)
      setProfile(prev => prev ? { ...prev, avatar_url: avatarUrl } : null)
    } catch (e) {
      console.error('Failed to upload avatar:', e)
      toast.error(e instanceof Error ? e.message : '上传失败')
    } finally {
      setUploadingAvatar(false)
    }
  }

  const COLORS = ['#6366f1', '#8b5cf6', '#ec4899', '#d946ef', '#f43f5e', '#f43f5e', '#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16', '#22c55e', '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9', '#3b82f6', '#0284c7']

  const processedChartData = useMemo(() => {
    if (!stats?.tech_stack_stats || stats.tech_stack_stats.length === 0) {
      return [];
    }

    // Sort descending by count
    const sortedStats = [...stats.tech_stack_stats].sort((a, b) => b.count - a.count);
    
    // If more than 8 items, group the rest into "Other"
    if (sortedStats.length > 8) {
      const top8 = sortedStats.slice(0, 8);
      const others = sortedStats.slice(8);
      const otherCount = others.reduce((sum, item) => sum + item.count, 0);
      
      top8.push({ name: '其他', count: otherCount });
      return top8;
    }
    
    return sortedStats;
  }, [stats]);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-indigo-500 animate-spin" />
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto bg-background custom-scrollbar">
      <div className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-12">
        {/* Profile Section */}
        <div className="flex items-center gap-6 bg-card p-6 rounded-2xl border border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
          <div className="relative group shrink-0">
            <div className="w-20 h-20 rounded-full overflow-hidden border-2 border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)] bg-secondary flex items-center justify-center">
              {profile?.avatar_url ? (
                <img src={profile.avatar_url} alt="Avatar" className="w-full h-full object-cover" />
              ) : (
                <User className="w-8 h-8 text-muted-foreground" />
              )}
            </div>
            <label className="absolute inset-0 bg-black/40 text-white flex flex-col items-center justify-center rounded-full opacity-0 group-hover:opacity-100 cursor-pointer transition-opacity backdrop-blur-sm">
              {uploadingAvatar ? <Loader2 className="w-5 h-5 animate-spin" /> : <Upload className="w-5 h-5" />}
              <span className="text-[10px] mt-1 font-medium">更换</span>
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
                  className="text-2xl font-semibold text-foreground border-b border-border focus:border-foreground focus:outline-none bg-transparent transition-colors"
                />
              ) : (
                <h1
                  className="text-2xl font-semibold text-foreground cursor-pointer hover:text-muted-foreground transition-colors"
                  onClick={() => setEditingUsername(true)}
                  title="点击修改用户名"
                >
                  {profile?.username || '未命名用户'}
                </h1>
              )}
            </div>
            <p className="text-sm text-muted-foreground mt-1">{profile?.email}</p>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-card p-5 rounded-xl border border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)] flex items-start gap-4">
            <div className="p-2.5 bg-secondary text-foreground rounded-lg">
              <Hash className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">消耗 Token</p>
              <h3 className="text-2xl font-semibold text-foreground mt-1">{stats?.tokens_used?.toLocaleString() || 0}</h3>
            </div>
          </div>

          <div className="bg-card p-5 rounded-xl border border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)] flex items-start gap-4">
            <div className="p-2.5 bg-secondary text-foreground rounded-lg">
              <Coins className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">预估费用 (元)</p>
              <h3 className="text-2xl font-semibold text-foreground mt-1">¥{stats?.estimated_cost?.toFixed(2) || '0.00'}</h3>
              <p className="text-[10px] text-muted-foreground mt-1">按 2.3元/1M Tokens 估算</p>
            </div>
          </div>

          <div className="bg-card p-5 rounded-xl border border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)] flex items-start gap-4">
            <div className="p-2.5 bg-secondary text-foreground rounded-lg">
              <FileText className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">生成文章数</p>
              <h3 className="text-2xl font-semibold text-foreground mt-1">{stats?.total_articles?.toLocaleString() || 0}</h3>
            </div>
          </div>

          <div className="bg-card p-5 rounded-xl border border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)] flex items-start gap-4">
            <div className="p-2.5 bg-secondary text-foreground rounded-lg">
              <BookOpen className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">生成总字数</p>
              <h3 className="text-2xl font-semibold text-foreground mt-1">{stats?.total_words?.toLocaleString() || 0}</h3>
            </div>
          </div>
        </div>

        {/* Charts Section */}
        <div className="bg-card p-6 rounded-2xl border border-border shadow-[0_2px_8px_rgba(0,0,0,0.02)]">
          <h2 className="text-lg font-medium text-foreground mb-6">技术栈涉及频率分布</h2>
          
          <div className="h-[400px] w-full">
            {processedChartData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={processedChartData}
                    cx="50%"
                    cy="50%"
                    innerRadius={80}
                    outerRadius={130}
                    paddingAngle={2}
                    dataKey="count"
                    nameKey="name"
                    label={({ name, percent }) => `${name} ${((percent || 0) * 100).toFixed(0)}%`}
                  >
                    {processedChartData.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    formatter={(value: unknown, name: unknown) => [String(value), String(name)]}
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e4e4e7', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
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
