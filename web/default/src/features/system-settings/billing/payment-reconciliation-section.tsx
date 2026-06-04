/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useMemo, useState } from 'react'
import { Download, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  downloadTopupReconciliationCsv,
  getTopupReconciliation,
} from '@/features/wallet/api'
import type {
  TopupReconciliationQuery,
  TopupReconciliationRow,
} from '@/features/wallet/types'
import { SettingsSection } from '../components/settings-section'

const daySeconds = 24 * 60 * 60

function toDateTimeLocal(timestamp: number) {
  const date = new Date(timestamp * 1000)
  const offset = date.getTimezoneOffset() * 60 * 1000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function fromDateTimeLocal(value: string) {
  const timestamp = Math.floor(new Date(value).getTime() / 1000)
  return Number.isFinite(timestamp) ? timestamp : 0
}

function formatTime(timestamp: number) {
  if (!timestamp) return '-'
  return new Date(timestamp * 1000).toLocaleString()
}

function formatMoney(value: number) {
  return value.toFixed(2)
}

function buildDefaultQuery(): TopupReconciliationQuery {
  const now = Math.floor(Date.now() / 1000)
  return {
    start_time: now - daySeconds,
    end_time: now,
  }
}

export function PaymentReconciliationSection() {
  const { t } = useTranslation()
  const [query, setQuery] = useState<TopupReconciliationQuery>(
    buildDefaultQuery
  )
  const [rows, setRows] = useState<TopupReconciliationRow[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isExporting, setIsExporting] = useState(false)

  const totals = useMemo(
    () =>
      rows.reduce(
        (acc, row) => ({
          orderCount: acc.orderCount + row.order_count,
          quotaAmount: acc.quotaAmount + row.total_amount,
          moneyAmount: acc.moneyAmount + row.total_money,
        }),
        { orderCount: 0, quotaAmount: 0, moneyAmount: 0 }
      ),
    [rows]
  )

  const load = async () => {
    setIsLoading(true)
    try {
      const response = await getTopupReconciliation(query)
      setRows(response.data?.items ?? [])
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const exportCsv = async () => {
    setIsExporting(true)
    try {
      const { blob, filename } = await downloadTopupReconciliationCsv(query)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <SettingsSection title={t('Payment Reconciliation')}>
      <div className='space-y-5'>
        <div className='grid gap-3 md:grid-cols-5'>
          <Input
            type='datetime-local'
            value={toDateTimeLocal(query.start_time)}
            onChange={(event) =>
              setQuery((current) => ({
                ...current,
                start_time: fromDateTimeLocal(event.target.value),
              }))
            }
          />
          <Input
            type='datetime-local'
            value={toDateTimeLocal(query.end_time)}
            onChange={(event) =>
              setQuery((current) => ({
                ...current,
                end_time: fromDateTimeLocal(event.target.value),
              }))
            }
          />
          <Input
            placeholder={t('Provider')}
            value={query.payment_provider ?? ''}
            onChange={(event) =>
              setQuery((current) => ({
                ...current,
                payment_provider: event.target.value.trim() || undefined,
              }))
            }
          />
          <Input
            placeholder={t('Payment Method')}
            value={query.payment_method ?? ''}
            onChange={(event) =>
              setQuery((current) => ({
                ...current,
                payment_method: event.target.value.trim() || undefined,
              }))
            }
          />
          <Input
            placeholder={t('Status')}
            value={query.status ?? ''}
            onChange={(event) =>
              setQuery((current) => ({
                ...current,
                status: event.target.value.trim() || undefined,
              }))
            }
          />
        </div>

        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='grid gap-3 text-sm md:grid-cols-3'>
            <div>
              <span className='text-muted-foreground'>{t('Orders')}</span>
              <div className='font-semibold'>{totals.orderCount}</div>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Quota')}</span>
              <div className='font-semibold'>{totals.quotaAmount}</div>
            </div>
            <div>
              <span className='text-muted-foreground'>{t('Payment Amount')}</span>
              <div className='font-semibold'>{formatMoney(totals.moneyAmount)}</div>
            </div>
          </div>
          <div className='flex gap-2'>
            <Button type='button' variant='outline' onClick={load} disabled={isLoading}>
              <RefreshCw className='h-4 w-4' />
              {t('Refresh')}
            </Button>
            <Button
              type='button'
              variant='outline'
              onClick={exportCsv}
              disabled={isExporting}
            >
              <Download className='h-4 w-4' />
              {t('Export CSV')}
            </Button>
          </div>
        </div>

        <div className='overflow-x-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Provider')}</TableHead>
                <TableHead>{t('Payment Method')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead className='text-right'>{t('Orders')}</TableHead>
                <TableHead className='text-right'>{t('Quota')}</TableHead>
                <TableHead className='text-right'>{t('Payment Amount')}</TableHead>
                <TableHead>{t('First Created')}</TableHead>
                <TableHead>{t('Last Created')}</TableHead>
                <TableHead>{t('Last Completed')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className='h-24 text-center'>
                    {isLoading ? t('Loading...') : t('No data')}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((row) => (
                  <TableRow
                    key={`${row.payment_provider}-${row.payment_method}-${row.status}`}
                  >
                    <TableCell>{row.payment_provider || '-'}</TableCell>
                    <TableCell>{row.payment_method || '-'}</TableCell>
                    <TableCell>{row.status || '-'}</TableCell>
                    <TableCell className='text-right'>{row.order_count}</TableCell>
                    <TableCell className='text-right'>{row.total_amount}</TableCell>
                    <TableCell className='text-right'>
                      {formatMoney(row.total_money)}
                    </TableCell>
                    <TableCell>{formatTime(row.first_create_time)}</TableCell>
                    <TableCell>{formatTime(row.last_create_time)}</TableCell>
                    <TableCell>{formatTime(row.last_complete_time)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </SettingsSection>
  )
}
