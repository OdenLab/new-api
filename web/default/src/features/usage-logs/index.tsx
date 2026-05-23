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
import { useCallback, useMemo } from 'react'
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useSidebarConfig } from '@/hooks/use-sidebar-config'
import { useIsAdmin } from '@/hooks/use-admin'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SectionPageLayout } from '@/components/layout'
import type { NavGroup } from '@/components/layout/types'
import { CacheStatsDialog } from '@/features/system-settings/general/channel-affinity/cache-stats-dialog'
import { UserInfoDialog } from './components/dialogs/user-info-dialog'
import {
  UsageLogsProvider,
  useUsageLogsContext,
} from './components/usage-logs-provider'
import { UsageLogsTable } from './components/usage-logs-table'
import { ConversationLogsPanel } from './components/conversation-logs-panel'
import {
  isUsageLogsSectionId,
  USAGE_LOGS_DEFAULT_SECTION,
  type UsageLogsSectionId,
} from './section-registry'

const route = getRouteApi('/_authenticated/usage-logs/$section')
const TASK_LOG_SECTIONS = ['drawing', 'task'] as const

const SECTION_META: Record<
  UsageLogsSectionId,
  { titleKey: string; descriptionKey: string }
> = {
  common: {
    titleKey: 'Common Logs',
    descriptionKey: 'View and manage your API usage logs',
  },
  conversation: {
    titleKey: 'Conversation Logs',
    descriptionKey: 'View prompt and response history by user and key',
  },
  drawing: {
    titleKey: 'Drawing Logs',
    descriptionKey: 'View and manage your drawing logs',
  },
  task: {
    titleKey: 'Task Logs',
    descriptionKey: 'View and manage your task logs',
  },
}

function UsageLogsContent() {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const navigate = useNavigate()
  const params = route.useParams()
  const activeCategory: UsageLogsSectionId =
    params.section && isUsageLogsSectionId(params.section)
      ? params.section
      : USAGE_LOGS_DEFAULT_SECTION
  const {
    selectedUserId,
    userInfoDialogOpen,
    setUserInfoDialogOpen,
    affinityTarget,
    affinityDialogOpen,
    setAffinityDialogOpen,
  } = useUsageLogsContext()
  const tabNavGroups = useMemo<NavGroup[]>(
    () => [
      {
        title: 'Usage Logs',
        items: [
          {
            title: SECTION_META.common.titleKey,
            url: '/usage-logs/common',
          },
          ...(isAdmin
            ? [
                {
                  title: SECTION_META.conversation.titleKey,
                  url: '/usage-logs/conversation',
                },
              ]
            : []),
        ],
      },
      {
        title: 'Task Logs',
        items: TASK_LOG_SECTIONS.map((section) => ({
          title: SECTION_META[section].titleKey,
          url: `/usage-logs/${section}`,
        })),
      },
    ],
    [isAdmin]
  )
  const filteredTabGroups = useSidebarConfig(tabNavGroups)
  const visibleSections = useMemo(
    () =>
      (filteredTabGroups[0]?.items ?? [])
        .map((item) => {
          if (!('url' in item) || typeof item.url !== 'string') return null
          return item.url.split('/').pop() ?? null
        })
        .filter((section): section is UsageLogsSectionId =>
          Boolean(section && isUsageLogsSectionId(section))
        ),
    [filteredTabGroups]
  )

  const handleSectionChange = useCallback(
    (section: string) => {
      void navigate({
        to: '/usage-logs/$section',
        params: { section: section as UsageLogsSectionId },
      })
    },
    [navigate]
  )

  const pageMeta = SECTION_META[activeCategory] ?? SECTION_META.common
  const showSectionSwitcher = visibleSections.length > 1

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t(pageMeta.titleKey)}
        </SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t(pageMeta.descriptionKey)}
        </SectionPageLayout.Description>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <div className='from-background/70 to-background/55 rounded-2xl border bg-linear-to-br p-3 shadow-sm backdrop-blur-md'>
              <div className='text-muted-foreground mb-2 text-xs font-medium tracking-wide'>
                {t('Log workspace')}
              </div>
            {showSectionSwitcher && (
              <Tabs value={activeCategory} onValueChange={handleSectionChange}>
                <TabsList className='bg-muted/60 group-data-horizontal/tabs:h-auto max-w-full flex-wrap justify-start rounded-xl border p-1 shadow-sm backdrop-blur-sm'>
                  {visibleSections.map((section) => (
                    <TabsTrigger
                      key={section}
                      value={section}
                      className='data-[state=active]:shadow-xs rounded-lg'
                    >
                      {t(SECTION_META[section].titleKey)}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            )}
            </div>
            {activeCategory === 'conversation' ? (
              <ConversationLogsPanel />
            ) : (
              <UsageLogsTable logCategory={activeCategory} />
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <UserInfoDialog
        userId={selectedUserId}
        open={userInfoDialogOpen}
        onOpenChange={setUserInfoDialogOpen}
      />

      <CacheStatsDialog
        open={affinityDialogOpen}
        onOpenChange={setAffinityDialogOpen}
        target={
          affinityTarget
            ? {
                rule_name: affinityTarget.rule_name || '',
                using_group:
                  affinityTarget.using_group ||
                  affinityTarget.selected_group ||
                  '',
                key_hint: affinityTarget.key_hint || '',
                key_fp: affinityTarget.key_fp || '',
              }
            : null
        }
      />
    </>
  )
}

export function UsageLogs() {
  return (
    <UsageLogsProvider>
      <UsageLogsContent />
    </UsageLogsProvider>
  )
}
