import { createFileRoute } from '@tanstack/react-router'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { UserPlus } from 'lucide-react'

export const Route = createFileRoute('/_app/settings/members')({
  component: MembersPage,
})

function MembersPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-medium text-white">Team Members</h2>
          <p className="text-muted-foreground text-xs">Manage who has access</p>
        </div>
        <Button className="bg-blue-500 text-xs text-black hover:bg-blue-400">
          <UserPlus className="mr-1 h-3.5 w-3.5" />
          Invite Member
        </Button>
      </div>

      {/* Invite form */}
      <div className="border-border border bg-neutral-900/50 p-4">
        <div className="flex gap-2">
          <Input placeholder="Email address" className="bg-background flex-1 text-xs" />
          <Button variant="outline" className="text-xs">
            Send Invite
          </Button>
        </div>
      </div>

      {/* Members table */}
      <div className="border-border border bg-neutral-900/50">
        <div className="text-muted-foreground flex items-center justify-center py-8 text-xs">
          <div className="text-center">
            <p>No team members yet</p>
            <p className="mt-1 text-[10px]">Invite team members by email</p>
          </div>
        </div>
      </div>
    </div>
  )
}
