import { createFileRoute } from '@tanstack/react-router'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'

export const Route = createFileRoute('/_app/settings/organization')({
  component: OrgSettingsPage,
})

function OrgSettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-white">Organization Settings</h2>
        <p className="text-muted-foreground text-xs">Manage your organization</p>
      </div>

      <div className="border-border max-w-lg border bg-neutral-900/50 p-6">
        <div className="space-y-4">
          <div className="space-y-1">
            <Label className="text-xs">Organization Name</Label>
            <Input className="bg-background text-xs" placeholder="My Organization" />
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Slug</Label>
            <Input className="bg-background text-xs" placeholder="my-org" disabled />
            <p className="text-muted-foreground text-[10px]">
              Slug cannot be changed after creation
            </p>
          </div>

          <Button className="bg-blue-500 text-xs text-black hover:bg-blue-400">Save Changes</Button>
        </div>
      </div>

      <div className="border-border max-w-lg border border-red-500/30 bg-red-500/5 p-6">
        <h3 className="text-xs font-medium text-red-400">Danger Zone</h3>
        <p className="text-muted-foreground mt-1 text-[10px]">
          Deleting an organization is permanent and cannot be undone.
        </p>
        <Button variant="outline" className="mt-3 border-red-500/30 text-xs text-red-400">
          Delete Organization
        </Button>
      </div>
    </div>
  )
}
