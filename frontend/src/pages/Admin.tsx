import { useCallback, useEffect, useState } from 'react'
import { Button } from 'primereact/button'
import { InputText } from 'primereact/inputtext'
import { InputTextarea } from 'primereact/inputtextarea'

type SettingRow = {
    key: string
    value: string
}

async function readErrorMessage(res: Response): Promise<string> {
    const data = (await res.json().catch(() => ({}))) as { error?: string }
    return typeof data.error === 'string' && data.error !== '' ? data.error : `Request failed (${res.status})`
}

export function AdminPage() {
    const [rows, setRows] = useState<SettingRow[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [newKey, setNewKey] = useState('')
    const [newValue, setNewValue] = useState('')
    const [busyKey, setBusyKey] = useState<string | null>(null)

    const load = useCallback(async () => {
        setError(null)
        const res = await fetch('/api/settings')
        if (!res.ok) {
            setError(await readErrorMessage(res))
            setRows([])
            return
        }
        const data = (await res.json()) as unknown
        if (!Array.isArray(data)) {
            setError('Invalid settings response')
            setRows([])
            return
        }
        setRows(
            data.map((item) => {
                const r = item as { key?: unknown; value?: unknown }
                return {
                    key: typeof r.key === 'string' ? r.key : '',
                    value: typeof r.value === 'string' ? r.value : '',
                }
            }),
        )
    }, [])

    useEffect(() => {
        setLoading(true)
        load().finally(() => setLoading(false))
    }, [load])

    async function putSetting(key: string, value: string): Promise<boolean> {
        setBusyKey(key)
        setError(null)
        try {
            const res = await fetch('/api/admin/settings', {
                method: 'PUT',
                credentials: 'include',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ key, value }),
            })
            if (!res.ok) {
                setError(await readErrorMessage(res))
                return false
            }
            await load()
            return true
        } catch {
            setError('Network error while saving')
            return false
        } finally {
            setBusyKey(null)
        }
    }

    async function removeSetting(key: string) {
        if (!window.confirm(`Remove setting “${key}”?`)) {
            return
        }
        setBusyKey(key)
        setError(null)
        try {
            const res = await fetch(`/api/admin/settings/${encodeURIComponent(key)}`, {
                method: 'DELETE',
                credentials: 'include',
            })
            if (!res.ok && res.status !== 204) {
                setError(await readErrorMessage(res))
                return
            }
            await load()
        } catch {
            setError('Network error while deleting')
        } finally {
            setBusyKey(null)
        }
    }

    async function addNew() {
        const k = newKey.trim()
        if (!k) {
            setError('Key is required')
            return
        }
        const ok = await putSetting(k, newValue)
        if (ok) {
            setNewKey('')
            setNewValue('')
        }
    }

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-slate-50 text-slate-600">
                Loading…
            </div>
        )
    }

    return (
        <div className="min-h-screen flex flex-col items-stretch bg-slate-50 p-6 gap-6 max-w-3xl mx-auto w-full box-border">
            <div>
                <h1 className="text-2xl font-semibold text-slate-800 m-0">Admin</h1>
                <p className="text-slate-600 m-0 mt-2">
                    Key–value settings are stored in the database. The map and other features can read them from{' '}
                    <code className="text-sm bg-slate-200/80 px-1 rounded">/api/settings</code>.
                </p>
            </div>

            {error && (
                <p
                    className="text-sm text-amber-900 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3 m-0"
                    role="alert"
                >
                    {error}
                </p>
            )}

            <section className="flex flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm text-left">
                <h2 className="text-lg font-medium text-slate-800 m-0">Existing settings</h2>
                {rows.length === 0 ? (
                    <p className="text-slate-600 m-0">No settings yet. Add one below.</p>
                ) : (
                    <ul className="flex flex-col gap-4 list-none m-0 p-0">
                        {rows.map((row) => (
                            <li
                                key={row.key}
                                className="flex flex-col gap-2 border-b border-slate-100 pb-4 last:border-b-0 last:pb-0"
                            >
                                <label className="text-xs font-medium uppercase tracking-wide text-slate-500">Key</label>
                                <span className="text-slate-800 font-mono text-sm break-all">{row.key}</span>
                                <label htmlFor={`val-${row.key}`} className="text-xs font-medium text-slate-500">
                                    Value
                                </label>
                                <InputTextarea
                                    id={`val-${row.key}`}
                                    value={row.value}
                                    onChange={(e) =>
                                        setRows((prev) =>
                                            prev.map((r) =>
                                                r.key === row.key ? { ...r, value: e.target.value } : r,
                                            ),
                                        )
                                    }
                                    rows={3}
                                    autoResize
                                    className="w-full font-mono text-sm"
                                />
                                <div className="flex flex-wrap gap-2">
                                    <Button
                                        type="button"
                                        label="Save"
                                        size="small"
                                        disabled={busyKey !== null}
                                        loading={busyKey === row.key}
                                        onClick={() => void putSetting(row.key, row.value)}
                                    />
                                    <Button
                                        type="button"
                                        label="Delete"
                                        size="small"
                                        severity="danger"
                                        outlined
                                        disabled={busyKey !== null}
                                        loading={busyKey === row.key}
                                        onClick={() => void removeSetting(row.key)}
                                    />
                                </div>
                            </li>
                        ))}
                    </ul>
                )}
            </section>

            <section className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm text-left">
                <h2 className="text-lg font-medium text-slate-800 m-0">Add setting</h2>
                <div className="flex flex-col gap-1">
                    <label htmlFor="new-key" className="text-sm text-slate-700">
                        Key
                    </label>
                    <InputText
                        id="new-key"
                        value={newKey}
                        onChange={(e) => setNewKey(e.target.value)}
                        className="w-full font-mono text-sm"
                        placeholder="e.g. finish_latitude"
                    />
                </div>
                <div className="flex flex-col gap-1">
                    <label htmlFor="new-value" className="text-sm text-slate-700">
                        Value
                    </label>
                    <InputTextarea
                        id="new-value"
                        value={newValue}
                        onChange={(e) => setNewValue(e.target.value)}
                        rows={3}
                        autoResize
                        className="w-full font-mono text-sm"
                    />
                </div>
                <Button
                    type="button"
                    label="Add or update"
                    disabled={busyKey !== null}
                    loading={busyKey === newKey.trim() && newKey.trim() !== ''}
                    onClick={() => void addNew()}
                />
            </section>
        </div>
    )
}
