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
import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, RefreshCw, Save, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  deleteModelRegistry,
  deleteProviderRegistry,
  getModelRegistries,
  getProviderRegistries,
  saveModelRegistry,
  saveProviderRegistry,
} from '../api'
import type { ModelRegistry, ProviderRegistry } from '../types'

const PAGE_SIZE = 20

const emptyModelRegistry: Partial<ModelRegistry> = {
  external_model: '',
  provider: '',
  upstream_model: '',
  protocol: 'openai-compatible',
  capabilities: '',
  context_window: 0,
  max_output_tokens: 0,
  enabled: true,
  priority: 0,
}

const emptyProviderRegistry: Partial<ProviderRegistry> = {
  provider: '',
  protocol: 'openai-compatible',
  base_url: '',
  auth_type: 'bearer',
  enabled: true,
  health_status: 'healthy',
}

export function RegistryTable() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<'models' | 'providers'>('models')
  const [modelPage, setModelPage] = useState(1)
  const [providerPage, setProviderPage] = useState(1)
  const [modelFilter, setModelFilter] = useState('')
  const [providerFilter, setProviderFilter] = useState('')
  const [modelDraft, setModelDraft] =
    useState<Partial<ModelRegistry>>(emptyModelRegistry)
  const [providerDraft, setProviderDraft] = useState<Partial<ProviderRegistry>>(
    emptyProviderRegistry
  )

  const modelQueryKey = useMemo(
    () => ['model-registries', modelPage, modelFilter],
    [modelFilter, modelPage]
  )
  const providerQueryKey = useMemo(
    () => ['provider-registries', providerPage, providerFilter],
    [providerFilter, providerPage]
  )

  const modelQuery = useQuery({
    queryKey: modelQueryKey,
    queryFn: () =>
      getModelRegistries({
        p: modelPage,
        page_size: PAGE_SIZE,
        model: modelFilter || undefined,
      }),
  })

  const providerQuery = useQuery({
    queryKey: providerQueryKey,
    queryFn: () =>
      getProviderRegistries({
        p: providerPage,
        page_size: PAGE_SIZE,
        provider: providerFilter || undefined,
      }),
  })

  const saveModelMutation = useMutation({
    mutationFn: saveModelRegistry,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to save registry'))
        return
      }
      toast.success(t('Registry saved'))
      setModelDraft(emptyModelRegistry)
      await queryClient.invalidateQueries({ queryKey: ['model-registries'] })
    },
  })

  const saveProviderMutation = useMutation({
    mutationFn: saveProviderRegistry,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to save registry'))
        return
      }
      toast.success(t('Registry saved'))
      setProviderDraft(emptyProviderRegistry)
      await queryClient.invalidateQueries({ queryKey: ['provider-registries'] })
    },
  })

  const deleteModelMutation = useMutation({
    mutationFn: deleteModelRegistry,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to delete registry'))
        return
      }
      toast.success(t('Registry deleted'))
      await queryClient.invalidateQueries({ queryKey: ['model-registries'] })
    },
  })

  const deleteProviderMutation = useMutation({
    mutationFn: deleteProviderRegistry,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to delete registry'))
        return
      }
      toast.success(t('Registry deleted'))
      await queryClient.invalidateQueries({ queryKey: ['provider-registries'] })
    },
  })

  const modelItems = modelQuery.data?.data?.items ?? []
  const providerItems = providerQuery.data?.data?.items ?? []
  const modelTotal = modelQuery.data?.data?.total ?? 0
  const providerTotal = providerQuery.data?.data?.total ?? 0

  return (
    <div className='space-y-4'>
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)}>
        <TabsList className='max-w-full flex-wrap justify-start group-data-horizontal/tabs:h-auto'>
          <TabsTrigger value='models'>{t('Model Registry')}</TabsTrigger>
          <TabsTrigger value='providers'>{t('Provider Registry')}</TabsTrigger>
        </TabsList>
      </Tabs>

      {activeTab === 'models' ? (
        <div className='space-y-3'>
          <div className='grid gap-2 lg:grid-cols-[1.4fr_1fr_1fr_1fr_0.8fr_auto_auto]'>
            <Input
              placeholder={t('External model')}
              value={modelDraft.external_model ?? ''}
              onChange={(event) =>
                setModelDraft((draft) => ({
                  ...draft,
                  external_model: event.target.value,
                }))
              }
            />
            <Input
              placeholder={t('Provider')}
              value={modelDraft.provider ?? ''}
              onChange={(event) =>
                setModelDraft((draft) => ({ ...draft, provider: event.target.value }))
              }
            />
            <Input
              placeholder={t('Upstream model')}
              value={modelDraft.upstream_model ?? ''}
              onChange={(event) =>
                setModelDraft((draft) => ({
                  ...draft,
                  upstream_model: event.target.value,
                }))
              }
            />
            <Input
              placeholder={t('Protocol')}
              value={modelDraft.protocol ?? ''}
              onChange={(event) =>
                setModelDraft((draft) => ({ ...draft, protocol: event.target.value }))
              }
            />
            <Input
              placeholder={t('Priority')}
              type='number'
              value={modelDraft.priority ?? 0}
              onChange={(event) =>
                setModelDraft((draft) => ({
                  ...draft,
                  priority: Number(event.target.value),
                }))
              }
            />
            <div className='flex items-center gap-2 px-2'>
              <Switch
                checked={modelDraft.enabled ?? true}
                onCheckedChange={(checked) =>
                  setModelDraft((draft) => ({ ...draft, enabled: checked }))
                }
              />
              <span className='text-sm'>{t('Enabled')}</span>
            </div>
            <Button
              onClick={() => saveModelMutation.mutate(modelDraft)}
              disabled={saveModelMutation.isPending}
            >
              {modelDraft.id ? <Save className='h-4 w-4' /> : <Plus className='h-4 w-4' />}
              {modelDraft.id ? t('Save') : t('Add')}
            </Button>
          </div>
          <RegistryToolbar
            filter={modelFilter}
            setFilter={setModelFilter}
            onRefresh={() => void modelQuery.refetch()}
          />
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('External model')}</TableHead>
                <TableHead>{t('Provider')}</TableHead>
                <TableHead>{t('Upstream model')}</TableHead>
                <TableHead>{t('Protocol')}</TableHead>
                <TableHead>{t('Priority')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead className='w-32'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {modelItems.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className='font-medium'>{item.external_model}</TableCell>
                  <TableCell>{item.provider}</TableCell>
                  <TableCell>{item.upstream_model}</TableCell>
                  <TableCell>{item.protocol}</TableCell>
                  <TableCell>{item.priority}</TableCell>
                  <TableCell>
                    <StatusBadge enabled={item.enabled} />
                  </TableCell>
                  <TableCell>
                    <div className='flex gap-1'>
                      <Button variant='ghost' size='icon' onClick={() => setModelDraft(item)}>
                        <Save className='h-4 w-4' />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => deleteModelMutation.mutate(item.id)}
                      >
                        <Trash2 className='h-4 w-4' />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <RegistryPagination
            page={modelPage}
            setPage={setModelPage}
            total={modelTotal}
          />
        </div>
      ) : (
        <div className='space-y-3'>
          <div className='grid gap-2 lg:grid-cols-[1fr_1fr_1.5fr_0.8fr_0.8fr_auto_auto]'>
            <Input
              placeholder={t('Provider')}
              value={providerDraft.provider ?? ''}
              onChange={(event) =>
                setProviderDraft((draft) => ({
                  ...draft,
                  provider: event.target.value,
                }))
              }
            />
            <Input
              placeholder={t('Protocol')}
              value={providerDraft.protocol ?? ''}
              onChange={(event) =>
                setProviderDraft((draft) => ({
                  ...draft,
                  protocol: event.target.value,
                }))
              }
            />
            <Input
              placeholder={t('Base URL')}
              value={providerDraft.base_url ?? ''}
              onChange={(event) =>
                setProviderDraft((draft) => ({
                  ...draft,
                  base_url: event.target.value,
                }))
              }
            />
            <Input
              placeholder={t('Auth type')}
              value={providerDraft.auth_type ?? ''}
              onChange={(event) =>
                setProviderDraft((draft) => ({
                  ...draft,
                  auth_type: event.target.value,
                }))
              }
            />
            <Input
              placeholder={t('Health')}
              value={providerDraft.health_status ?? ''}
              onChange={(event) =>
                setProviderDraft((draft) => ({
                  ...draft,
                  health_status: event.target.value,
                }))
              }
            />
            <div className='flex items-center gap-2 px-2'>
              <Switch
                checked={providerDraft.enabled ?? true}
                onCheckedChange={(checked) =>
                  setProviderDraft((draft) => ({ ...draft, enabled: checked }))
                }
              />
              <span className='text-sm'>{t('Enabled')}</span>
            </div>
            <Button
              onClick={() => saveProviderMutation.mutate(providerDraft)}
              disabled={saveProviderMutation.isPending}
            >
              {providerDraft.id ? <Save className='h-4 w-4' /> : <Plus className='h-4 w-4' />}
              {providerDraft.id ? t('Save') : t('Add')}
            </Button>
          </div>
          <RegistryToolbar
            filter={providerFilter}
            setFilter={setProviderFilter}
            onRefresh={() => void providerQuery.refetch()}
          />
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Provider')}</TableHead>
                <TableHead>{t('Protocol')}</TableHead>
                <TableHead>{t('Base URL')}</TableHead>
                <TableHead>{t('Auth type')}</TableHead>
                <TableHead>{t('Health')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead className='w-32'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {providerItems.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className='font-medium'>{item.provider}</TableCell>
                  <TableCell>{item.protocol}</TableCell>
                  <TableCell className='max-w-80 truncate'>{item.base_url}</TableCell>
                  <TableCell>{item.auth_type}</TableCell>
                  <TableCell>{item.health_status}</TableCell>
                  <TableCell>
                    <StatusBadge enabled={item.enabled} />
                  </TableCell>
                  <TableCell>
                    <div className='flex gap-1'>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => setProviderDraft(item)}
                      >
                        <Save className='h-4 w-4' />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => deleteProviderMutation.mutate(item.id)}
                      >
                        <Trash2 className='h-4 w-4' />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <RegistryPagination
            page={providerPage}
            setPage={setProviderPage}
            total={providerTotal}
          />
        </div>
      )}
    </div>
  )
}

function RegistryToolbar({
  filter,
  setFilter,
  onRefresh,
}: {
  filter: string
  setFilter: (value: string) => void
  onRefresh: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
      <Input
        className='max-w-sm'
        placeholder={t('Search...')}
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
      />
      <Button variant='outline' onClick={onRefresh}>
        <RefreshCw className='h-4 w-4' />
        {t('Refresh')}
      </Button>
    </div>
  )
}

function RegistryPagination({
  page,
  setPage,
  total,
}: {
  page: number
  setPage: (page: number) => void
  total: number
}) {
  const { t } = useTranslation()
  const maxPage = Math.max(1, Math.ceil(total / PAGE_SIZE))
  return (
    <div className='flex items-center justify-end gap-2'>
      <span className='text-muted-foreground text-sm'>
        {t('Page')} {page} / {maxPage}
      </span>
      <Button
        variant='outline'
        disabled={page <= 1}
        onClick={() => setPage(page - 1)}
      >
        {t('Previous')}
      </Button>
      <Button
        variant='outline'
        disabled={page >= maxPage}
        onClick={() => setPage(page + 1)}
      >
        {t('Next')}
      </Button>
    </div>
  )
}

function StatusBadge({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
  return (
    <Badge variant={enabled ? 'default' : 'secondary'}>
      {enabled ? t('Enabled') : t('Disabled')}
    </Badge>
  )
}
