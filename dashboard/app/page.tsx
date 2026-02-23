'use client'

import { useEffect, useState, useCallback } from 'react'
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  Legend, LineChart, Line, ResponsiveContainer
} from 'recharts'

const ADMIN_URL = 'http://localhost:8090'

type RuleMetric = { rule_id: string; allowed: number; blocked: number }
type Metrics = {
  total_allowed: number
  total_blocked: number
  by_rule: RuleMetric[]
  timestamp: string
}
type Rule = {
  rule_id: string
  algorithm: string
  limit: number
  window_secs: number
  enabled: boolean
}
type HistoryPoint = { time: string; allowed: number; blocked: number }

// ── tiny helpers ────────────────────────────────────────────────────────────
const StatCard = ({
  label, value, color = 'text-white',
}: { label: string; value: string | number; color?: string }) => (
  <div className="rounded-xl border border-white/[0.06] bg-[#0f1117] p-5">
    <p className="text-xs font-medium tracking-widest text-zinc-500 uppercase">{label}</p>
    <p className={`mt-2 text-4xl font-black tabular-nums ${color}`}>{value}</p>
  </div>
)

const Badge = ({ enabled }: { enabled: boolean }) => (
  <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold
    ${enabled ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
    <span className={`h-1.5 w-1.5 rounded-full ${enabled ? 'bg-emerald-400' : 'bg-red-400'}`} />
    {enabled ? 'active' : 'disabled'}
  </span>
)

const ALG_COLORS: Record<string, string> = {
  fixed_window:   'bg-blue-500/10 text-blue-400',
  sliding_window: 'bg-violet-500/10 text-violet-400',
  token_bucket:   'bg-amber-500/10 text-amber-400',
}
const AlgBadge = ({ alg }: { alg: string }) => (
  <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${ALG_COLORS[alg] ?? 'bg-zinc-800 text-zinc-400'}`}>
    {alg.replace(/_/g, ' ')}
  </span>
)

// ── modal ────────────────────────────────────────────────────────────────────
type ModalProps = {
  rule?: Rule | null
  onClose: () => void
  onSave: () => void
}

const ALGORITHMS = ['fixed_window', 'sliding_window', 'token_bucket']

function RuleModal({ rule, onClose, onSave }: ModalProps) {
  const isEdit = !!rule
  const [form, setForm] = useState({
    rule_id:     rule?.rule_id     ?? '',
    algorithm:   rule?.algorithm   ?? 'fixed_window',
    limit:       rule?.limit       ?? 10,
    window_secs: rule?.window_secs ?? 60,
  })
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState('')

  const set = (k: string, v: string | number) => setForm(f => ({ ...f, [k]: v }))

  const handleSubmit = async () => {
    setLoading(true); setError('')
    try {
      const url    = isEdit ? `${ADMIN_URL}/rules/${rule!.rule_id}` : `${ADMIN_URL}/rules`
      const method = isEdit ? 'PATCH' : 'POST'
      const body   = isEdit
        ? JSON.stringify({ limit: form.limit, window_secs: form.window_secs })
        : JSON.stringify(form)

      const res = await fetch(url, {
        method, headers: { 'Content-Type': 'application/json' }, body,
      })
      if (!res.ok) throw new Error(await res.text())
      onSave()
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Request failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-2xl border border-white/[0.08] bg-[#0f1117] p-6 shadow-2xl">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-lg font-bold">{isEdit ? 'Edit Rule' : 'New Rule'}</h2>
          <button onClick={onClose} className="text-zinc-500 hover:text-white transition-colors text-xl leading-none">✕</button>
        </div>

        <div className="space-y-4">
          {/* Rule ID — only on create */}
          {!isEdit && (
            <div>
              <label className="mb-1.5 block text-xs font-medium text-zinc-400">Rule ID</label>
              <input
                value={form.rule_id}
                onChange={e => set('rule_id', e.target.value)}
                placeholder="e.g. api, login, search"
                className="w-full rounded-lg border border-white/[0.08] bg-[#1a1d27] px-3 py-2.5 text-sm text-white placeholder-zinc-600 outline-none focus:border-blue-500 transition-colors"
              />
            </div>
          )}

          {/* Algorithm — only on create */}
          {!isEdit && (
            <div>
              <label className="mb-1.5 block text-xs font-medium text-zinc-400">Algorithm</label>
              <select
                value={form.algorithm}
                onChange={e => set('algorithm', e.target.value)}
                className="w-full rounded-lg border border-white/[0.08] bg-[#1a1d27] px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500 transition-colors"
              >
                {ALGORITHMS.map(a => (
                  <option key={a} value={a}>{a.replace(/_/g, ' ')}</option>
                ))}
              </select>
            </div>
          )}

          {/* Limit */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-zinc-400">
              Request Limit
            </label>
            <input
              type="number" min={1}
              value={form.limit}
              onChange={e => set('limit', parseInt(e.target.value))}
              className="w-full rounded-lg border border-white/[0.08] bg-[#1a1d27] px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500 transition-colors"
            />
          </div>

          {/* Window */}
          <div>
            <label className="mb-1.5 block text-xs font-medium text-zinc-400">
              Window (seconds)
            </label>
            <input
              type="number" min={1}
              value={form.window_secs}
              onChange={e => set('window_secs', parseInt(e.target.value))}
              className="w-full rounded-lg border border-white/[0.08] bg-[#1a1d27] px-3 py-2.5 text-sm text-white outline-none focus:border-blue-500 transition-colors"
            />
          </div>

          {error && <p className="rounded-lg bg-red-500/10 px-3 py-2 text-xs text-red-400">{error}</p>}
        </div>

        <div className="mt-6 flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 rounded-lg border border-white/[0.08] py-2.5 text-sm font-medium text-zinc-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            className="flex-1 rounded-lg bg-blue-600 py-2.5 text-sm font-semibold text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
          >
            {loading ? 'Saving…' : isEdit ? 'Update Rule' : 'Create Rule'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── custom tooltip ───────────────────────────────────────────────────────────
const ChartTooltip = ({ active, payload, label }: any) => {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border border-white/[0.08] bg-[#0f1117] px-3 py-2 text-xs shadow-xl">
      <p className="mb-1 font-semibold text-zinc-300">{label}</p>
      {payload.map((p: any) => (
        <p key={p.name} style={{ color: p.color }}>{p.name}: {p.value}</p>
      ))}
    </div>
  )
}

// ── main page ────────────────────────────────────────────────────────────────
export default function Dashboard() {
  const [metrics,     setMetrics]     = useState<Metrics | null>(null)
  const [rules,       setRules]       = useState<Rule[]>([])
  const [history,     setHistory]     = useState<HistoryPoint[]>([])
  const [lastUpdated, setLastUpdated] = useState('')
  const [modal,       setModal]       = useState<'create' | Rule | null>(null)
  const [deleting,    setDeleting]    = useState<string | null>(null)
  const [connected,   setConnected]   = useState(true)

  const fetchRules = useCallback(async () => {
    const res = await fetch(`${ADMIN_URL}/rules`)
    setRules(await res.json())
  }, [])

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await fetch(`${ADMIN_URL}/metrics`)
      const data: Metrics = await res.json()
      setMetrics(data)
      setConnected(true)
      setLastUpdated(new Date().toLocaleTimeString())
      setHistory(prev => [...prev, {
        time:    new Date().toLocaleTimeString(),
        allowed: data.total_allowed,
        blocked: data.total_blocked,
      }].slice(-20))
    } catch {
      setConnected(false)
    }
  }, [])

  useEffect(() => {
    fetchRules()
    fetchMetrics()
    const t = setInterval(() => { fetchRules(); fetchMetrics() }, 2000)
    return () => clearInterval(t)
  }, [fetchRules, fetchMetrics])

  const handleDisable = async (ruleId: string) => {
    setDeleting(ruleId)
    await fetch(`${ADMIN_URL}/rules/${ruleId}`, { method: 'DELETE' })
    await fetchRules()
    setDeleting(null)
  }

  const total     = (metrics?.total_allowed ?? 0) + (metrics?.total_blocked ?? 0)
  const allowRate = total > 0 ? ((metrics?.total_allowed ?? 0) / total * 100).toFixed(1) : '—'

  return (
    <div className="min-h-screen bg-[#080a0f] font-mono text-white">

      {/* top bar */}
      <header className="border-b border-white/[0.05] px-8 py-4">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-600">
              <svg className="h-4 w-4 text-white" fill="none" stroke="currentColor" strokeWidth={2.5} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
              </svg>
            </div>
            <div>
              <h1 className="text-sm font-bold tracking-tight">RLaaS</h1>
              <p className="text-[10px] text-zinc-500 tracking-widest uppercase">Rate Limiter as a Service</p>
            </div>
          </div>

          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2 text-xs">
              <span className={`h-2 w-2 rounded-full ${connected ? 'bg-emerald-400 shadow-[0_0_6px_#34d399]' : 'bg-red-500'}`} />
              <span className="text-zinc-500">{connected ? `live · ${lastUpdated}` : 'disconnected'}</span>
            </div>
            <button
              onClick={() => setModal('create')}
              className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-xs font-semibold hover:bg-blue-500 transition-colors"
            >
              <span>+</span> New Rule
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-8 py-8 space-y-8">

        {/* stat cards */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Total Requests (60s)" value={total} />
          <StatCard label="Allowed"    value={metrics?.total_allowed ?? 0} color="text-emerald-400" />
          <StatCard label="Blocked"    value={metrics?.total_blocked ?? 0} color="text-red-400" />
          <StatCard label="Allow Rate" value={allowRate === '—' ? '—' : `${allowRate}%`} color="text-blue-400" />
        </div>

        {/* charts */}
        <div className="grid grid-cols-2 gap-6">

          <div className="rounded-xl border border-white/[0.06] bg-[#0f1117] p-5">
            <p className="mb-4 text-xs font-semibold tracking-widest text-zinc-500 uppercase">By Rule — last 60s</p>
            {metrics?.by_rule?.length ? (
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={metrics.by_rule} barCategoryGap="30%">
                  <CartesianGrid strokeDasharray="3 3" stroke="#ffffff08" />
                  <XAxis dataKey="rule_id" stroke="#52525b" tick={{ fontSize: 11 }} />
                  <YAxis stroke="#52525b" tick={{ fontSize: 11 }} />
                  <Tooltip content={<ChartTooltip />} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Bar dataKey="allowed" fill="#34d399" name="Allowed" radius={[4,4,0,0]} />
                  <Bar dataKey="blocked" fill="#f87171" name="Blocked" radius={[4,4,0,0]} />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex h-[220px] items-center justify-center text-xs text-zinc-600">
                no data — run some requests against your gRPC service
              </div>
            )}
          </div>

          <div className="rounded-xl border border-white/[0.06] bg-[#0f1117] p-5">
            <p className="mb-4 text-xs font-semibold tracking-widest text-zinc-500 uppercase">Request Timeline</p>
            {history.length > 1 ? (
              <ResponsiveContainer width="100%" height={220}>
                <LineChart data={history}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#ffffff08" />
                  <XAxis dataKey="time" stroke="#52525b" tick={{ fontSize: 10 }} />
                  <YAxis stroke="#52525b" tick={{ fontSize: 11 }} />
                  <Tooltip content={<ChartTooltip />} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Line type="monotone" dataKey="allowed" stroke="#34d399" dot={false} strokeWidth={2} name="Allowed" />
                  <Line type="monotone" dataKey="blocked" stroke="#f87171" dot={false} strokeWidth={2} name="Blocked" />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex h-[220px] items-center justify-center text-xs text-zinc-600">
                collecting data…
              </div>
            )}
          </div>
        </div>

        {/* rules table */}
        <div className="rounded-xl border border-white/[0.06] bg-[#0f1117] overflow-hidden">
          <div className="border-b border-white/[0.06] px-5 py-4">
            <p className="text-xs font-semibold tracking-widest text-zinc-500 uppercase">Rules</p>
          </div>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-white/[0.04] text-[11px] tracking-widest text-zinc-600 uppercase">
                {['Rule ID', 'Algorithm', 'Limit', 'Window', 'Status', 'Actions'].map(h => (
                  <th key={h} className="px-5 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rules.map((rule, i) => (
                <tr
                  key={rule.rule_id}
                  className={`border-b border-white/[0.03] transition-colors hover:bg-white/[0.02]
                    ${i === rules.length - 1 ? 'border-b-0' : ''}`}
                >
                  <td className="px-5 py-3.5 font-mono text-xs text-blue-300">{rule.rule_id}</td>
                  <td className="px-5 py-3.5"><AlgBadge alg={rule.algorithm} /></td>
                  <td className="px-5 py-3.5 tabular-nums text-zinc-300">{rule.limit}</td>
                  <td className="px-5 py-3.5 tabular-nums text-zinc-300">{rule.window_secs}s</td>
                  <td className="px-5 py-3.5"><Badge enabled={rule.enabled} /></td>
                  <td className="px-5 py-3.5">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setModal(rule)}
                        className="rounded-md border border-white/[0.08] px-3 py-1 text-xs text-zinc-400 hover:border-blue-500 hover:text-blue-400 transition-colors"
                      >
                        Edit
                      </button>
                      {rule.enabled && (
                        <button
                          onClick={() => handleDisable(rule.rule_id)}
                          disabled={deleting === rule.rule_id}
                          className="rounded-md border border-white/[0.08] px-3 py-1 text-xs text-zinc-400 hover:border-red-500 hover:text-red-400 transition-colors disabled:opacity-40"
                        >
                          {deleting === rule.rule_id ? '…' : 'Disable'}
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </main>

      {/* modal */}
      {modal && (
        <RuleModal
          rule={modal === 'create' ? null : modal}
          onClose={() => setModal(null)}
          onSave={fetchRules}
        />
      )}
    </div>
  )
}