import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useAuth } from '~/lib/auth'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'

export const Route = createFileRoute('/_app/settings/')({
  component: ProfileSettingsPage,
})

function ProfileSettingsPage() {
  const { user } = useAuth()
  const [displayName, setDisplayName] = useState(user?.display_name || '')

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-white">Profile Settings</h2>
        <p className="text-muted-foreground text-xs">Manage your account information</p>
      </div>

      <div className="border-border max-w-lg border bg-neutral-900/50 p-6">
        <div className="space-y-4">
          <div className="space-y-1">
            <Label className="text-xs">Display Name</Label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="bg-background text-xs"
            />
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Email</Label>
            <Input
              value={user?.email || ''}
              disabled
              className="bg-background text-xs opacity-60"
            />
            <p className="text-muted-foreground text-[10px]">Email cannot be changed</p>
          </div>

          <Button className="bg-blue-500 text-xs text-black hover:bg-blue-400">Save Changes</Button>
        </div>
      </div>
    </div>
  )
}
