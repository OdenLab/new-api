import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { getConversationLogs } from '../api'

type ConversationLogItem = {
  id: number
  created_at: number
  username: string
  token_name: string
  model_name: string
  prompt_text: string
  reply_text: string
}

function formatLogTime(createdAt: number): string {
  if (!createdAt) {
    return '-'
  }
  return dayjs.unix(createdAt).format('YYYY-MM-DD HH:mm:ss')
}

export function ConversationLogsPanel() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [username, setUsername] = useState('')
  const [tokenName, setTokenName] = useState('')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')

  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: [
      'conversation-logs',
      page,
      username,
      tokenName,
      fromDate,
      toDate,
    ],
    queryFn: () =>
      getConversationLogs({
        p: page,
        page_size: 20,
        username,
        token_name: tokenName,
        start_time: fromDate
          ? dayjs(fromDate).startOf('day').unix()
          : undefined,
        end_time: toDate ? dayjs(toDate).endOf('day').unix() : undefined,
      }),
  })

  const items = useMemo<ConversationLogItem[]>(
    () => data?.data?.items ?? [],
    [data]
  )
  const total = data?.data?.total ?? 0

  const groupedItems = useMemo(() => {
    const groups = new Map<string, Map<string, ConversationLogItem[]>>()

    for (const item of items) {
      const user = item.username || t('Unknown user')
      const keyName = item.token_name || t('Unknown key')
      if (!groups.has(user)) {
        groups.set(user, new Map<string, ConversationLogItem[]>())
      }
      const userGroup = groups.get(user)
      if (!userGroup) {
        continue
      }
      if (!userGroup.has(keyName)) {
        userGroup.set(keyName, [])
      }
      const tokenGroup = userGroup.get(keyName)
      if (!tokenGroup) {
        continue
      }
      tokenGroup.push(item)
    }

    return [...groups.entries()].map(([user, tokens]) => ({
      user,
      tokens: [...tokens.entries()].map(([token, logs]) => ({
        token,
        logs,
      })),
    }))
  }, [items, t])

  return (
    <div className='space-y-4'>
      <div className='bg-card rounded-xl border p-4 shadow-sm'>
        <div className='mb-3 text-sm font-medium'>{t('Filter logs')}</div>
        <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
          <Input
            placeholder={t('Filter by username')}
            value={username}
            onChange={(event) => {
              setUsername(event.target.value)
              setPage(1)
            }}
          />
          <Input
            placeholder={t('Filter by key name')}
            value={tokenName}
            onChange={(event) => {
              setTokenName(event.target.value)
              setPage(1)
            }}
          />
          <Input
            type='date'
            value={fromDate}
            onChange={(event) => {
              setFromDate(event.target.value)
              setPage(1)
            }}
          />
          <Input
            type='date'
            value={toDate}
            onChange={(event) => {
              setToDate(event.target.value)
              setPage(1)
            }}
          />
        </div>
        <div className='mt-3 flex flex-wrap gap-2'>
          <Button variant='secondary' onClick={() => void refetch()}>
            {isFetching ? t('Loading...') : t('Refresh')}
          </Button>
          <Button
            variant='outline'
            onClick={() => {
              setUsername('')
              setTokenName('')
              setFromDate('')
              setToDate('')
              setPage(1)
            }}
          >
            {t('Reset filters')}
          </Button>
        </div>
        <div className='text-muted-foreground mt-3 text-sm'>
          {t('Total records')}: {total} · {t('Showing current page')}:{' '}
          {items.length}
        </div>
      </div>

      {isLoading && (
        <div className='bg-card/80 rounded-2xl border p-6 text-sm shadow-sm backdrop-blur-sm'>
          {t('Loading...')}
        </div>
      )}
      {!isLoading && groupedItems.length === 0 && (
        <div className='bg-card/80 text-muted-foreground rounded-2xl border p-6 text-sm shadow-sm backdrop-blur-sm'>
          {t('No conversation logs found for current filters.')}
        </div>
      )}
      {!isLoading &&
        groupedItems.length > 0 &&
        groupedItems.map((userGroup) => (
          <div
            key={userGroup.user}
            className='bg-card/90 rounded-2xl border p-4 shadow-sm backdrop-blur-sm'
          >
            <div className='mb-3 text-sm font-semibold tracking-wide'>
              {t('User')}: {userGroup.user}
            </div>
            <div className='space-y-3'>
              {userGroup.tokens.map((tokenGroup) => (
                <div
                  key={`${userGroup.user}-${tokenGroup.token}`}
                  className='bg-background/70 rounded-xl border p-3'
                >
                  <div className='text-muted-foreground mb-2 text-xs font-medium'>
                    {t('Key')}: {tokenGroup.token} ({tokenGroup.logs.length}{' '}
                    {t('records')})
                  </div>
                  <div className='space-y-2'>
                    {tokenGroup.logs.map((item) => (
                      <details
                        key={item.id}
                        className='bg-card rounded-lg border p-2.5 shadow-xs'
                      >
                        <summary className='text-muted-foreground cursor-pointer text-xs'>
                          #{item.id} · {formatLogTime(item.created_at)} ·{' '}
                          {item.model_name || '-'}
                        </summary>
                        <div className='mt-2 grid gap-3 md:grid-cols-2'>
                          <div>
                            <div className='mb-1 text-xs font-medium'>
                              {t('Prompt')}
                            </div>
                            <pre className='bg-muted max-h-52 overflow-auto rounded-md p-2 text-xs whitespace-pre-wrap'>
                              {item.prompt_text}
                            </pre>
                          </div>
                          <div>
                            <div className='mb-1 text-xs font-medium'>
                              {t('Response')}
                            </div>
                            <pre className='bg-muted max-h-52 overflow-auto rounded-md p-2 text-xs whitespace-pre-wrap'>
                              {item.reply_text}
                            </pre>
                          </div>
                        </div>
                      </details>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}

      <div className='flex flex-wrap gap-2'>
        <Button
          variant='outline'
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          disabled={page === 1}
        >
          {t('Previous')}
        </Button>
        <Button
          variant='outline'
          onClick={() => setPage((current) => current + 1)}
          disabled={page * 20 >= total}
        >
          {t('Next')}
        </Button>
        <div className='text-muted-foreground self-center text-xs'>
          {t('Page')}: {page}
        </div>
      </div>
    </div>
  )
}
