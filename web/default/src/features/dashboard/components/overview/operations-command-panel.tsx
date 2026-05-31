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
import { Link } from '@tanstack/react-router'
import {
  Activity,
  ArrowRight,
  BarChart3,
  CheckCircle2,
  CreditCard,
  KeyRound,
  RadioTower,
  ShieldCheck,
  TrendingUp,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { formatNumber, formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'
import {
  useApiInfo,
  useDashboardContentVisibility,
} from '../../hooks/use-status-data'

type OperationsLaneTone = 'ready' | 'watch' | 'action'
type OperationsActionPath =
  | '/wallet'
  | '/usage-logs'
  | '/channels'
  | '/keys'
  | '/system-settings/security'

interface OperationsLane {
  title: string
  metric: string
  description: string
  action: string
  to: OperationsActionPath
  icon: LucideIcon
  tone: OperationsLaneTone
}

const TONE_STYLES: Record<
  OperationsLaneTone,
  { dot: string; label: string; surface: string; icon: string }
> = {
  ready: {
    dot: 'bg-success',
    label: 'Ready',
    surface: 'bg-success/5 border-success/20',
    icon: 'text-success',
  },
  watch: {
    dot: 'bg-warning',
    label: 'Watch',
    surface: 'bg-warning/5 border-warning/20',
    icon: 'text-warning',
  },
  action: {
    dot: 'bg-destructive',
    label: 'Action needed',
    surface: 'bg-destructive/5 border-destructive/20',
    icon: 'text-destructive',
  },
}

function getCreditTone(remainQuota: number, usedQuota: number) {
  if (remainQuota <= 0) return 'action'
  if (usedQuota > 0 && remainQuota < usedQuota * 0.1) return 'watch'
  return 'ready'
}

function getTrafficTone(requestCount: number) {
  if (requestCount <= 0) return 'watch'
  return 'ready'
}

function getRouteTone(apiInfoCount: number, isAdmin: boolean) {
  if (!isAdmin) return 'ready'
  if (apiInfoCount <= 0) return 'watch'
  return 'ready'
}

function getControlsTone(args: {
  emailVerification?: boolean
  turnstileCheck?: boolean
  passkeyLogin?: boolean
}) {
  if (args.emailVerification || args.turnstileCheck || args.passkeyLogin) {
    return 'ready'
  }
  return 'watch'
}

function OperationsLaneCard(props: { lane: OperationsLane }) {
  const { t } = useTranslation()
  const Icon = props.lane.icon
  const tone = TONE_STYLES[props.lane.tone]

  return (
    <div
      className={cn(
        'flex min-h-36 flex-col justify-between gap-4 rounded-xl border p-3.5',
        tone.surface
      )}
    >
      <div className='flex items-start justify-between gap-3'>
        <span className='bg-background/80 flex size-9 shrink-0 items-center justify-center rounded-lg border shadow-xs'>
          <Icon className={cn('size-4', tone.icon)} aria-hidden='true' />
        </span>
        <span className='bg-background/70 inline-flex h-6 shrink-0 items-center gap-1.5 rounded-full border px-2 text-[11px] font-medium'>
          <span className={cn('size-1.5 rounded-full', tone.dot)} />
          {t(tone.label)}
        </span>
      </div>

      <div className='min-w-0 space-y-1'>
        <div className='text-muted-foreground text-xs font-medium'>
          {props.lane.title}
        </div>
        <div
          className='truncate font-mono text-xl font-semibold tracking-tight tabular-nums'
          title={props.lane.metric}
        >
          {props.lane.metric}
        </div>
        <p className='text-muted-foreground line-clamp-2 text-xs leading-relaxed'>
          {props.lane.description}
        </p>
      </div>

      <Button
        variant='outline'
        size='sm'
        className='bg-background/70 h-8 justify-between px-2.5 text-xs'
        render={<Link to={props.lane.to} />}
      >
        <span className='truncate'>{props.lane.action}</span>
        <ArrowRight data-icon='inline-end' />
      </Button>
    </div>
  )
}

export function OperationsCommandPanel() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const { status } = useStatus()
  const { items: apiInfoItems } = useApiInfo()
  const visibility = useDashboardContentVisibility()
  const isAdmin = Boolean(user?.role && user.role >= ROLE.ADMIN)

  const remainQuota = Number(user?.quota ?? 0)
  const usedQuota = Number(user?.used_quota ?? 0)
  const requestCount = Number(user?.request_count ?? 0)

  const lanes: OperationsLane[] = [
    {
      title: t('Revenue runway'),
      metric: formatQuota(remainQuota),
      description:
        remainQuota > 0
          ? t('Balance is available for live traffic and renewals.')
          : t('Balance is depleted. Top up before routing paid requests.'),
      action: t('Open wallet'),
      to: '/wallet',
      icon: CreditCard,
      tone: getCreditTone(remainQuota, usedQuota),
    },
    {
      title: t('Demand signal'),
      metric: formatNumber(requestCount),
      description:
        requestCount > 0
          ? t('Requests have been recorded. Review logs for conversion quality.')
          : t('No requests yet. Create a key and send a test request.'),
      action: requestCount > 0 ? t('Review logs') : t('Create API Key'),
      to: requestCount > 0 ? '/usage-logs' : '/keys',
      icon: requestCount > 0 ? TrendingUp : KeyRound,
      tone: getTrafficTone(requestCount),
    },
    {
      title: t('Route readiness'),
      metric: isAdmin
        ? t('{{count}} public endpoint(s)', { count: apiInfoItems.length })
        : t('User access'),
      description: isAdmin
        ? t('Keep channel and endpoint information ready for customer traffic.')
        : t('Use API keys and logs to validate your integration.'),
      action: isAdmin ? t('Manage channels') : t('Manage keys'),
      to: isAdmin ? '/channels' : '/keys',
      icon: RadioTower,
      tone: getRouteTone(apiInfoItems.length, isAdmin),
    },
    {
      title: t('Trust controls'),
      metric:
        visibility.uptimeKuma || status?.email_verification
          ? t('Monitored')
          : t('Needs review'),
      description: t(
        'Review verification, bot protection, and monitoring before scaling.'
      ),
      action: isAdmin ? t('Security settings') : t('Usage Logs'),
      to: isAdmin ? '/system-settings/security' : '/usage-logs',
      icon: ShieldCheck,
      tone: getControlsTone({
        emailVerification: Boolean(status?.email_verification),
        passkeyLogin: Boolean(status?.passkey_login),
        turnstileCheck: Boolean(status?.turnstile_check),
      }),
    },
  ]

  const actionCount = lanes.filter((lane) => lane.tone === 'action').length
  const watchCount = lanes.filter((lane) => lane.tone === 'watch').length

  return (
    <section className='bg-card overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
        <div className='flex min-w-0 items-center gap-3'>
          <span className='bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-xl border border-primary/20'>
            <Activity className='text-primary size-4' aria-hidden='true' />
          </span>
          <div className='min-w-0'>
            <h3 className='truncate text-sm font-semibold'>
              {t('Operations command center')}
            </h3>
            <p className='text-muted-foreground line-clamp-1 text-xs'>
              {t('Prioritize revenue, traffic, routing, and trust controls.')}
            </p>
          </div>
        </div>

        <div className='flex items-center gap-2'>
          <span className='bg-background inline-flex h-7 items-center gap-1.5 rounded-full border px-2.5 text-xs'>
            <CheckCircle2
              className='text-success size-3.5'
              aria-hidden='true'
            />
            <span className='font-mono tabular-nums'>
              {lanes.length - actionCount - watchCount}/{lanes.length}
            </span>
            <span className='text-muted-foreground'>{t('ready')}</span>
          </span>
          {watchCount > 0 && (
            <span className='bg-warning/10 text-warning inline-flex h-7 items-center gap-1.5 rounded-full border border-warning/20 px-2.5 text-xs'>
              <BarChart3 className='size-3.5' aria-hidden='true' />
              <span className='font-mono tabular-nums'>{watchCount}</span>
              <span>{t('watch')}</span>
            </span>
          )}
        </div>
      </div>

      <div className='grid gap-3 p-4 sm:p-5 md:grid-cols-2 xl:grid-cols-4'>
        {lanes.map((lane) => (
          <OperationsLaneCard key={lane.title} lane={lane} />
        ))}
      </div>
    </section>
  )
}
